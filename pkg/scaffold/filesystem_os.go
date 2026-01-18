package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// OSFileSystem implements FileSystem using the operating system
type OSFileSystem struct {
	// basePath restricts operations to within this directory for safety
	basePath string
	mu       sync.RWMutex
}

// NewOSFileSystem creates a new OS filesystem with optional base path restriction
func NewOSFileSystem(basePath string) (*OSFileSystem, error) {
	if basePath != "" {
		// Ensure base path exists and is absolute
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path (basePath=%s): %w", basePath, err)
		}
		basePath = absPath
	}

	return &OSFileSystem{
		basePath: basePath,
	}, nil
}

// ReadFile reads a file from the filesystem
func (osfs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return nil, ErrFileRead(path, err)
	}

	return data, nil
}

// WriteFile writes content to a file with non-destructive safety checks
func (osfs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}

	// Check if file already exists
	if osfs.Exists(path) {
		return ErrFileExists(path)
	}

	// Create directory if needed
	// Note: we need to get the directory relative to the base path
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := osfs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory (dir=%s, file=%s): %w", dir, path, err)
		}
	}

	// Write file atomically using temporary file
	return osfs.writeFileAtomic(safePath, data, perm)
}

// writeFileAtomic writes a file atomically using a temporary file
func (osfs *OSFileSystem) writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	// Create temporary file in the same directory
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, ".tmp-scaffold-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file (directory=%s): %w", dir, err)
	}

	tmpPath := tmpFile.Name()

	// Cleanup function
	cleanup := func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}

	// Write data to temporary file
	if _, err := tmpFile.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("failed to write to temporary file (tempFile=%s): %w", tmpPath, err)
	}

	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("failed to sync temporary file (tempFile=%s): %w", tmpPath, err)
	}

	// Close temporary file
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temporary file (tempFile=%s): %w", tmpPath, err)
	}

	// Set permissions
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to set file permissions (tempFile=%s, permissions=%v): %w", tmpPath, perm, err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return ErrFileWrite(path, err)
	}

	return nil
}

// Stat returns file info
func (osfs *OSFileSystem) Stat(path string) (os.FileInfo, error) {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file (path=%s): %w", path, err)
	}

	return info, nil
}

// MkdirAll creates directories
func (osfs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	// Special case: if path is "." or empty and we have a basePath, create the basePath
	if (path == "." || path == "") && osfs.basePath != "" {
		if err := os.MkdirAll(osfs.basePath, perm); err != nil {
			return fmt.Errorf("failed to create base directory (path=%s, permissions=%v): %w", osfs.basePath, perm, err)
		}
		return nil
	}

	safePath, err := osfs.safePath(path)
	if err != nil {
		// Add more details to understand what's failing
		return fmt.Errorf("failed to validate path for MkdirAll (input_path=%s, base_path=%s): %w", path, osfs.basePath, err)
	}

	if err := os.MkdirAll(safePath, perm); err != nil {
		return fmt.Errorf("failed to create directories (path=%s, permissions=%v): %w", path, perm, err)
	}

	return nil
}

// Exists checks if a path exists
func (osfs *OSFileSystem) Exists(path string) bool {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return false
	}

	_, err = os.Stat(safePath)
	return err == nil
}

// IsDir checks if a path is a directory
func (osfs *OSFileSystem) IsDir(path string) bool {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return false
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// Remove removes a file or directory
func (osfs *OSFileSystem) Remove(path string) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}

	if err := os.Remove(safePath); err != nil {
		return fmt.Errorf("failed to remove file (path=%s): %w", path, err)
	}

	return nil
}

// RemoveAll removes a directory and all its contents
func (osfs *OSFileSystem) RemoveAll(path string) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(safePath); err != nil {
		return fmt.Errorf("failed to remove directory (path=%s): %w", path, err)
	}

	return nil
}

// safePath ensures the path is safe and within the base path if restricted
func (osfs *OSFileSystem) safePath(path string) (string, error) {
	osfs.mu.RLock()
	basePath := osfs.basePath
	osfs.mu.RUnlock()

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts after cleaning
	// This properly checks if the cleaned path tries to escape
	if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, string(filepath.Separator)+"..") {
		return "", ErrInvalidPath(path, "path traversal not allowed")
	}

	// If no base path restriction
	if basePath == "" {
		// Allow absolute paths when no base restriction
		if filepath.IsAbs(cleanPath) {
			return cleanPath, nil
		}
		// For relative paths, resolve to absolute
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path (path=%s): %w", path, err)
		}
		return absPath, nil
	}

	// With base path restriction, reject absolute paths
	if filepath.IsAbs(cleanPath) {
		return "", ErrInvalidPath(path, "absolute paths not allowed when base path is set")
	}

	// Join with base path (basePath is already absolute from NewOSFileSystem)
	fullPath := filepath.Join(basePath, cleanPath)

	// Get the real absolute path (handles any symlinks)
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve full path (path=%s): %w", path, err)
	}

	// Check if the resolved path is within the base path
	// basePath is already absolute, so no need to call Abs again

	// Ensure base path ends with separator for consistent comparison
	baseWithSep := basePath
	if !strings.HasSuffix(baseWithSep, string(filepath.Separator)) {
		baseWithSep = basePath + string(filepath.Separator)
	}

	// Path is valid if it's the exact base or starts with base+separator
	if absFullPath != basePath && !strings.HasPrefix(absFullPath, baseWithSep) {
		return "", fmt.Errorf("path escapes base directory: path=%s, cleanPath=%s, absFullPath=%s, basePath=%s", path, cleanPath, absFullPath, basePath)
	}

	return absFullPath, nil
}

// Symlink creates a symbolic link
func (osfs *OSFileSystem) Symlink(oldname, newname string) error {
	safeOld, err := osfs.safePath(oldname)
	if err != nil {
		return err
	}

	safeNew, err := osfs.safePath(newname)
	if err != nil {
		return err
	}

	if err := os.Symlink(safeOld, safeNew); err != nil {
		return fmt.Errorf("failed to create symlink (oldname=%s, newname=%s): %w", oldname, newname, err)
	}

	return nil
}
