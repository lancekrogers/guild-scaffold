// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scaffold

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Registry file locations
const (
	// GlobalRegistryPath is the default path for the global scaffold registry
	GlobalRegistryPath = "~/.guild/scaffold.yaml"

	// ProjectRegistryPath is the path for project-level scaffold registry
	ProjectRegistryPath = ".campaign/scaffold.yaml"

	// BuiltinScaffoldName is the name of the embedded guild-campaign scaffold
	BuiltinScaffoldName = "guild-campaign"
)

// RegistryConfig represents a scaffold registry configuration file.
// Found at ~/.guild/scaffold.yaml (global) or .campaign/scaffold.yaml (project)
//
// Example:
//
//	scaffolds:
//	  - name: modular-justfile
//	    path: ~/.guild/scaffolds/justfile-system/
//	    description: "Modular justfile structure"
//	  - name: go-cli
//	    path: /path/to/scaffolds/go-cli/
//	    description: "Go CLI application"
type RegistryConfig struct {
	// Scaffolds is the list of available scaffold definitions
	Scaffolds []ScaffoldEntry `yaml:"scaffolds" json:"scaffolds"`
}

// ScaffoldEntry represents a single scaffold in the registry.
// Points to a directory containing a scaffold.yaml definition.
type ScaffoldEntry struct {
	// Name is the unique identifier for this scaffold (used with --template flag)
	Name string `yaml:"name" json:"name"`

	// Path is the filesystem path to the scaffold directory (contains scaffold.yaml)
	// Supports ~ expansion for home directory
	Path string `yaml:"path" json:"path"`

	// Description provides a human-readable description of this scaffold
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Category groups related scaffolds (e.g., "workspace", "agent", "utility")
	Category string `yaml:"category,omitempty" json:"category,omitempty"`

	// Source indicates where this entry came from (for debugging/display)
	// Values: "builtin", "global", "project"
	Source string `yaml:"-" json:"source,omitempty"`

	// Builtin indicates this is an embedded scaffold (path is ignored)
	Builtin bool `yaml:"-" json:"builtin,omitempty"`
}

// ScaffoldDefinition represents a scaffold's scaffold.yaml file.
// This is the definition of what the scaffold creates, not the registry entry.
//
// Example:
//
//	name: modular-justfile
//	version: "1.0"
//	description: "Modular justfile system with categorized recipes"
//
//	variables:
//	  project_name:
//	    type: string
//	    required: true
//	  categories:
//	    type: array
//	    default: [build, test, dev]
//
//	tree:
//	  justfile: justfile.tmpl
//	  justfiles/:
//	    _empty: true
type ScaffoldDefinition struct {
	// Name is the scaffold identifier (should match registry entry)
	Name string `yaml:"name" json:"name"`

	// Version is the scaffold version (semver recommended)
	Version string `yaml:"version" json:"version"`

	// Description provides detailed scaffold documentation
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Variables defines the configurable variables for this scaffold
	Variables map[string]VariableDefinition `yaml:"variables,omitempty" json:"variables,omitempty"`

	// Tree defines the directory/file structure to create
	// This is the tree-format structure (see tree_parser.go)
	Tree map[string]any `yaml:"tree,omitempty" json:"tree,omitempty"`

	// Files is an alternative to Tree - explicit file list (legacy format)
	// Use Tree for new scaffolds
	Files []FileEntry `yaml:"files,omitempty" json:"files,omitempty"`

	// TemplatesDir specifies where templates are located relative to scaffold.yaml
	TemplatesDir string `yaml:"templates_dir,omitempty" json:"templates_dir,omitempty"`
}

// VariableDefinition describes a scaffold variable's type and constraints.
type VariableDefinition struct {
	// Type is the variable type: string, int, bool, array, object
	Type string `yaml:"type" json:"type"`

	// Required indicates the variable must be provided
	Required bool `yaml:"required,omitempty" json:"required,omitempty"`

	// Default is the default value if not provided
	Default any `yaml:"default,omitempty" json:"default,omitempty"`

	// Description explains the variable's purpose
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Enum restricts string values to a specific set
	Enum []string `yaml:"enum,omitempty" json:"enum,omitempty"`

	// Pattern is a regex pattern for string validation
	Pattern string `yaml:"pattern,omitempty" json:"pattern,omitempty"`

	// Min/Max for numeric types
	Min *float64 `yaml:"min,omitempty" json:"min,omitempty"`
	Max *float64 `yaml:"max,omitempty" json:"max,omitempty"`
}

// Registry provides access to scaffold entries from multiple sources.
// It merges global and project registries with project taking precedence.
type Registry struct {
	// entries maps scaffold name to entry
	entries map[string]ScaffoldEntry

	// sources tracks where entries came from for debugging
	sources []string
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]ScaffoldEntry),
		sources: make([]string, 0),
	}
}

// Add adds or updates a scaffold entry.
// If an entry with the same name exists, it is replaced.
func (r *Registry) Add(entry ScaffoldEntry) {
	r.entries[entry.Name] = entry
}

// Get retrieves a scaffold entry by name.
// Returns the entry and true if found, zero value and false otherwise.
func (r *Registry) Get(name string) (ScaffoldEntry, bool) {
	entry, ok := r.entries[name]
	return entry, ok
}

// List returns all scaffold entries.
func (r *Registry) List() []ScaffoldEntry {
	result := make([]ScaffoldEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		result = append(result, entry)
	}
	return result
}

// Merge combines another registry into this one.
// Entries from the other registry take precedence on name conflicts.
func (r *Registry) Merge(other *Registry) {
	for name, entry := range other.entries {
		r.entries[name] = entry
	}
	r.sources = append(r.sources, other.sources...)
}

// LoadRegistryFromFile loads a registry from a YAML file.
func LoadRegistryFromFile(ctx context.Context, path string) (*Registry, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	// Expand ~ to home directory
	expandedPath := expandPath(path)

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Not an error - registry file is optional
			return NewRegistry(), nil
		}
		return nil, fmt.Errorf("failed to read registry file (path=%v): %w", expandedPath, err)
	}

	var config RegistryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse registry file (path=%v): %w", expandedPath, err)
	}

	registry := NewRegistry()
	registry.sources = append(registry.sources, expandedPath)

	// Determine source based on path
	source := "project"
	if path == GlobalRegistryPath || expandedPath == expandPath(GlobalRegistryPath) {
		source = "global"
	}

	for _, entry := range config.Scaffolds {
		entry.Source = source
		entry.Path = expandPath(entry.Path)
		registry.Add(entry)
	}

	return registry, nil
}

// LoadScaffoldDefinition loads a scaffold definition from a directory.
// The directory must contain a scaffold.yaml file.
func LoadScaffoldDefinition(ctx context.Context, scaffoldDir string) (*ScaffoldDefinition, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	scaffoldPath := filepath.Join(scaffoldDir, "scaffold.yaml")
	data, err := os.ReadFile(scaffoldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scaffold definition (path=%v): %w", scaffoldPath, err)
	}

	var def ScaffoldDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse scaffold definition (path=%v): %w", scaffoldPath, err)
	}

	return &def, nil
}

// LoadScaffoldDefinitionFromFS loads a scaffold definition from an embedded filesystem.
func LoadScaffoldDefinitionFromFS(ctx context.Context, fsys fs.FS, scaffoldPath string) (*ScaffoldDefinition, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	data, err := fs.ReadFile(fsys, scaffoldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scaffold definition (path=%v): %w", scaffoldPath, err)
	}

	var def ScaffoldDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse scaffold definition (path=%v): %w", scaffoldPath, err)
	}

	return &def, nil
}

// ValidateScaffoldEntry validates a scaffold entry has required fields.
func ValidateScaffoldEntry(entry ScaffoldEntry) error {
	if entry.Name == "" {
		return fmt.Errorf("scaffold entry missing required field: name")
	}
	if entry.Path == "" && !entry.Builtin {
		return fmt.Errorf("scaffold entry missing required field (path, name=%v)", entry.Name)
	}
	return nil
}

// ValidateScaffoldDefinition validates a scaffold definition has required fields.
func ValidateScaffoldDefinition(def *ScaffoldDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("scaffold definition missing required field: name")
	}
	if len(def.Tree) == 0 && len(def.Files) == 0 {
		return fmt.Errorf("scaffold definition must have tree or files: name=%v", def.Name)
	}
	return nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) == 0 {
		return path
	}
	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
