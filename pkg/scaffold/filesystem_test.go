package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryFileSystem_BasicOperations(t *testing.T) {
	fs := NewMemoryFileSystem()

	// Test file doesn't exist initially
	assert.False(t, fs.Exists("test.txt"))

	// Write a file
	content := []byte("Hello, World!")
	err := fs.WriteFile("test.txt", content, 0644)
	require.NoError(t, err)

	// Check file exists
	assert.True(t, fs.Exists("test.txt"))
	assert.False(t, fs.IsDir("test.txt"))

	// Read file back
	readContent, err := fs.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Check file info
	info, err := fs.Stat("test.txt")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name())
	assert.Equal(t, int64(len(content)), info.Size())
	assert.False(t, info.IsDir())
}

func TestMemoryFileSystem_Directories(t *testing.T) {
	fs := NewMemoryFileSystem()

	// Create directory structure
	err := fs.MkdirAll("config/app", 0755)
	require.NoError(t, err)

	// Check directories exist
	assert.True(t, fs.Exists("config"))
	assert.True(t, fs.IsDir("config"))
	assert.True(t, fs.Exists("config/app"))
	assert.True(t, fs.IsDir("config/app"))

	// Write file in subdirectory
	err = fs.WriteFile("config/app/settings.yaml", []byte("debug: true"), 0644)
	require.NoError(t, err)

	assert.True(t, fs.Exists("config/app/settings.yaml"))
	assert.False(t, fs.IsDir("config/app/settings.yaml"))
}

func TestMemoryFileSystem_NonDestructiveWrite(t *testing.T) {
	fs := NewMemoryFileSystem()

	// Write initial file
	err := fs.WriteFile("test.txt", []byte("initial content"), 0644)
	require.NoError(t, err)

	// Try to overwrite - should fail
	err = fs.WriteFile("test.txt", []byte("new content"), 0644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Content should remain unchanged
	content, err := fs.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("initial content"), content)
}

func TestMemoryFileSystem_RemoveOperations(t *testing.T) {
	fs := NewMemoryFileSystem()

	// Create test structure
	err := fs.MkdirAll("dir1/subdir", 0755)
	require.NoError(t, err)
	err = fs.WriteFile("dir1/file1.txt", []byte("content1"), 0644)
	require.NoError(t, err)
	err = fs.WriteFile("dir1/subdir/file2.txt", []byte("content2"), 0644)
	require.NoError(t, err)

	// Remove single file
	err = fs.Remove("dir1/file1.txt")
	require.NoError(t, err)
	assert.False(t, fs.Exists("dir1/file1.txt"))
	assert.True(t, fs.Exists("dir1/subdir/file2.txt"))

	// Remove directory with contents
	err = fs.RemoveAll("dir1")
	require.NoError(t, err)
	assert.False(t, fs.Exists("dir1"))
	assert.False(t, fs.Exists("dir1/subdir"))
	assert.False(t, fs.Exists("dir1/subdir/file2.txt"))
}

func TestMemoryFileSystem_PathSafety(t *testing.T) {
	fs := NewMemoryFileSystem()

	// Test reading non-existent file
	_, err := fs.ReadFile("nonexistent.txt")
	assert.Error(t, err)

	// Test stat on non-existent path
	_, err = fs.Stat("nonexistent.txt")
	assert.Error(t, err)
}

func TestOSFileSystem_PathSafety(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	fs, err := NewOSFileSystem(tempDir)
	require.NoError(t, err)

	// Test writing within base path
	err = fs.WriteFile("test.txt", []byte("safe content"), 0644)
	require.NoError(t, err)

	// Verify file was created in temp directory
	actualPath := filepath.Join(tempDir, "test.txt")
	assert.FileExists(t, actualPath)

	// Test reading back
	content, err := fs.ReadFile("test.txt")
	require.NoError(t, err)
	assert.Equal(t, []byte("safe content"), content)
}

func TestOSFileSystem_PathTraversalPrevention(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewOSFileSystem(tempDir)
	require.NoError(t, err)

	// Test various path traversal attempts
	// Note: Windows-style paths (backslashes, drive letters) are only relevant on Windows
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"subdir/../../../etc/passwd",
		"/etc/passwd",
	}

	// Add Windows-specific paths only on Windows
	if runtime.GOOS == "windows" {
		pathTraversalAttempts = append(pathTraversalAttempts,
			"..\\..\\..\\windows\\system32\\config\\sam",
			"C:\\Windows\\System32\\config\\sam",
		)
	}

	for _, attempt := range pathTraversalAttempts {
		t.Run("prevent_"+attempt, func(t *testing.T) {
			err := fs.WriteFile(attempt, []byte("malicious"), 0644)
			assert.Error(t, err, "should prevent path traversal for: %s", attempt)
		})
	}
}

func TestOSFileSystem_AtomicWrite(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewOSFileSystem(tempDir)
	require.NoError(t, err)

	// Write a file
	content := []byte("atomic write test")
	err = fs.WriteFile("atomic.txt", content, 0644)
	require.NoError(t, err)

	// Verify file exists and has correct content
	actualPath := filepath.Join(tempDir, "atomic.txt")
	assert.FileExists(t, actualPath)

	readContent, err := os.ReadFile(actualPath)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)

	// Verify no temporary files remain
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	for _, entry := range entries {
		assert.False(t,
			filepath.HasPrefix(entry.Name(), ".tmp-scaffold-"),
			"temporary file should not remain: %s", entry.Name(),
		)
	}
}

func TestOSFileSystem_NonDestructiveWrite(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewOSFileSystem(tempDir)
	require.NoError(t, err)

	// Create initial file using OS operations
	initialPath := filepath.Join(tempDir, "existing.txt")
	err = os.WriteFile(initialPath, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Try to write using filesystem - should fail
	err = fs.WriteFile("existing.txt", []byte("new content"), 0644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// Verify original content is preserved
	content, err := os.ReadFile(initialPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("existing content"), content)
}

func TestOSFileSystem_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()

	fs, err := NewOSFileSystem(tempDir)
	require.NoError(t, err)

	// Write file in nested directory that doesn't exist
	err = fs.WriteFile("deep/nested/structure/file.txt", []byte("content"), 0644)
	require.NoError(t, err)

	// Verify directory structure was created
	assert.DirExists(t, filepath.Join(tempDir, "deep"))
	assert.DirExists(t, filepath.Join(tempDir, "deep", "nested"))
	assert.DirExists(t, filepath.Join(tempDir, "deep", "nested", "structure"))
	assert.FileExists(t, filepath.Join(tempDir, "deep", "nested", "structure", "file.txt"))
}

func TestSafeFileWriter_DryRun(t *testing.T) {
	fs := NewMemoryFileSystem()
	writer := NewSafeFileWriter(fs, true, nil) // dry run enabled

	// Write should succeed but not actually write
	err := writer.WriteFileWithContext(context.Background(), "test.txt", []byte("content"), 0644)
	require.NoError(t, err)

	// File should not exist in filesystem
	assert.False(t, fs.Exists("test.txt"))
}

func TestSafeFileWriter_AllowedPaths(t *testing.T) {
	fs := NewMemoryFileSystem()
	allowedPaths := []string{"allowed/", "also-allowed.txt"}
	writer := NewSafeFileWriter(fs, false, allowedPaths)

	// Writing to allowed path should succeed
	err := writer.WriteFileWithContext(context.Background(), "allowed/file.txt", []byte("content"), 0644)
	require.NoError(t, err)
	assert.True(t, fs.Exists("allowed/file.txt"))

	// Writing to disallowed path should fail
	err = writer.WriteFileWithContext(context.Background(), "forbidden/file.txt", []byte("content"), 0644)
	require.Error(t, err)
	// The error should indicate the path is not allowed (may have different message formats)
	errMsg := err.Error()
	pathRejected := strings.Contains(errMsg, "not in allowed list") || strings.Contains(errMsg, "invalid file path")
	assert.True(t, pathRejected, "expected error about disallowed path, got: %v", err)
}

func TestSafeFileWriter_ContextCancellation(t *testing.T) {
	fs := NewMemoryFileSystem()
	writer := NewSafeFileWriter(fs, false, nil)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Write should fail due to context cancellation
	err := writer.WriteFileWithContext(ctx, "test.txt", []byte("content"), 0644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestPathTraversalDetection(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"normal/path", false},
		{"../traversal", true},
		{"path/../traversal", true},
		{"path/../../traversal", true},
		{"..\\windows\\traversal", true},
		{"/absolute/path", false}, // Not traversal, but still invalid
		{".", false},
		{"..", true},
		{"...", false}, // Just dots, not traversal
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := containsPathTraversal(tt.path)
			assert.Equal(t, tt.expected, result, "path: %s", tt.path)
		})
	}
}

func BenchmarkMemoryFileSystem_Write(b *testing.B) {
	fs := NewMemoryFileSystem()
	content := []byte("benchmark content")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filename := filepath.Join("bench", string(rune(i%1000)), "file.txt")
		_ = fs.WriteFile(filename, content, 0644)
	}
}

func BenchmarkMemoryFileSystem_Read(b *testing.B) {
	fs := NewMemoryFileSystem()
	content := []byte("benchmark content")

	// Prepare files
	for i := 0; i < 1000; i++ {
		filename := filepath.Join("bench", string(rune(i)), "file.txt")
		_ = fs.WriteFile(filename, content, 0644)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filename := filepath.Join("bench", string(rune(i%1000)), "file.txt")
		_, _ = fs.ReadFile(filename)
	}
}

func BenchmarkOSFileSystem_AtomicWrite(b *testing.B) {
	tempDir := b.TempDir()
	fs, _ := NewOSFileSystem(tempDir)
	content := []byte("benchmark content for atomic write")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filename := filepath.Join("bench", string(rune(i%100)), "file.txt")
		_ = fs.WriteFile(filename, content, 0644)
	}
}
