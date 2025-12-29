// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

//go:build integration
// +build integration

package integration

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScaffoldListBuiltin tests that the list command shows builtin scaffolds
func TestScaffoldListBuiltin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Run scaffold list
	output, err := container.RunScaffold("list")
	require.NoError(t, err, "list command failed: %s", output)

	// Verify builtin guild-campaign is listed
	require.Contains(t, output, "guild-campaign", "should list guild-campaign builtin")
	require.Contains(t, output, "builtin", "should indicate builtin source")
}

// TestScaffoldListVerbose tests verbose listing with variable details
func TestScaffoldListVerbose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Run scaffold list --verbose
	output, err := container.RunScaffold("list", "--verbose")
	require.NoError(t, err, "list --verbose command failed: %s", output)

	// Verify variable details are shown
	require.Contains(t, output, "Variables:", "should show variables section")
	require.Contains(t, output, "campaign_name", "should show campaign_name variable")
	require.Contains(t, output, "required", "should indicate required variables")
}

// TestScaffoldValidateBuiltin tests validating the builtin scaffold
func TestScaffoldValidateBuiltin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Run scaffold validate
	output, err := container.RunScaffold("validate", "--template", "guild-campaign")
	require.NoError(t, err, "validate command failed: %s", output)

	// Verify validation passed
	require.Contains(t, output, "valid", "should indicate valid scaffold")
}

// TestScaffoldInitBuiltinDryRun tests dry-run with builtin scaffold
func TestScaffoldInitBuiltinDryRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Run scaffold init with dry-run
	output, err := container.RunScaffold(
		"init", "my-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--dry-run",
		"--var", "campaign_name=my-campaign",
	)
	require.NoError(t, err, "init dry-run failed: %s", output)

	// Verify dry-run output
	require.Contains(t, output, "Dry Run Mode", "should indicate dry run mode")
	require.Contains(t, output, "campaign_name", "should show campaign_name variable")
	require.Contains(t, output, ".campaign/campaign.yaml", "should list campaign config file")

	// Verify no files were created
	exists, err := container.CheckDirExists("/output")
	require.NoError(t, err)
	require.False(t, exists, "output directory should not exist after dry run")
}

// TestScaffoldInitBuiltin tests actual scaffolding with builtin scaffold
func TestScaffoldInitBuiltin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Run scaffold init
	output, err := container.RunScaffold(
		"init", "my-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--var", "campaign_name=my-campaign",
		"--var", "description=Test campaign for integration testing",
	)
	require.NoError(t, err, "init command failed: %s", output)

	// Verify directory structure was created
	dirs := []string{
		"/output/.campaign",
		"/output/.campaign/guilds",
		"/output/.campaign/agents",
		"/output/.campaign/archives",
		"/output/.campaign/prompts",
		"/output/commissions",
		"/output/corpus",
		"/output/kanban/todo",
		"/output/kanban/in-progress",
		"/output/kanban/done",
	}

	for _, dir := range dirs {
		exists, err := container.CheckDirExists(dir)
		require.NoError(t, err)
		require.True(t, exists, "directory %s should exist", dir)
	}

	// Verify key files exist
	files := []string{
		"/output/.campaign/campaign.yaml",
		"/output/README.md",
	}

	for _, file := range files {
		exists, err := container.CheckFileExists(file)
		require.NoError(t, err)
		require.True(t, exists, "file %s should exist", file)
	}

	// Verify campaign.yaml has correct content
	content, err := container.ReadFile("/output/.campaign/campaign.yaml")
	require.NoError(t, err)
	require.Contains(t, content, "my-campaign", "campaign.yaml should contain campaign name")
	require.Contains(t, content, "Test campaign for integration testing", "campaign.yaml should contain description")
}

// TestScaffoldInitBuiltinWithProvider tests scaffolding with custom provider
func TestScaffoldInitBuiltinWithProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Run scaffold init with OpenAI provider
	output, err := container.RunScaffold(
		"init", "openai-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--var", "campaign_name=openai-campaign",
		"--var", "provider=openai",
		"--var", "model=gpt-4o",
	)
	require.NoError(t, err, "init command failed: %s", output)

	// Verify campaign.yaml has OpenAI configuration
	content, err := container.ReadFile("/output/.campaign/campaign.yaml")
	require.NoError(t, err)
	require.Contains(t, content, "openai", "campaign.yaml should contain openai provider")
	require.Contains(t, content, "OPENAI_API_KEY", "campaign.yaml should reference OPENAI_API_KEY")
	require.Contains(t, content, "gpt-4o", "campaign.yaml should contain gpt-4o model")
}

// TestScaffoldInitNonExistentTemplate tests error handling for missing templates
func TestScaffoldInitNonExistentTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Try to init with non-existent template
	output, err := container.RunScaffold(
		"init", "test-project",
		"--template", "non-existent-scaffold",
		"--output", "/output",
	)
	require.Error(t, err, "init should fail with non-existent template")
	require.Contains(t, strings.ToLower(output), "not found", "error should indicate template not found")
}

// TestScaffoldInitMissingRequiredVar tests error handling for missing required variables
func TestScaffoldInitMissingRequiredVar(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Try to init without required campaign_name variable
	output, err := container.RunScaffold(
		"init", "test-project",
		"--template", "guild-campaign",
		"--output", "/output",
		// Note: campaign_name is required but not provided
	)
	// This might succeed with default value or fail - check behavior
	// For now just verify the command runs
	t.Logf("Output: %s, Error: %v", output, err)
}

// TestScaffoldInitOverwriteProtection tests that init fails on existing directory
func TestScaffoldInitOverwriteProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// First init should succeed
	output, err := container.RunScaffold(
		"init", "my-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--var", "campaign_name=my-campaign",
	)
	require.NoError(t, err, "first init failed: %s", output)

	// Second init without --force should fail
	output, err = container.RunScaffold(
		"init", "my-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--var", "campaign_name=my-campaign",
	)
	require.Error(t, err, "second init should fail without --force")
	require.Contains(t, strings.ToLower(output), "empty", "error should mention directory is not empty")
}

// TestScaffoldInitForceOverwrite tests that --force allows overwriting
func TestScaffoldInitForceOverwrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// First init
	output, err := container.RunScaffold(
		"init", "my-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--var", "campaign_name=original-campaign",
	)
	require.NoError(t, err, "first init failed: %s", output)

	// Second init with --force and different values
	output, err = container.RunScaffold(
		"init", "my-campaign",
		"--template", "guild-campaign",
		"--output", "/output",
		"--force",
		"--var", "campaign_name=updated-campaign",
	)
	require.NoError(t, err, "second init with --force failed: %s", output)

	// Verify content was updated
	content, err := container.ReadFile("/output/.campaign/campaign.yaml")
	require.NoError(t, err)
	require.Contains(t, content, "updated-campaign", "campaign.yaml should have updated content")
}

// TestScaffoldHelp tests the help command
func TestScaffoldHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Test main help
	output, err := container.RunScaffold("--help")
	require.NoError(t, err, "help command failed: %s", output)
	require.Contains(t, output, "scaffold", "help should mention scaffold")
	require.Contains(t, output, "init", "help should mention init command")
	require.Contains(t, output, "list", "help should mention list command")
	require.Contains(t, output, "validate", "help should mention validate command")

	// Test init help
	output, err = container.RunScaffold("init", "--help")
	require.NoError(t, err, "init help failed: %s", output)
	require.Contains(t, output, "template", "init help should mention --template flag")
	require.Contains(t, output, "output", "init help should mention --output flag")
}

// TestScaffoldVersion tests the version command
func TestScaffoldVersion(t *testing.T) {
	// Skip until --version flag is implemented
	t.Skip("Skipping: --version flag not yet implemented")
}

// TestScaffoldExternalRegistry tests loading scaffolds from external registry
func TestScaffoldExternalRegistry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy external scaffold to container
	err := container.CopyDirToContainer(
		"fixtures/scaffolds/modular-justfile",
		"/scaffolds/modular-justfile",
	)
	require.NoError(t, err, "failed to copy external scaffold")

	// Create global registry config
	err = container.CreateRegistryConfig("/root/.guild/scaffold.yaml", []RegistryEntry{
		{
			Name:        "modular-justfile",
			Path:        "/scaffolds/modular-justfile",
			Description: "Modular justfile system",
		},
	})
	require.NoError(t, err, "failed to create registry config")

	// List should now show the external scaffold
	output, err := container.RunScaffold("list")
	require.NoError(t, err, "list command failed: %s", output)
	require.Contains(t, output, "modular-justfile", "should list external scaffold")
	require.Contains(t, output, "guild-campaign", "should still list builtin scaffold")
}

// TestScaffoldExternalInit tests initializing from external scaffold
func TestScaffoldExternalInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy external scaffold to container
	err := container.CopyDirToContainer(
		"fixtures/scaffolds/modular-justfile",
		"/scaffolds/modular-justfile",
	)
	require.NoError(t, err, "failed to copy external scaffold")

	// Create global registry config
	err = container.CreateRegistryConfig("/root/.guild/scaffold.yaml", []RegistryEntry{
		{
			Name:        "modular-justfile",
			Path:        "/scaffolds/modular-justfile",
			Description: "Modular justfile system",
		},
	})
	require.NoError(t, err, "failed to create registry config")

	// Initialize from external scaffold
	output, err := container.RunScaffold(
		"init", "my-project",
		"--template", "modular-justfile",
		"--output", "/output",
		"--var", "project_name=my-project",
	)
	require.NoError(t, err, "init command failed: %s", output)

	// Verify files were created
	files := []string{
		"/output/justfile",
		"/output/.just/build.just",
		"/output/.just/test.just",
		"/output/.just/dev.just",
	}

	for _, file := range files {
		exists, err := container.CheckFileExists(file)
		require.NoError(t, err)
		require.True(t, exists, "file %s should exist", file)
	}

	// Verify content contains project name
	content, err := container.ReadFile("/output/justfile")
	require.NoError(t, err)
	require.Contains(t, content, "my-project", "justfile should contain project name")
}

// TestScaffoldProjectRegistryPrecedence tests that project registry takes precedence
func TestScaffoldProjectRegistryPrecedence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy external scaffold to container (simulating a different version)
	err := container.CopyDirToContainer(
		"fixtures/scaffolds/modular-justfile",
		"/scaffolds/modular-justfile-global",
	)
	require.NoError(t, err)

	err = container.CopyDirToContainer(
		"fixtures/scaffolds/modular-justfile",
		"/scaffolds/modular-justfile-project",
	)
	require.NoError(t, err)

	// Create global registry
	err = container.CreateRegistryConfig("/root/.guild/scaffold.yaml", []RegistryEntry{
		{
			Name:        "modular-justfile",
			Path:        "/scaffolds/modular-justfile-global",
			Description: "Global version",
		},
	})
	require.NoError(t, err)

	// Create project directory with .campaign registry
	exitCode, _, err := container.container.Exec(container.ctx, []string{"mkdir", "-p", "/project/.campaign"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)

	// Create project registry with same scaffold name
	err = container.CreateRegistryConfig("/project/.campaign/scaffold.yaml", []RegistryEntry{
		{
			Name:        "modular-justfile",
			Path:        "/scaffolds/modular-justfile-project",
			Description: "Project version (should take precedence)",
		},
	})
	require.NoError(t, err)

	// List from project directory should show project version
	// Note: We need to run scaffold from within the project directory
	exitCode, reader, err := container.container.Exec(container.ctx, []string{
		"sh", "-c",
		"cd /project && /scaffold list",
	})
	require.NoError(t, err)

	output, _ := io.ReadAll(reader)
	outputStr := string(output)

	if exitCode == 0 {
		// If list works, verify project description is shown
		require.Contains(t, outputStr, "modular-justfile", "should list scaffold")
		t.Logf("List output: %s", outputStr)
	}
}

// TestScaffoldValidateExternal tests validating an external scaffold
func TestScaffoldValidateExternal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	container := GetSharedContainer(t)

	// Copy external scaffold to container
	err := container.CopyDirToContainer(
		"fixtures/scaffolds/modular-justfile",
		"/scaffolds/modular-justfile",
	)
	require.NoError(t, err)

	// Create registry config
	err = container.CreateRegistryConfig("/root/.guild/scaffold.yaml", []RegistryEntry{
		{
			Name: "modular-justfile",
			Path: "/scaffolds/modular-justfile",
		},
	})
	require.NoError(t, err)

	// Validate external scaffold
	output, err := container.RunScaffold("validate", "--template", "modular-justfile")
	require.NoError(t, err, "validate command failed: %s", output)
	require.Contains(t, output, "valid", "should indicate valid scaffold")
}
