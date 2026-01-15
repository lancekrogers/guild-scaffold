// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scaffold

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-scaffold/pkg/scaffold/config"
)

// RegistryLoader loads and merges scaffold registries from multiple sources.
type RegistryLoader struct {
	// workDir is the current working directory for project registry lookup
	workDir string

	// homeDir is the user's home directory for global registry lookup
	homeDir string

	// includeBuiltins controls whether builtin scaffolds are included
	// (now loads from config directory instead of embedded)
	includeBuiltins bool

	// paths is the path resolver for template locations
	paths *config.PathResolver
}

// NewRegistryLoader creates a new registry loader with default settings.
func NewRegistryLoader() (*RegistryLoader, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to get home directory")
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to get working directory")
	}

	paths, err := config.NewPathResolver()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create path resolver")
	}

	return &RegistryLoader{
		workDir:         wd,
		homeDir:         home,
		includeBuiltins: true,
		paths:           paths,
	}, nil
}

// NewRegistryLoaderWithPaths creates a registry loader with explicit paths.
func NewRegistryLoaderWithPaths(workDir, homeDir string) *RegistryLoader {
	// Create a path resolver with the explicit paths for testing
	paths := config.NewPathResolverWithPaths(homeDir, workDir, "")
	return &RegistryLoader{
		workDir:         workDir,
		homeDir:         homeDir,
		includeBuiltins: true,
		paths:           paths,
	}
}

// WithBuiltins controls whether builtin scaffolds are included.
func (l *RegistryLoader) WithBuiltins(include bool) *RegistryLoader {
	l.includeBuiltins = include
	return l
}

// Load loads and merges all scaffold registries.
// Precedence (highest to lowest):
// 1. Workspace templates (.campaign/templates/)
// 2. Global templates (~/.config/guild/templates/)
// 3. Legacy registries (for backwards compatibility)
func (l *RegistryLoader) Load(ctx context.Context) (*Registry, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	registry := NewRegistry()

	// 1. Load global templates (lowest precedence)
	if l.includeBuiltins {
		builtins := l.loadBuiltins()
		registry.Merge(builtins)
	}

	// 2. Load workspace templates (higher precedence)
	workspace := l.loadWorkspaceTemplates()
	registry.Merge(workspace)

	// 3. Legacy: Load global registry file (~/.guild/scaffold.yaml)
	// This is kept for backwards compatibility with existing configurations
	globalPath := filepath.Join(l.homeDir, ".guild", "scaffold.yaml")
	global, err := LoadRegistryFromFile(ctx, globalPath)
	if err == nil {
		registry.Merge(global)
	}

	// 4. Legacy: Load project registry (.campaign/scaffold.yaml)
	// This is kept for backwards compatibility
	projectPath := filepath.Join(l.workDir, ".campaign", "scaffold.yaml")
	project, err := LoadRegistryFromFile(ctx, projectPath)
	if err == nil {
		registry.Merge(project)
	}

	return registry, nil
}

// loadWorkspaceTemplates scans the workspace templates directory.
func (l *RegistryLoader) loadWorkspaceTemplates() *Registry {
	registry := NewRegistry()

	if l.paths == nil {
		return registry
	}

	templatesDir := l.paths.WorkspaceTemplatesDir()

	// Check if templates directory exists
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		return registry
	}

	// Scan for scaffold directories
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return registry
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scaffoldPath := filepath.Join(templatesDir, entry.Name(), "scaffold.yaml")
		if _, err := os.Stat(scaffoldPath); err != nil {
			continue
		}

		registry.Add(ScaffoldEntry{
			Name:   entry.Name(),
			Path:   filepath.Join(templatesDir, entry.Name()),
			Source: "workspace",
		})
	}

	return registry
}

// loadBuiltins scans the global templates directory and returns discovered scaffolds.
// This replaces the previous embedded templates pattern.
func (l *RegistryLoader) loadBuiltins() *Registry {
	registry := NewRegistry()

	if l.paths == nil {
		return registry
	}

	templatesDir := l.paths.GlobalTemplatesDir()

	// Check if templates directory exists
	info, err := os.Stat(templatesDir)
	if err != nil || !info.IsDir() {
		// No global templates synced yet - that's fine
		return registry
	}

	// Scan for scaffold directories (each should have a scaffold.yaml)
	entries, err := os.ReadDir(templatesDir)
	if err != nil {
		return registry
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scaffoldPath := filepath.Join(templatesDir, entry.Name(), "scaffold.yaml")
		if _, err := os.Stat(scaffoldPath); err != nil {
			continue // Skip directories without scaffold.yaml
		}

		registry.Add(ScaffoldEntry{
			Name:    entry.Name(),
			Path:    filepath.Join(templatesDir, entry.Name()),
			Source:  "global",
			Builtin: false, // No longer embedded
		})
	}

	return registry
}

// GetGlobalTemplatesFS returns the global templates directory as an fs.FS.
// Returns nil if the directory doesn't exist (templates not synced).
func GetGlobalTemplatesFS() fs.FS {
	paths, err := config.NewPathResolver()
	if err != nil {
		return nil
	}

	templatesDir := paths.GlobalTemplatesDir()
	if _, err := os.Stat(templatesDir); err != nil {
		return nil
	}

	return os.DirFS(templatesDir)
}

// GetBuiltinFS returns the global templates filesystem.
// Deprecated: Use GetGlobalTemplatesFS instead.
func GetBuiltinFS() fs.FS {
	return GetGlobalTemplatesFS()
}

// ResolveScaffold resolves a scaffold entry to its definition and filesystem.
// All scaffolds are now filesystem-based (loaded from config directory).
func ResolveScaffold(ctx context.Context, entry ScaffoldEntry) (*ScaffoldDefinition, fs.FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// All scaffolds are now external (filesystem-based)
	return resolveExternalScaffold(ctx, entry)
}

// resolveExternalScaffold resolves an external (filesystem-based) scaffold.
func resolveExternalScaffold(ctx context.Context, entry ScaffoldEntry) (*ScaffoldDefinition, fs.FS, error) {
	// Verify path exists
	info, err := os.Stat(entry.Path)
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "scaffold path not found").
			WithDetails("path", entry.Path)
	}
	if !info.IsDir() {
		return nil, nil, gerror.New(gerror.ErrCodeValidation, "scaffold path is not a directory", nil).
			WithDetails("path", entry.Path)
	}

	// Load definition
	def, err := LoadScaffoldDefinition(ctx, entry.Path)
	if err != nil {
		return nil, nil, err
	}

	// Return OS filesystem rooted at scaffold path
	return def, os.DirFS(entry.Path), nil
}

// FindScaffold looks up a scaffold by name in the merged registry.
func (l *RegistryLoader) FindScaffold(ctx context.Context, name string) (ScaffoldEntry, error) {
	registry, err := l.Load(ctx)
	if err != nil {
		return ScaffoldEntry{}, err
	}

	entry, ok := registry.Get(name)
	if !ok {
		return ScaffoldEntry{}, gerror.New(gerror.ErrCodeNotFound, "scaffold not found", nil).
			WithDetails("name", name).
			WithDetails("available", registryNames(registry))
	}

	return entry, nil
}

// registryNames returns a slice of all scaffold names in the registry.
func registryNames(r *Registry) []string {
	entries := r.List()
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}
