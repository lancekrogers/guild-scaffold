package scaffold

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MemoryFileSystem implements FileSystem in memory for testing
type MemoryFileSystem struct {
	files    map[string][]byte
	dirs     map[string]bool
	symlinks map[string]string // maps link path to target path
	mu       sync.RWMutex
}

// NewMemoryFileSystem creates a new in-memory filesystem
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files:    make(map[string][]byte),
		dirs:     make(map[string]bool),
		symlinks: make(map[string]string),
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

	// Check symlinks
	if _, exists := mfs.symlinks[cleanPath]; exists {
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
				return fmt.Errorf("directory not empty: path=%v", path)
			}
		}

		for dirPath := range mfs.dirs {
			if dirPath != cleanPath && isWithinDir(dirPath, cleanPath) {
				return fmt.Errorf("directory not empty: path=%v", path)
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

// memoryFileInfo implements os.FileInfo for memory filesystem
type memoryFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (mfi *memoryFileInfo) Name() string { return mfi.name }
func (mfi *memoryFileInfo) Size() int64  { return mfi.size }
func (mfi *memoryFileInfo) Mode() os.FileMode {
	if mfi.isDir {
		return 0755 | os.ModeDir
	}
	return 0644
}
func (mfi *memoryFileInfo) ModTime() time.Time { return time.Now() }
func (mfi *memoryFileInfo) IsDir() bool        { return mfi.isDir }
func (mfi *memoryFileInfo) Sys() interface{}   { return nil }

// Symlink creates a symbolic link in memory
func (mfs *MemoryFileSystem) Symlink(target, linkPath string) error {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()

	cleanLinkPath := filepath.Clean(linkPath)

	// Check if link already exists
	if _, exists := mfs.files[cleanLinkPath]; exists {
		return ErrFileExists(linkPath)
	}
	if _, exists := mfs.symlinks[cleanLinkPath]; exists {
		return ErrFileExists(linkPath)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(cleanLinkPath)
	if dir != "." && dir != "/" {
		mfs.dirs[dir] = true
	}

	// Store the symlink
	mfs.symlinks[cleanLinkPath] = target

	return nil
}

// ReadLink reads the target of a symbolic link
func (mfs *MemoryFileSystem) ReadLink(path string) (string, error) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	cleanPath := filepath.Clean(path)

	target, exists := mfs.symlinks[cleanPath]
	if !exists {
		return "", fmt.Errorf("not a symlink: path=%v", path)
	}

	return target, nil
}

// IsSymlink checks if a path is a symbolic link
func (mfs *MemoryFileSystem) IsSymlink(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()

	cleanPath := filepath.Clean(path)
	_, exists := mfs.symlinks[cleanPath]
	return exists
}
