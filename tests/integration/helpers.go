// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestContainer wraps container operations for testing
type TestContainer struct {
	container testcontainers.Container
	ctx       context.Context
	t         *testing.T
}

// NewTestContainer creates a new Alpine container for testing
func NewTestContainer(t *testing.T) (*TestContainer, error) {
	ctx := context.Background()

	// Build scaffold binary first
	scaffoldBinary, err := buildScaffoldBinary(t)
	if err != nil {
		return nil, fmt.Errorf("failed to build scaffold binary: %w", err)
	}

	req := testcontainers.ContainerRequest{
		Image:        "alpine:latest",
		Cmd:          []string{"sleep", "3600"}, // Keep container running
		WaitingFor:   wait.ForExec([]string{"true"}).WithStartupTimeout(30 * time.Second),
		AutoRemove:   true,
		Mounts: testcontainers.ContainerMounts{
			{
				Source:   testcontainers.GenericBindMountSource{HostPath: scaffoldBinary},
				Target:   "/scaffold",
				ReadOnly: false,
			},
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Check if scaffold binary exists and make it executable
	exitCode, output, err := container.Exec(ctx, []string{"ls", "-la", "/scaffold"})
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to check scaffold binary: %w", err)
	}
	if exitCode != 0 {
		outputBytes, _ := io.ReadAll(output)
		container.Terminate(ctx)
		return nil, fmt.Errorf("scaffold binary not found, ls output: %s", string(outputBytes))
	}

	// Make scaffold executable in container
	exitCode, output, err = container.Exec(ctx, []string{"chmod", "+x", "/scaffold"})
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to make scaffold executable: %w", err)
	}
	if exitCode != 0 {
		outputBytes, _ := io.ReadAll(output)
		container.Terminate(ctx)
		return nil, fmt.Errorf("chmod failed with exit code %d, output: %s", exitCode, string(outputBytes))
	}

	return &TestContainer{
		container: container,
		ctx:       ctx,
		t:         t,
	}, nil
}

// buildScaffoldBinary builds the scaffold binary for testing
func buildScaffoldBinary(t *testing.T) (string, error) {
	// Get the project root directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Navigate to project root (from tests/integration/)
	projectRoot := filepath.Join(cwd, "../..")
	projectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Build to bin/linux directory in project root (accessible to Docker)
	binDir := filepath.Join(projectRoot, "bin", "linux")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin/linux directory: %w", err)
	}

	binaryPath := filepath.Join(binDir, "scaffold")

	// Build the binary for Linux (required for Alpine containers)
	cmd := fmt.Sprintf("cd %s && GOOS=linux GOARCH=amd64 go build -o %s ./cmd/scaffold", projectRoot, binaryPath)
	if err := runCommand(cmd); err != nil {
		return "", fmt.Errorf("failed to build binary: %w", err)
	}

	return binaryPath, nil
}

// runCommand executes a shell command
func runCommand(cmd string) error {
	if cmd == "" {
		return fmt.Errorf("empty command")
	}

	// Use shell to handle complex commands with && and environment variables
	c := exec.Command("sh", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// CopyToContainer copies files to the container
func (tc *TestContainer) CopyToContainer(sourcePath, targetPath string) error {
	fileContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	return tc.container.CopyToContainer(
		tc.ctx,
		fileContent,
		targetPath,
		0644,
	)
}

// CopyDirToContainer copies a directory to the container
func (tc *TestContainer) CopyDirToContainer(sourceDir, targetDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			// Create directory in container
			exitCode, _, err := tc.container.Exec(tc.ctx, []string{"mkdir", "-p", targetPath})
			if err != nil || exitCode != 0 {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			return nil
		}

		// Copy file
		return tc.CopyToContainer(path, targetPath)
	})
}

// RunScaffold runs the scaffold command in the container
func (tc *TestContainer) RunScaffold(args ...string) (string, error) {
	cmd := append([]string{"/scaffold"}, args...)
	
	exitCode, reader, err := tc.container.Exec(tc.ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to execute scaffold: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read output: %w", err)
	}

	if exitCode != 0 {
		return string(output), fmt.Errorf("scaffold exited with code %d: %s", exitCode, output)
	}

	return string(output), nil
}

// ListDirectory lists files in a container directory
func (tc *TestContainer) ListDirectory(path string) ([]string, error) {
	exitCode, reader, err := tc.container.Exec(tc.ctx, []string{"find", path, "-type", "f"})
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	if exitCode != 0 {
		return nil, fmt.Errorf("find command failed with exit code %d", exitCode)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" && line != path {
			files = append(files, line)
		}
	}

	return files, nil
}

// ReadFile reads a file from the container
func (tc *TestContainer) ReadFile(path string) (string, error) {
	exitCode, reader, err := tc.container.Exec(tc.ctx, []string{"cat", path})
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read output: %w", err)
	}

	if exitCode != 0 {
		return "", fmt.Errorf("cat command failed with exit code %d: %s", exitCode, output)
	}

	return string(output), nil
}

// CheckFileExists checks if a file exists in the container
func (tc *TestContainer) CheckFileExists(path string) (bool, error) {
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{"test", "-f", path})
	if err != nil {
		return false, fmt.Errorf("failed to check file: %w", err)
	}

	return exitCode == 0, nil
}

// CheckDirExists checks if a directory exists in the container
func (tc *TestContainer) CheckDirExists(path string) (bool, error) {
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{"test", "-d", path})
	if err != nil {
		return false, fmt.Errorf("failed to check directory: %w", err)
	}

	return exitCode == 0, nil
}

// Cleanup terminates the container
func (tc *TestContainer) Cleanup() {
	if tc.container != nil {
		tc.container.Terminate(tc.ctx)
	}
}

// FileSystemSnapshot captures the state of a directory tree
type FileSystemSnapshot struct {
	Files map[string]string // path -> content
	Dirs  []string          // directory paths
}

// CaptureSnapshot captures the filesystem state in the container
func (tc *TestContainer) CaptureSnapshot(rootPath string) (*FileSystemSnapshot, error) {
	snapshot := &FileSystemSnapshot{
		Files: make(map[string]string),
		Dirs:  []string{},
	}

	// Find all directories
	exitCode, reader, err := tc.container.Exec(tc.ctx, []string{"find", rootPath, "-type", "d"})
	if err != nil {
		return nil, fmt.Errorf("failed to find directories: %w", err)
	}

	if exitCode == 0 {
		output, _ := io.ReadAll(reader)
		for _, dir := range strings.Split(strings.TrimSpace(string(output)), "\n") {
			if dir != "" && dir != rootPath {
				relPath, _ := filepath.Rel(rootPath, dir)
				if relPath != "." {
					snapshot.Dirs = append(snapshot.Dirs, relPath)
				}
			}
		}
	}

	// Find all files and read their content
	files, err := tc.ListDirectory(rootPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		content, err := tc.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}

		relPath, _ := filepath.Rel(rootPath, file)
		snapshot.Files[relPath] = content
	}

	return snapshot, nil
}

// ValidateSnapshot compares actual snapshot with expected
func ValidateSnapshot(t *testing.T, actual, expected *FileSystemSnapshot) {
	// Check directories
	require.ElementsMatch(t, expected.Dirs, actual.Dirs, "directory structure mismatch")

	// Check files
	require.Equal(t, len(expected.Files), len(actual.Files), "file count mismatch")

	for path, expectedContent := range expected.Files {
		actualContent, exists := actual.Files[path]
		require.True(t, exists, "missing file: %s", path)
		require.Equal(t, expectedContent, actualContent, "content mismatch in file: %s", path)
	}
}

// CreateRegistryConfig creates a scaffold registry config file in the container
func (tc *TestContainer) CreateRegistryConfig(configPath string, scaffolds []RegistryEntry) error {
	content := "scaffolds:\n"
	for _, s := range scaffolds {
		content += "  - name: " + s.Name + "\n"
		content += "    path: " + s.Path + "\n"
		if s.Description != "" {
			content += "    description: " + s.Description + "\n"
		}
	}

	// Write config to container
	exitCode, _, err := tc.container.Exec(tc.ctx, []string{
		"sh", "-c",
		"mkdir -p $(dirname " + configPath + ") && printf '%s' '" + content + "' > " + configPath,
	})
	if err != nil || exitCode != 0 {
		return fmt.Errorf("failed to create registry config: %w", err)
	}

	return nil
}

// RegistryEntry represents a scaffold registry entry
type RegistryEntry struct {
	Name        string
	Path        string
	Description string
}

// LoadExpectedSnapshot loads expected output from fixtures
func LoadExpectedSnapshot(fixtureDir string) (*FileSystemSnapshot, error) {
	snapshot := &FileSystemSnapshot{
		Files: make(map[string]string),
		Dirs:  []string{},
	}

	err := filepath.Walk(fixtureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(fixtureDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if info.IsDir() {
			snapshot.Dirs = append(snapshot.Dirs, relPath)
		} else {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snapshot.Files[relPath] = string(content)
		}

		return nil
	})

	return snapshot, err
}