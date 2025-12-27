// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scaffold

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistryLoader_LoadBuiltins(t *testing.T) {
	loader := NewRegistryLoaderWithPaths("/nonexistent", "/nonexistent")

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Should have builtin scaffold
	entry, ok := registry.Get(BuiltinScaffoldName)
	if !ok {
		t.Fatal("Expected to find guild-campaign builtin")
	}

	if !entry.Builtin {
		t.Error("Expected entry to be marked as builtin")
	}

	if entry.Source != "builtin" {
		t.Errorf("Expected source 'builtin', got '%s'", entry.Source)
	}
}

func TestRegistryLoader_WithoutBuiltins(t *testing.T) {
	loader := NewRegistryLoaderWithPaths("/nonexistent", "/nonexistent").
		WithBuiltins(false)

	ctx := context.Background()
	registry, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	// Should NOT have builtin scaffold
	if _, ok := registry.Get(BuiltinScaffoldName); ok {
		t.Error("Expected no builtin scaffolds when disabled")
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
	fsys := GetBuiltinFS()
	if fsys == nil {
		t.Fatal("Expected non-nil filesystem")
	}

	// Check guild-campaign exists
	entries, err := fs.ReadDir(fsys, "guild-campaign")
	if err != nil {
		t.Fatalf("Failed to read guild-campaign dir: %v", err)
	}

	// Should have scaffold.yaml and templates
	hasScaffoldYaml := false
	hasTemplates := false
	for _, entry := range entries {
		if entry.Name() == "scaffold.yaml" {
			hasScaffoldYaml = true
		}
		if entry.Name() == "templates" {
			hasTemplates = true
		}
	}

	if !hasScaffoldYaml {
		t.Error("Expected scaffold.yaml in guild-campaign")
	}
	if !hasTemplates {
		t.Error("Expected templates dir in guild-campaign")
	}
}

func TestResolveScaffold_Builtin(t *testing.T) {
	entry := ScaffoldEntry{
		Name:    BuiltinScaffoldName,
		Builtin: true,
		Source:  "builtin",
	}

	ctx := context.Background()
	def, fsys, err := ResolveScaffold(ctx, entry)
	if err != nil {
		t.Fatalf("Failed to resolve builtin scaffold: %v", err)
	}

	if def.Name != BuiltinScaffoldName {
		t.Errorf("Expected name '%s', got '%s'", BuiltinScaffoldName, def.Name)
	}

	if fsys == nil {
		t.Error("Expected non-nil filesystem")
	}

	// Check we can read templates from the filesystem
	_, err = fs.ReadDir(fsys, "templates")
	if err != nil {
		t.Errorf("Failed to read templates dir: %v", err)
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

func TestFindScaffold(t *testing.T) {
	loader := NewRegistryLoaderWithPaths("/nonexistent", "/nonexistent")

	ctx := context.Background()

	// Should find builtin
	entry, err := loader.FindScaffold(ctx, BuiltinScaffoldName)
	if err != nil {
		t.Fatalf("Failed to find builtin scaffold: %v", err)
	}
	if entry.Name != BuiltinScaffoldName {
		t.Errorf("Expected name '%s', got '%s'", BuiltinScaffoldName, entry.Name)
	}

	// Should not find nonexistent
	_, err = loader.FindScaffold(ctx, "nonexistent-scaffold")
	if err == nil {
		t.Error("Expected error for nonexistent scaffold")
	}
}
