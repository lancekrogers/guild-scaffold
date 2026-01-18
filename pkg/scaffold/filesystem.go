package scaffold

import (
	"os"
	"path/filepath"
	"strings"
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

	// Symlink creates a symbolic link
	Symlink(oldname, newname string) error
}

// containsPathTraversal checks for path traversal patterns
func containsPathTraversal(path string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)

	// Check for various path traversal patterns
	if normalizedPath == ".." {
		return true
	}

	patterns := []string{
		"../",
		"..\\",
		"/..",
		"\\..",
	}

	for _, pattern := range patterns {
		if strings.Contains(normalizedPath, pattern) {
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

// isWithinDir checks if a path is within a directory
func isWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && !containsPathTraversal(rel)
}
