// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryLoader_LoadBuiltins(t *testing.T) {
	// Create a temp dir structure that mimics the global templates dir
	tempHome := t.TempDir()
	configDir := filepath.Join(tempHome, ".config", "guild", "templates")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create a mock scaffold in the templates dir
	scaffoldDir := filepath.Join(configDir, "test-template")
	if err := os.MkdirAll(scaffoldDir, 0755); err != nil {
		t.Fatalf("Failed to create scaffold dir: %v", err)
	}

	scaffoldYaml := `name: test-template
version: "1.0"
tree:
  README.md: readme.tmpl
`
	if err := os.WriteFile(filepath.Join(scaffoldDir, "scaffold.yaml"), []byte(scaffoldYaml), 0644); err != nil {
		t.Fatalf("Failed to write scaffold.yaml: %v", err)
	}

	// Set HOME to temp dir for the test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	loader, err := NewRegistryLoader()
	if err != nil {
		t.Fatalf("Failed to create loader: %v", err)
	}

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Should have the template from global templates
	entry, ok := registry.Get("test-template")
	if !ok {
		t.Fatal("Expected to find test-template in registry")
	}

	if entry.Source != "global" {
		t.Errorf("Expected source 'global', got '%s'", entry.Source)
	}
}

func TestRegistryLoader_WithoutBuiltins(t *testing.T) {
	loader := NewRegistryLoaderWithPaths("/nonexistent", "/nonexistent").WithBuiltins(false)

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Should have no entries when builtins disabled and no templates synced
	if len(registry.List()) != 0 {
		t.Errorf("Expected empty registry when builtins disabled, got %d entries", len(registry.List()))
	}
}

func TestRegistryLoader_MergePrecedence(t *testing.T) {
	// Create temp directories for global and project
	globalDir := t.TempDir()
	projectDir := t.TempDir()

	// Create global registry with shared and global-only entries
	globalGuildDir := filepath.Join(globalDir, ".guild")
	if err := os.MkdirAll(globalGuildDir, 0755); err != nil {
		t.Fatalf("Failed to create global dir: %v", err)
	}

	globalRegistry := `
scaffolds:
  - name: shared
    path: /global/shared
    description: "Global shared"
  - name: global-only
    path: /global/only
    description: "Global only"
`
	if err := os.WriteFile(filepath.Join(globalGuildDir, "scaffold.yaml"), []byte(globalRegistry), 0644); err != nil {
		t.Fatalf("Failed to write global registry: %v", err)
	}

	// Create project registry with shared and project-only entries
	projectCampaignDir := filepath.Join(projectDir, ".campaign")
	if err := os.MkdirAll(projectCampaignDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	projectRegistry := `
scaffolds:
  - name: shared
    path: /project/shared
    description: "Project shared"
  - name: project-only
    path: /project/only
    description: "Project only"
`
	if err := os.WriteFile(filepath.Join(projectCampaignDir, "scaffold.yaml"), []byte(projectRegistry), 0644); err != nil {
		t.Fatalf("Failed to write project registry: %v", err)
	}

	// Load with both registries
	loader := NewRegistryLoaderWithPaths(projectDir, globalDir).WithBuiltins(false)

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Check shared entry uses project value (higher precedence)
	shared, ok := registry.Get("shared")
	if !ok {
		t.Fatal("Expected to find shared entry")
	}
	if shared.Path != "/project/shared" {
		t.Errorf("Expected project path, got '%s'", shared.Path)
	}
	if shared.Description != "Project shared" {
		t.Errorf("Expected project description, got '%s'", shared.Description)
	}

	// Check both unique entries exist
	if _, ok := registry.Get("global-only"); !ok {
		t.Error("Expected to find global-only entry")
	}
	if _, ok := registry.Get("project-only"); !ok {
		t.Error("Expected to find project-only entry")
	}
}

func TestGetBuiltinFS(t *testing.T) {
	// GetBuiltinFS now returns nil when templates aren't synced
	// (since we no longer embed templates)
	fsys := GetBuiltinFS()

	// Will be nil if ~/.config/guild/templates doesn't exist
	// This is expected behavior - templates need to be synced
	if fsys != nil {
		t.Log("Templates directory exists, filesystem returned")
	} else {
		t.Log("No templates synced, nil filesystem returned (expected)")
	}
}

func TestGetGlobalTemplatesFS(t *testing.T) {
	// Create temp home with templates
	tempHome := t.TempDir()
	templatesDir := filepath.Join(tempHome, ".config", "guild", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a template
	scaffoldDir := filepath.Join(templatesDir, "test-scaffold")
	if err := os.MkdirAll(scaffoldDir, 0755); err != nil {
		t.Fatalf("Failed to create scaffold dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(scaffoldDir, "scaffold.yaml"), []byte("name: test"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Set HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	fsys := GetGlobalTemplatesFS()
	if fsys == nil {
		t.Fatal("Expected non-nil filesystem when templates exist")
	}
}

func TestResolveScaffold_External(t *testing.T) {
	// Create a temp scaffold directory
	scaffoldDir := t.TempDir()

	scaffoldYaml := `
name: test-scaffold
version: "1.0"
tree:
  README.md: readme.tmpl
`
	if err := os.WriteFile(filepath.Join(scaffoldDir, "scaffold.yaml"), []byte(scaffoldYaml), 0644); err != nil {
		t.Fatalf("Failed to write scaffold.yaml: %v", err)
	}

	entry := ScaffoldEntry{
		Name:   "test-scaffold",
		Path:   scaffoldDir,
		Source: "project",
	}

	ctx := context.Background()
	def, fsys, err := ResolveScaffold(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to resolve external scaffold: %v", err)
	}

	if def.Name != "test-scaffold" {
		t.Errorf("Expected name 'test-scaffold', got '%s'", def.Name)
	}

	if fsys == nil {
		t.Error("Expected non-nil filesystem")
	}
}

func TestFindScaffold_NotFound(t *testing.T) {
	loader := NewRegistryLoaderWithPaths("/nonexistent", "/nonexistent").WithBuiltins(false)

	ctx := context.Background()

	// Should not find nonexistent
	_, err := loader.FindScaffold(ctx, "nonexistent-scaffold")
	if err == nil {
		t.Error("Expected error for nonexistent scaffold")
	}
}

func TestLoadWorkspaceTemplates(t *testing.T) {
	// Create temp workspace with templates
	workspaceDir := t.TempDir()
	templatesDir := filepath.Join(workspaceDir, ".campaign", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates dir: %v", err)
	}

	// Create a workspace template
	scaffoldDir := filepath.Join(templatesDir, "my-scaffold")
	if err := os.MkdirAll(scaffoldDir, 0755); err != nil {
		t.Fatalf("Failed to create scaffold dir: %v", err)
	}

	scaffoldYaml := `name: my-scaffold
version: "1.0"
tree:
  README.md: readme.tmpl
`
	if err := os.WriteFile(filepath.Join(scaffoldDir, "scaffold.yaml"), []byte(scaffoldYaml), 0644); err != nil {
		t.Fatalf("Failed to write scaffold.yaml: %v", err)
	}

	// Use a temp home to avoid picking up real templates
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	loader := NewRegistryLoaderWithPaths(workspaceDir, tempHome).WithBuiltins(false)

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Should find the workspace template
	entry, ok := registry.Get("my-scaffold")
	if !ok {
		t.Fatal("Expected to find my-scaffold in registry")
	}

	if entry.Source != "workspace" {
		t.Errorf("Expected source 'workspace', got '%s'", entry.Source)
	}
}

func TestWorkspaceTakesPrecedenceOverGlobal(t *testing.T) {
	// Create temp home with global templates
	tempHome := t.TempDir()
	globalTemplatesDir := filepath.Join(tempHome, ".config", "guild", "templates")
	if err := os.MkdirAll(globalTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create global templates dir: %v", err)
	}

	// Create global template
	globalScaffoldDir := filepath.Join(globalTemplatesDir, "shared-scaffold")
	if err := os.MkdirAll(globalScaffoldDir, 0755); err != nil {
		t.Fatalf("Failed to create global scaffold dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(globalScaffoldDir, "scaffold.yaml"), []byte("name: shared-scaffold\ndescription: global version"), 0644); err != nil {
		t.Fatalf("Failed to write global scaffold.yaml: %v", err)
	}

	// Create workspace with same template name
	workspaceDir := t.TempDir()
	workspaceTemplatesDir := filepath.Join(workspaceDir, ".campaign", "templates")
	if err := os.MkdirAll(workspaceTemplatesDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace templates dir: %v", err)
	}

	workspaceScaffoldDir := filepath.Join(workspaceTemplatesDir, "shared-scaffold")
	if err := os.MkdirAll(workspaceScaffoldDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace scaffold dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceScaffoldDir, "scaffold.yaml"), []byte("name: shared-scaffold\ndescription: workspace version"), 0644); err != nil {
		t.Fatalf("Failed to write workspace scaffold.yaml: %v", err)
	}

	// Set HOME
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	loader := NewRegistryLoaderWithPaths(workspaceDir, tempHome)

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Should find the scaffold with workspace source (workspace takes precedence)
	entry, ok := registry.Get("shared-scaffold")
	if !ok {
		t.Fatal("Expected to find shared-scaffold in registry")
	}

	if entry.Source != "workspace" {
		t.Errorf("Expected source 'workspace' to take precedence, got '%s'", entry.Source)
	}
}
