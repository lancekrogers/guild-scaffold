package scaffold

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// FileSystem abstracts filesystem operations for testing and safety
type FileSystem interface {
	// ReadFile reads a file from the filesystem
	ReadFile(path string) ([]byte, error)
	
	// WriteFile writes content to a file
	WriteFile(path string, data []byte, perm os.FileMode) error
	
	// Stat returns file info
	Stat(path string) (os.FileInfo, error)
	
	// MkdirAll creates directories
	MkdirAll(path string, perm os.FileMode) error
	
	// Exists checks if a path exists
	Exists(path string) bool
	
	// IsDir checks if a path is a directory
	IsDir(path string) bool
	
	// Remove removes a file or directory
	Remove(path string) error
	
	// RemoveAll removes a directory and all its contents
	RemoveAll(path string) error
}

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
			return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get absolute path").
				WithDetails("basePath", basePath)
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
	if osfs.Exists(safePath) {
		return ErrFileExists(path)
	}
	
	// Create directory if needed
	dir := filepath.Dir(safePath)
	if err := osfs.MkdirAll(dir, 0755); err != nil {
		return err
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
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create temporary file").
			WithDetails("directory", dir)
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
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to write to temporary file").
			WithDetails("tempFile", tmpPath)
	}
	
	// Sync to disk
	if err := tmpFile.Sync(); err != nil {
		cleanup()
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to sync temporary file").
			WithDetails("tempFile", tmpPath)
	}
	
	// Close temporary file
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to close temporary file").
			WithDetails("tempFile", tmpPath)
	}
	
	// Set permissions
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath)
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to set file permissions").
			WithDetails("tempFile", tmpPath).
			WithDetails("permissions", perm)
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
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to stat file").
			WithDetails("path", path)
	}
	
	return info, nil
}

// MkdirAll creates directories
func (osfs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	safePath, err := osfs.safePath(path)
	if err != nil {
		return err
	}
	
	if err := os.MkdirAll(safePath, perm); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directories").
			WithDetails("path", path).
			WithDetails("permissions", perm)
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
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove file").
			WithDetails("path", path)
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
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to remove directory").
			WithDetails("path", path)
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
	
	// Check for path traversal attempts
	if filepath.IsAbs(cleanPath) {
		return "", ErrInvalidPath(path, "absolute paths not allowed")
	}
	
	if containsPathTraversal(cleanPath) {
		return "", ErrInvalidPath(path, "path traversal not allowed")
	}
	
	// If no base path restriction, return cleaned path
	if basePath == "" {
		return cleanPath, nil
	}
	
	// Join with base path
	fullPath := filepath.Join(basePath, cleanPath)
	
	// Ensure the resolved path is still within base path
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve absolute path").
			WithDetails("path", path)
	}
	
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeInternal, "failed to resolve base path").
			WithDetails("basePath", basePath)
	}
	
	// Check if the resolved path is within the base path
	rel, err := filepath.Rel(absBasePath, absFullPath)
	if err != nil || filepath.IsAbs(rel) || containsPathTraversal(rel) {
		return "", ErrInvalidPath(path, "path escapes base directory")
	}
	
	return absFullPath, nil
}

// containsPathTraversal checks for path traversal patterns
func containsPathTraversal(path string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)
	
	// Check for various path traversal patterns
	patterns := []string{
		"../",
		"..\\",
		"/..",
		"\\..",
	}
	
	for _, pattern := range patterns {
		if fmt.Sprintf("%s", normalizedPath) != normalizedPath {
			continue
		}
		if containsPattern(normalizedPath, pattern) {
			return true
		}
	}
	
	return false
}

// containsPattern checks if a string contains a specific pattern
func containsPattern(s, pattern string) bool {
	return len(s) >= len(pattern) && 
		(s == pattern || 
		 s[:len(pattern)] == pattern || 
		 s[len(s)-len(pattern):] == pattern ||
		 containsSubstring(s, pattern))
}

// containsSubstring is a simple substring check
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MemoryFileSystem implements FileSystem in memory for testing
type MemoryFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
	mu    sync.RWMutex
}

// NewMemoryFileSystem creates a new in-memory filesystem
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// ReadFile reads a file from memory
func (mfs *MemoryFileSystem) ReadFile(path string) ([]byte, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	
	cleanPath := filepath.Clean(path)
	data, exists := mfs.files[cleanPath]
	if !exists {
		return nil, ErrFileRead(path, fs.ErrNotExist)
	}
	
	// Return copy to prevent modification
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}

// WriteFile writes content to memory
func (mfs *MemoryFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	
	cleanPath := filepath.Clean(path)
	
	// Check if file already exists
	if _, exists := mfs.files[cleanPath]; exists {
		return ErrFileExists(path)
	}
	
	// Ensure directory exists
	dir := filepath.Dir(cleanPath)
	if dir != "." && dir != "/" {
		mfs.dirs[dir] = true
	}
	
	// Store copy to prevent external modification
	fileCopy := make([]byte, len(data))
	copy(fileCopy, data)
	mfs.files[cleanPath] = fileCopy
	
	return nil
}

// Stat returns file info for memory filesystem
func (mfs *MemoryFileSystem) Stat(path string) (os.FileInfo, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	
	cleanPath := filepath.Clean(path)
	
	// Check if it's a file
	if data, exists := mfs.files[cleanPath]; exists {
		return &memoryFileInfo{
			name:  filepath.Base(cleanPath),
			size:  int64(len(data)),
			isDir: false,
		}, nil
	}
	
	// Check if it's a directory
	if _, exists := mfs.dirs[cleanPath]; exists {
		return &memoryFileInfo{
			name:  filepath.Base(cleanPath),
			size:  0,
			isDir: true,
		}, nil
	}
	
	return nil, fs.ErrNotExist
}

// MkdirAll creates directories in memory
func (mfs *MemoryFileSystem) MkdirAll(path string, perm os.FileMode) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "/" {
		return nil
	}
	
	// Create all parent directories
	parts := strings.Split(cleanPath, string(filepath.Separator))
	currentPath := ""
	
	for _, part := range parts {
		if part == "" {
			continue
		}
		
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = filepath.Join(currentPath, part)
		}
		
		mfs.dirs[currentPath] = true
	}
	
	return nil
}

// Exists checks if a path exists in memory
func (mfs *MemoryFileSystem) Exists(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	
	cleanPath := filepath.Clean(path)
	
	// Check files
	if _, exists := mfs.files[cleanPath]; exists {
		return true
	}
	
	// Check directories
	if _, exists := mfs.dirs[cleanPath]; exists {
		return true
	}
	
	return false
}

// IsDir checks if a path is a directory in memory
func (mfs *MemoryFileSystem) IsDir(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	
	cleanPath := filepath.Clean(path)
	_, isDir := mfs.dirs[cleanPath]
	return isDir
}

// Remove removes a file or directory from memory
func (mfs *MemoryFileSystem) Remove(path string) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	
	cleanPath := filepath.Clean(path)
	
	// Try to remove file
	if _, exists := mfs.files[cleanPath]; exists {
		delete(mfs.files, cleanPath)
		return nil
	}
	
	// Try to remove directory (only if empty)
	if _, exists := mfs.dirs[cleanPath]; exists {
		// Check if directory has contents
		for filePath := range mfs.files {
			if isWithinDir(filePath, cleanPath) {
				return gerror.New(gerror.ErrCodeIO, "directory not empty", nil).
					WithDetails("path", path)
			}
		}
		
		for dirPath := range mfs.dirs {
			if dirPath != cleanPath && isWithinDir(dirPath, cleanPath) {
				return gerror.New(gerror.ErrCodeIO, "directory not empty", nil).
					WithDetails("path", path)
			}
		}
		
		delete(mfs.dirs, cleanPath)
		return nil
	}
	
	return fs.ErrNotExist
}

// RemoveAll removes a directory and all its contents from memory
func (mfs *MemoryFileSystem) RemoveAll(path string) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	
	cleanPath := filepath.Clean(path)
	
	// Remove all files within the directory
	for filePath := range mfs.files {
		if isWithinDir(filePath, cleanPath) || filePath == cleanPath {
			delete(mfs.files, filePath)
		}
	}
	
	// Remove all subdirectories
	for dirPath := range mfs.dirs {
		if isWithinDir(dirPath, cleanPath) || dirPath == cleanPath {
			delete(mfs.dirs, dirPath)
		}
	}
	
	return nil
}

// isWithinDir checks if a path is within a directory
func isWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && !containsPathTraversal(rel)
}

// memoryFileInfo implements os.FileInfo for memory filesystem
type memoryFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (mfi *memoryFileInfo) Name() string       { return mfi.name }
func (mfi *memoryFileInfo) Size() int64        { return mfi.size }
func (mfi *memoryFileInfo) Mode() os.FileMode  { 
	if mfi.isDir {
		return 0755 | os.ModeDir
	}
	return 0644 
}
func (mfi *memoryFileInfo) ModTime() time.Time { return time.Now() }
func (mfi *memoryFileInfo) IsDir() bool        { return mfi.isDir }
func (mfi *memoryFileInfo) Sys() interface{}   { return nil }

// SafeFileWriter provides additional safety for file operations
type SafeFileWriter struct {
	fs           FileSystem
	dryRun       bool
	allowedPaths []string
}

// NewSafeFileWriter creates a new safe file writer
func NewSafeFileWriter(fs FileSystem, dryRun bool, allowedPaths []string) *SafeFileWriter {
	return &SafeFileWriter{
		fs:           fs,
		dryRun:       dryRun,
		allowedPaths: allowedPaths,
	}
}

// WriteFileWithContext writes a file with context cancellation support
func (sfw *SafeFileWriter) WriteFileWithContext(ctx context.Context, path string, data []byte, perm os.FileMode) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled before writing file")
	}
	
	// Validate path is allowed
	if !sfw.isPathAllowed(path) {
		return ErrInvalidPath(path, "path not in allowed list")
	}
	
	// In dry run mode, just validate without writing
	if sfw.dryRun {
		return sfw.validateWrite(path, data, perm)
	}
	
	// Perform actual write
	return sfw.fs.WriteFile(path, data, perm)
}

// isPathAllowed checks if a path is in the allowed paths list
func (sfw *SafeFileWriter) isPathAllowed(path string) bool {
	if len(sfw.allowedPaths) == 0 {
		return true // No restrictions
	}
	
	cleanPath := filepath.Clean(path)
	
	for _, allowedPath := range sfw.allowedPaths {
		allowedClean := filepath.Clean(allowedPath)
		
		// Check exact match or within allowed directory
		if cleanPath == allowedClean || isWithinDir(cleanPath, allowedClean) {
			return true
		}
	}
	
	return false
}

// validateWrite performs validation without actually writing
func (sfw *SafeFileWriter) validateWrite(path string, data []byte, perm os.FileMode) error {
	// Check if file would already exist
	if sfw.fs.Exists(path) {
		return ErrFileExists(path)
	}
	
	// Validate directory can be created
	dir := filepath.Dir(path)
	if dir != "." && !sfw.fs.Exists(dir) {
		// This would need to create directories - validate that's allowed
		if !sfw.isPathAllowed(dir) {
			return ErrInvalidPath(dir, "directory creation not allowed")
		}
	}
	
	return nil
}