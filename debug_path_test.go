package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// safePath simulation for debugging
func debugSafePath(basePath, path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts - simple and effective
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}

	// If no base path restriction
	if basePath == "" {
		// Allow absolute paths when no base restriction
		if filepath.IsAbs(cleanPath) {
			return cleanPath, nil
		}
		return cleanPath, nil
	}

	// With base path restriction, reject absolute paths
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute paths not allowed when base path is set")
	}

	// Join with base path
	fullPath := filepath.Join(basePath, cleanPath)

	// Ensure the resolved path is still within base path
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}

	// Check if the resolved path is within the base path
	// Need to handle both exact match and subdirectory cases
	normalizedBase := strings.TrimSuffix(absBasePath, string(filepath.Separator))
	normalizedFull := strings.TrimSuffix(absFullPath, string(filepath.Separator))

	fmt.Printf("DEBUG: path=%s, cleanPath=%s, fullPath=%s\n", path, cleanPath, fullPath)
	fmt.Printf("DEBUG: absBasePath=%s, absFullPath=%s\n", absBasePath, absFullPath)
	fmt.Printf("DEBUG: normalizedBase=%s, normalizedFull=%s\n", normalizedBase, normalizedFull)

	// Path is valid if it's exactly the base path or is within the base directory
	if normalizedFull != normalizedBase && !strings.HasPrefix(normalizedFull+string(filepath.Separator), normalizedBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes base directory")
	}

	return absFullPath, nil
}

func main() {
	// Test case: base path /output, creating project/src/main.go
	result, err := debugSafePath("/output", "project")
	fmt.Printf("Result for 'project': %s, Error: %v\n", result, err)

	result, err = debugSafePath("/output", "project/src")
	fmt.Printf("Result for 'project/src': %s, Error: %v\n", result, err)

	result, err = debugSafePath("/output", ".")
	fmt.Printf("Result for '.': %s, Error: %v\n", result, err)
}