// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package scaffold

import (
	"context"
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

//go:embed builtin/*
var builtinFS embed.FS

// RegistryLoader loads and merges scaffold registries from multiple sources.
type RegistryLoader struct {
	// workDir is the current working directory for project registry lookup
	workDir string

	// homeDir is the user's home directory for global registry lookup
	homeDir string

	// includeBuiltins controls whether builtin scaffolds are included
	includeBuiltins bool
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

	return &RegistryLoader{
		workDir:         wd,
		homeDir:         home,
		includeBuiltins: true,
	}, nil
}

// NewRegistryLoaderWithPaths creates a registry loader with explicit paths.
func NewRegistryLoaderWithPaths(workDir, homeDir string) *RegistryLoader {
	return &RegistryLoader{
		workDir:         workDir,
		homeDir:         homeDir,
		includeBuiltins: true,
	}
}

// WithBuiltins controls whether builtin scaffolds are included.
func (l *RegistryLoader) WithBuiltins(include bool) *RegistryLoader {
	l.includeBuiltins = include
	return l
}

// Load loads and merges all scaffold registries.
// Precedence (highest to lowest):
// 1. Project registry (.campaign/scaffold.yaml)
// 2. Global registry (~/.guild/scaffold.yaml)
// 3. Builtin scaffolds (guild-campaign)
func (l *RegistryLoader) Load(ctx context.Context) (*Registry, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	registry := NewRegistry()

	// 1. Load builtins first (lowest precedence)
	if l.includeBuiltins {
		builtins := l.loadBuiltins()
		registry.Merge(builtins)
	}

	// 2. Load global registry
	globalPath := filepath.Join(l.homeDir, ".guild", "scaffold.yaml")
	global, err := LoadRegistryFromFile(ctx, globalPath)
	if err != nil {
		// Log warning but continue
		// TODO: add logging
	} else {
		registry.Merge(global)
	}

	// 3. Load project registry (highest precedence)
	projectPath := filepath.Join(l.workDir, ".campaign", "scaffold.yaml")
	project, err := LoadRegistryFromFile(ctx, projectPath)
	if err != nil {
		// Log warning but continue
	} else {
		registry.Merge(project)
	}

	return registry, nil
}

// loadBuiltins returns the registry of builtin scaffolds.
func (l *RegistryLoader) loadBuiltins() *Registry {
	registry := NewRegistry()

	// Add guild-campaign builtin
	registry.Add(ScaffoldEntry{
		Name:        BuiltinScaffoldName,
		Description: "Complete campaign workspace with guild configuration",
		Category:    "workspace",
		Source:      "builtin",
		Builtin:     true,
	})

	return registry
}

// GetBuiltinFS returns the embedded filesystem containing builtin scaffolds.
func GetBuiltinFS() fs.FS {
	sub, err := fs.Sub(builtinFS, "builtin")
	if err != nil {
		// This should never happen with valid embed
		return builtinFS
	}
	return sub
}

// ResolveScaffold resolves a scaffold entry to its definition and filesystem.
// For builtin scaffolds, returns the embedded filesystem.
// For external scaffolds, returns an OS-based filesystem.
func ResolveScaffold(ctx context.Context, entry ScaffoldEntry) (*ScaffoldDefinition, fs.FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	if entry.Builtin {
		return resolveBuiltinScaffold(ctx, entry)
	}

	return resolveExternalScaffold(ctx, entry)
}

// resolveBuiltinScaffold resolves a builtin scaffold.
func resolveBuiltinScaffold(ctx context.Context, entry ScaffoldEntry) (*ScaffoldDefinition, fs.FS, error) {
	builtinFSys := GetBuiltinFS()

	// Load definition from builtin filesystem
	scaffoldPath := filepath.Join(entry.Name, "scaffold.yaml")
	def, err := LoadScaffoldDefinitionFromFS(ctx, builtinFSys, scaffoldPath)
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeNotFound, "builtin scaffold not found").
			WithDetails("name", entry.Name)
	}

	// Return a sub-filesystem for this scaffold
	subFS, err := fs.Sub(builtinFSys, entry.Name)
	if err != nil {
		return nil, nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to access builtin scaffold").
			WithDetails("name", entry.Name)
	}

	return def, subFS, nil
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
