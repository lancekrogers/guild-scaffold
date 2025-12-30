// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContainerEnvironment(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Test that we can execute commands in the container
	output, err := container.RunScaffold("--help")
	require.NoError(t, err)
	require.Contains(t, output, "scaffold", "help output should mention scaffold")

	// Test that we can create directories
	exitCode, _, err := container.container.Exec(container.ctx, []string{"mkdir", "-p", "/test/dir"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)

	// Test that we can check directory existence
	exists, err := container.CheckDirExists("/test/dir")
	require.NoError(t, err)
	require.True(t, exists, "created directory should exist")
}

func TestMinimalScaffoldInContainer(t *testing.T) {
	// Skip if Docker not available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy minimal fixture to container
	err := container.CopyToContainer("fixtures/minimal.yaml", "/test/minimal.yaml")
	require.NoError(t, err)

	// Copy templates
	err = container.CopyDirToContainer("fixtures/templates/minimal", "/test/templates/minimal")
	require.NoError(t, err)

	// Run scaffold with minimal template
	output, err := container.RunScaffold(
		"init", "my-project",
		"--template", "/test/minimal.yaml",
		"--output", "/output",
		"--var", "project_name=MyTestProject",
		"--var", "module_name=github.com/test/myproject",
	)
	require.NoError(t, err, "scaffold failed: %s", output)

	// Check that key files were created
	files := []string{
		"/output/project/src/main.go",
		"/output/project/README.md",
		"/output/project/.gitignore",
		"/output/project/go.mod",
	}

	for _, file := range files {
		exists, err := container.CheckFileExists(file)
		require.NoError(t, err)
		require.True(t, exists, "file %s should exist", file)
	}

	// Verify content substitution
	mainContent, err := container.ReadFile("/output/project/src/main.go")
	require.NoError(t, err)
	require.Contains(t, mainContent, "MyTestProject", "project name should be substituted")

	goModContent, err := container.ReadFile("/output/project/go.mod")
	require.NoError(t, err)
	require.Contains(t, goModContent, "github.com/test/myproject", "module name should be substituted")
}
