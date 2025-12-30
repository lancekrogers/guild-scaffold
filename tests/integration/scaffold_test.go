// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScaffoldMinimalTemplate(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/minimal.yaml", "/test/minimal.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/minimal", "/test/templates/minimal")
	require.NoError(t, err)

	// Run scaffold
	output, err := container.RunScaffold(
		"init", "test-project",
		"--template", "/test/minimal.yaml",
		"--output", "/output",
		"--var", "project_name=TestProject",
		"--var", "module_name=github.com/test/project",
	)
	require.NoError(t, err, "scaffold command failed: %s", output)

	// Capture actual output
	actual, err := container.CaptureSnapshot("/output")
	require.NoError(t, err)

	// Load expected output
	expected, err := LoadExpectedSnapshot("fixtures/expected/minimal")
	require.NoError(t, err)

	// Validate
	ValidateSnapshot(t, actual, expected)
}

func TestScaffoldLibraryTemplate(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/library.yaml", "/test/library.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/library", "/test/templates/library")
	require.NoError(t, err)

	// Run scaffold
	output, err := container.RunScaffold(
		"init", "test-library",
		"--template", "/test/library.yaml",
		"--output", "/output",
		"--var", "package_name=testlib",
		"--var", "module_name=github.com/test/library",
	)
	require.NoError(t, err, "scaffold command failed: %s", output)

	// Verify specific files exist
	exists, err := container.CheckFileExists("/output/pkg/testlib/testlib.go")
	require.NoError(t, err)
	require.True(t, exists, "main library file should exist")

	exists, err = container.CheckFileExists("/output/pkg/testlib/testlib_test.go")
	require.NoError(t, err)
	require.True(t, exists, "test file should exist")

	exists, err = container.CheckDirExists("/output/examples")
	require.NoError(t, err)
	require.True(t, exists, "examples directory should exist")
}

func TestScaffoldWithExistingFiles(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/minimal.yaml", "/test/minimal.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/minimal", "/test/templates/minimal")
	require.NoError(t, err)

	// First run - should succeed
	output, err := container.RunScaffold(
		"init", "test-project",
		"--template", "/test/minimal.yaml",
		"--output", "/output",
		"--var", "project_name=TestProject",
		"--var", "module_name=github.com/test/project",
	)
	require.NoError(t, err, "first scaffold run failed: %s", output)

	// Second run without force - should fail
	output, err = container.RunScaffold(
		"init", "test-project",
		"--template", "/test/minimal.yaml",
		"--output", "/output",
		"--var", "project_name=TestProject",
		"--var", "module_name=github.com/test/project",
	)
	require.Error(t, err, "second scaffold run should fail without --force")
	require.Contains(t, output, "not empty", "error should mention directory is not empty")

	// Third run with force - should succeed
	output, err = container.RunScaffold(
		"init", "test-project",
		"--template", "/test/minimal.yaml",
		"--output", "/output",
		"--force",
		"--var", "project_name=TestProject",
		"--var", "module_name=github.com/test/project",
	)
	require.NoError(t, err, "scaffold with --force failed: %s", output)
}

func TestScaffoldDryRun(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/minimal.yaml", "/test/minimal.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/minimal", "/test/templates/minimal")
	require.NoError(t, err)

	// Run scaffold with dry-run
	output, err := container.RunScaffold(
		"init", "test-project",
		"--template", "/test/minimal.yaml",
		"--output", "/output",
		"--dry-run",
		"--var", "project_name=TestProject",
		"--var", "module_name=github.com/test/project",
	)
	require.NoError(t, err, "dry-run failed: %s", output)
	require.Contains(t, output, "Dry Run Mode", "output should indicate dry run")

	// Verify no files were created
	exists, err := container.CheckDirExists("/output")
	require.NoError(t, err)
	require.False(t, exists, "output directory should not exist after dry run")
}

func TestScaffoldComplexStructure(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/complex.yaml", "/test/complex.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/complex", "/test/templates/complex")
	require.NoError(t, err)

	// Run scaffold
	output, err := container.RunScaffold(
		"init", "test-complex",
		"--template", "/test/complex.yaml",
		"--output", "/output",
		"--var", "project_name=ComplexProject",
		"--var", "module_name=github.com/test/complex",
		"--var", "author=Test Author",
		"--var", "license=MIT",
	)
	require.NoError(t, err, "scaffold command failed: %s", output)

	// Verify complex directory structure
	dirs := []string{
		"/output/cmd/server",
		"/output/cmd/client",
		"/output/pkg/api",
		"/output/pkg/models",
		"/output/internal/config",
		"/output/internal/database",
		"/output/deployments/docker",
		"/output/deployments/kubernetes",
		"/output/docs",
		"/output/tests",
	}

	for _, dir := range dirs {
		exists, err := container.CheckDirExists(dir)
		require.NoError(t, err)
		require.True(t, exists, "directory %s should exist", dir)
	}

	// Check that variables were properly substituted
	readme, err := container.ReadFile("/output/README.md")
	require.NoError(t, err)
	require.Contains(t, readme, "ComplexProject")
	require.Contains(t, readme, "Test Author")
	require.Contains(t, readme, "MIT")
}

func TestScaffoldEmptyDirectories(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/with-empty.yaml", "/test/with-empty.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/with-empty", "/test/templates/with-empty")
	require.NoError(t, err)

	// Run scaffold
	output, err := container.RunScaffold(
		"init", "test-empty",
		"--template", "/test/with-empty.yaml",
		"--output", "/output",
	)
	require.NoError(t, err, "scaffold command failed: %s", output)

	// Verify empty directories were created
	emptyDirs := []string{
		"/output/logs",
		"/output/data",
		"/output/tmp",
		"/output/assets/images",
		"/output/assets/styles",
	}

	for _, dir := range emptyDirs {
		exists, err := container.CheckDirExists(dir)
		require.NoError(t, err)
		require.True(t, exists, "empty directory %s should exist", dir)

		// Verify directory is empty or only contains .gitkeep (which is used to preserve empty dirs in git)
		files, err := container.ListDirectory(dir)
		require.NoError(t, err)
		// Filter out .gitkeep files which are expected markers for empty directories
		var nonGitkeepFiles []string
		for _, f := range files {
			if !strings.HasSuffix(f, ".gitkeep") {
				nonGitkeepFiles = append(nonGitkeepFiles, f)
			}
		}
		require.Empty(t, nonGitkeepFiles, "directory %s should be empty (except for .gitkeep)", dir)
	}
}

func TestScaffoldVariableSubstitution(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/variables.yaml", "/test/variables.yaml")
	require.NoError(t, err)

	err = container.CopyDirToContainer("fixtures/templates/variables", "/test/templates/variables")
	require.NoError(t, err)

	// Run scaffold with various variable types
	// Note: Cobra's StringSlice uses CSV parsing, so:
	// - Commas in values will be interpreted as separators (use StringArray instead if needed)
	// - Bare quotes in values are not supported
	output, err := container.RunScaffold(
		"init", "test-vars",
		"--template", "/test/variables.yaml",
		"--output", "/output",
		"--var", "string_var=hello world",
		"--var", "int_var=42",
		"--var", "bool_var=true",
		"--var", "special_chars=Hello World & Friends!",
		"--var", "list_var=item1;item2;item3",
	)
	require.NoError(t, err, "scaffold command failed: %s", output)

	// Read generated config file (in config/ directory per fixture structure)
	config, err := container.ReadFile("/output/config/config.yaml")
	require.NoError(t, err)

	// Verify variables were substituted correctly
	require.Contains(t, config, "string_value: hello world")
	require.Contains(t, config, "int_value: 42")
	require.Contains(t, config, "bool_value: true")
	require.Contains(t, config, "special: Hello World & Friends!")
}

func TestScaffoldPathValidation(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy fixtures to container
	err := container.CopyToContainer("fixtures/malicious.yaml", "/test/malicious.yaml")
	require.NoError(t, err)

	// Try to scaffold with path traversal attempts
	output, err := container.RunScaffold(
		"init", "test-malicious",
		"--template", "/test/malicious.yaml",
		"--output", "/output",
	)
	require.Error(t, err, "scaffold should fail with malicious paths")
	// Either "path traversal" or "absolute paths" error is acceptable - both are security errors
	hasPathTraversal := strings.Contains(output, "path traversal")
	hasAbsolutePath := strings.Contains(output, "absolute paths")
	require.True(t, hasPathTraversal || hasAbsolutePath, "error should mention path traversal or absolute paths")

	// Verify no files were created outside the output directory
	exists, err := container.CheckFileExists("/etc/passwd")
	require.NoError(t, err)
	// /etc/passwd should exist in Alpine but shouldn't be modified
	if exists {
		content, err := container.ReadFile("/etc/passwd")
		require.NoError(t, err)
		require.NotContains(t, content, "INJECTED", "system file should not be modified")
	}
}
