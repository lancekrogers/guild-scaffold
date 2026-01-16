package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

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
		return fmt.Errorf("context cancelled before writing file: %w", err)
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
