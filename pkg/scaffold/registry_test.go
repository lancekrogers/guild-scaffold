// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestRegistryConfig_Parse(t *testing.T) {
	yamlContent := `
scaffolds:
  - name: modular-justfile
    path: ~/.guild/scaffolds/justfile-system/
    description: "Modular justfile structure"
    category: utility
  - name: go-cli
    path: /path/to/scaffolds/go-cli/
    description: "Go CLI application"
    category: project
`

	var config RegistryConfig
	err := yaml.Unmarshal([]byte(yamlContent), &config)
	if err != nil {
		t.Fatalf("Failed to parse registry config: %v", err)
	}

	if len(config.Scaffolds) != 2 {
		t.Errorf("Expected 2 scaffolds, got %d", len(config.Scaffolds))
	}

	// Check first entry
	if config.Scaffolds[0].Name != "modular-justfile" {
		t.Errorf("Expected name 'modular-justfile', got '%s'", config.Scaffolds[0].Name)
	}
	if config.Scaffolds[0].Category != "utility" {
		t.Errorf("Expected category 'utility', got '%s'", config.Scaffolds[0].Category)
	}
}

func TestScaffoldDefinition_Parse(t *testing.T) {
	yamlContent := `
name: modular-justfile
version: "1.0"
description: "Modular justfile system with categorized recipes"

variables:
  project_name:
    type: string
    required: true
    description: "Name of the project"
  categories:
    type: array
    default: [build, test, dev]
    description: "Justfile categories to create"

tree:
  justfile: justfile.tmpl
  justfiles/:
    build.just: build.just.tmpl
    test.just: test.just.tmpl
`

	var def ScaffoldDefinition
	err := yaml.Unmarshal([]byte(yamlContent), &def)
	if err != nil {
		t.Fatalf("Failed to parse scaffold definition: %v", err)
	}

	if def.Name != "modular-justfile" {
		t.Errorf("Expected name 'modular-justfile', got '%s'", def.Name)
	}

	if def.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", def.Version)
	}

	// Check variables
	if len(def.Variables) != 2 {
		t.Errorf("Expected 2 variables, got %d", len(def.Variables))
	}

	projectName, ok := def.Variables["project_name"]
	if !ok {
		t.Error("Expected project_name variable")
	} else {
		if projectName.Type != "string" {
			t.Errorf("Expected type 'string', got '%s'", projectName.Type)
		}
		if !projectName.Required {
			t.Error("Expected project_name to be required")
		}
	}

	// Check tree
	if len(def.Tree) != 2 {
		t.Errorf("Expected 2 tree entries, got %d", len(def.Tree))
	}
}

func TestRegistry_AddAndGet(t *testing.T) {
	registry := NewRegistry()

	entry := ScaffoldEntry{
		Name:        "test-scaffold",
		Path:        "/path/to/scaffold",
		Description: "Test scaffold",
		Source:      "global",
	}

	registry.Add(entry)

	got, ok := registry.Get("test-scaffold")
	if !ok {
		t.Fatal("Expected to find test-scaffold")
	}

	if got.Name != entry.Name {
		t.Errorf("Expected name '%s', got '%s'", entry.Name, got.Name)
	}
	if got.Path != entry.Path {
		t.Errorf("Expected path '%s', got '%s'", entry.Path, got.Path)
	}
}

func TestRegistry_Merge(t *testing.T) {
	global := NewRegistry()
	global.Add(ScaffoldEntry{Name: "shared", Path: "/global/shared", Source: "global"})
	global.Add(ScaffoldEntry{Name: "global-only", Path: "/global/only", Source: "global"})

	project := NewRegistry()
	project.Add(ScaffoldEntry{Name: "shared", Path: "/project/shared", Source: "project"})
	project.Add(ScaffoldEntry{Name: "project-only", Path: "/project/only", Source: "project"})

	// Merge project into global (project takes precedence)
	global.Merge(project)

	// Check shared was overwritten by project
	shared, ok := global.Get("shared")
	if !ok {
		t.Fatal("Expected to find shared")
	}
	if shared.Path != "/project/shared" {
		t.Errorf("Expected project path to take precedence, got '%s'", shared.Path)
	}
	if shared.Source != "project" {
		t.Errorf("Expected source 'project', got '%s'", shared.Source)
	}

	// Check both unique entries exist
	if _, ok := global.Get("global-only"); !ok {
		t.Error("Expected to find global-only")
	}
	if _, ok := global.Get("project-only"); !ok {
		t.Error("Expected to find project-only")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()
	registry.Add(ScaffoldEntry{Name: "a", Path: "/a"})
	registry.Add(ScaffoldEntry{Name: "b", Path: "/b"})
	registry.Add(ScaffoldEntry{Name: "c", Path: "/c"})

	list := registry.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(list))
	}
}

func TestLoadRegistryFromFile(t *testing.T) {
	// Create temp directory and registry file
	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "scaffold.yaml")

	content := `
scaffolds:
  - name: test-scaffold
    path: /test/path
    description: "Test scaffold"
`
	if err := os.WriteFile(registryPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ctx := context.Background()
	registry, err := LoadRegistryFromFile(ctx, registryPath)
	if err != nil {
		t.Fatalf("Failed to load registry: %v", err)
	}

	entry, ok := registry.Get("test-scaffold")
	if !ok {
		t.Fatal("Expected to find test-scaffold")
	}
	if entry.Description != "Test scaffold" {
		t.Errorf("Expected description 'Test scaffold', got '%s'", entry.Description)
	}
}

func TestLoadRegistryFromFile_NotFound(t *testing.T) {
	ctx := context.Background()
	registry, err := LoadRegistryFromFile(ctx, "/nonexistent/path/scaffold.yaml")
	if err != nil {
		t.Fatalf("Expected no error for missing file, got: %v", err)
	}

	// Should return empty registry
	if len(registry.List()) != 0 {
		t.Errorf("Expected empty registry, got %d entries", len(registry.List()))
	}
}

func TestValidateScaffoldEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   ScaffoldEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: ScaffoldEntry{
				Name: "test",
				Path: "/path",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			entry: ScaffoldEntry{
				Path: "/path",
			},
			wantErr: true,
		},
		{
			name: "missing path (non-builtin)",
			entry: ScaffoldEntry{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "builtin without path is valid",
			entry: ScaffoldEntry{
				Name:    "test",
				Builtin: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScaffoldEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScaffoldEntry() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateScaffoldDefinition(t *testing.T) {
	tests := []struct {
		name    string
		def     *ScaffoldDefinition
		wantErr bool
	}{
		{
			name: "valid with tree",
			def: &ScaffoldDefinition{
				Name: "test",
				Tree: map[string]any{"file": "template"},
			},
			wantErr: false,
		},
		{
			name: "valid with files",
			def: &ScaffoldDefinition{
				Name:  "test",
				Files: []FileEntry{{Path: "file", Template: "tmpl"}},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			def: &ScaffoldDefinition{
				Tree: map[string]any{"file": "template"},
			},
			wantErr: true,
		},
		{
			name: "missing tree and files",
			def: &ScaffoldDefinition{
				Name: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScaffoldDefinition(tt.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateScaffoldDefinition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/test", filepath.Join(home, "test")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandPath(tt.input)
			if got != tt.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
