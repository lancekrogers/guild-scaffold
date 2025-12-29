// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

// Package config provides configuration directory management for guild-scaffold.
// It implements a two-tier template system:
//   - Global templates: ~/.config/guild/templates/
//   - Workspace templates: .campaign/templates/ (takes precedence)
package config

import (
	"context"
	"os"
	"path/filepath"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Directory names and paths
const (
	// ConfigDirName is the name of the config directory under XDG config
	ConfigDirName = "guild"

	// TemplatesDirName is the name of the templates subdirectory
	TemplatesDirName = "templates"

	// ConfigFileName is the name of the main config file
	ConfigFileName = "config.yaml"

	// RegistryFileName is the name of the template registry file
	RegistryFileName = "registry.yaml"

	// WorkspaceDirName is the workspace config directory name
	WorkspaceDirName = ".campaign"
)

// PathResolver provides path resolution for scaffold configuration.
// It abstracts filesystem access for testability.
type PathResolver struct {
	// homeDir is the user's home directory
	homeDir string

	// workDir is the current working directory
	workDir string

	// xdgConfigHome overrides XDG_CONFIG_HOME if set
	xdgConfigHome string
}

// NewPathResolver creates a new path resolver with system defaults.
func NewPathResolver() (*PathResolver, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to get home directory")
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to get working directory")
	}

	return &PathResolver{
		homeDir:       home,
		workDir:       wd,
		xdgConfigHome: os.Getenv("XDG_CONFIG_HOME"),
	}, nil
}

// NewPathResolverWithPaths creates a path resolver with explicit paths for testing.
func NewPathResolverWithPaths(homeDir, workDir, xdgConfigHome string) *PathResolver {
	return &PathResolver{
		homeDir:       homeDir,
		workDir:       workDir,
		xdgConfigHome: xdgConfigHome,
	}
}

// GlobalConfigDir returns the global configuration directory.
// Uses XDG_CONFIG_HOME if set, otherwise defaults to ~/.config/guild/
func (p *PathResolver) GlobalConfigDir() string {
	base := p.xdgConfigHome
	if base == "" {
		base = filepath.Join(p.homeDir, ".config")
	}
	return filepath.Join(base, ConfigDirName)
}

// GlobalTemplatesDir returns the global templates directory.
// Returns ~/.config/guild/templates/ (or XDG equivalent)
func (p *PathResolver) GlobalTemplatesDir() string {
	return filepath.Join(p.GlobalConfigDir(), TemplatesDirName)
}

// GlobalConfigFile returns the path to the global config file.
// Returns ~/.config/guild/config.yaml
func (p *PathResolver) GlobalConfigFile() string {
	return filepath.Join(p.GlobalConfigDir(), ConfigFileName)
}

// GlobalRegistryFile returns the path to the global registry file.
// Returns ~/.config/guild/templates/registry.yaml
func (p *PathResolver) GlobalRegistryFile() string {
	return filepath.Join(p.GlobalTemplatesDir(), RegistryFileName)
}

// WorkspaceDir returns the workspace configuration directory.
// Returns .campaign/ relative to workDir
func (p *PathResolver) WorkspaceDir() string {
	return filepath.Join(p.workDir, WorkspaceDirName)
}

// WorkspaceTemplatesDir returns the workspace templates directory.
// Returns .campaign/templates/
func (p *PathResolver) WorkspaceTemplatesDir() string {
	return filepath.Join(p.WorkspaceDir(), TemplatesDirName)
}

// WorkspaceRegistryFile returns the path to the workspace registry file.
// Returns .campaign/templates/registry.yaml
func (p *PathResolver) WorkspaceRegistryFile() string {
	return filepath.Join(p.WorkspaceTemplatesDir(), RegistryFileName)
}

// ResolveTemplatePath resolves a template name to its filesystem path.
// Resolution order:
//  1. Workspace templates (.campaign/templates/<name>/)
//  2. Global templates (~/.config/guild/templates/<name>/)
//
// Returns the path and source ("workspace" or "global").
// Returns ErrNotFound if template doesn't exist in either location.
func (p *PathResolver) ResolveTemplatePath(ctx context.Context, name string) (path string, source string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	if name == "" {
		return "", "", gerror.New(gerror.ErrCodeValidation, "template name cannot be empty", nil)
	}

	// Check workspace first (highest precedence)
	workspacePath := filepath.Join(p.WorkspaceTemplatesDir(), name)
	if p.isValidTemplateDir(workspacePath) {
		return workspacePath, "workspace", nil
	}

	// Check global templates
	globalPath := filepath.Join(p.GlobalTemplatesDir(), name)
	if p.isValidTemplateDir(globalPath) {
		return globalPath, "global", nil
	}

	return "", "", gerror.New(gerror.ErrCodeNotFound, "template not found", nil).
		WithDetails("name", name).
		WithDetails("searched", []string{workspacePath, globalPath})
}

// isValidTemplateDir checks if a path is a valid template directory.
// A valid template directory exists and contains a scaffold.yaml file.
func (p *PathResolver) isValidTemplateDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check for scaffold.yaml
	scaffoldFile := filepath.Join(path, "scaffold.yaml")
	_, err = os.Stat(scaffoldFile)
	return err == nil
}

// EnsureGlobalDirs creates the global configuration directories if they don't exist.
func (p *PathResolver) EnsureGlobalDirs(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	dirs := []string{
		p.GlobalConfigDir(),
		p.GlobalTemplatesDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeIO, "failed to create directory").
				WithDetails("path", dir)
		}
	}

	return nil
}

// ListTemplates returns all available templates from both workspace and global.
// Workspace templates take precedence over global templates with the same name.
func (p *PathResolver) ListTemplates(ctx context.Context) ([]TemplateInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	templates := make(map[string]TemplateInfo)

	// Load global templates first (lower precedence)
	globalTemplates, _ := p.listTemplatesInDir(p.GlobalTemplatesDir(), "global")
	for _, t := range globalTemplates {
		templates[t.Name] = t
	}

	// Load workspace templates (higher precedence, overwrites global)
	workspaceTemplates, _ := p.listTemplatesInDir(p.WorkspaceTemplatesDir(), "workspace")
	for _, t := range workspaceTemplates {
		templates[t.Name] = t
	}

	// Convert map to slice
	result := make([]TemplateInfo, 0, len(templates))
	for _, t := range templates {
		result = append(result, t)
	}

	return result, nil
}

// listTemplatesInDir lists all templates in a directory.
func (p *PathResolver) listTemplatesInDir(dir, source string) ([]TemplateInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var templates []TemplateInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a valid template directory
		templatePath := filepath.Join(dir, entry.Name())
		if p.isValidTemplateDir(templatePath) {
			templates = append(templates, TemplateInfo{
				Name:   entry.Name(),
				Path:   templatePath,
				Source: source,
			})
		}
	}

	return templates, nil
}

// TemplateInfo contains information about an available template.
type TemplateInfo struct {
	// Name is the template name (directory name)
	Name string

	// Path is the full filesystem path
	Path string

	// Source indicates where the template came from: "workspace" or "global"
	Source string
}

// Convenience functions for common operations

// GlobalConfigDir returns the global config directory using system defaults.
func GlobalConfigDir() (string, error) {
	p, err := NewPathResolver()
	if err != nil {
		return "", err
	}
	return p.GlobalConfigDir(), nil
}

// GlobalTemplatesDir returns the global templates directory using system defaults.
func GlobalTemplatesDir() (string, error) {
	p, err := NewPathResolver()
	if err != nil {
		return "", err
	}
	return p.GlobalTemplatesDir(), nil
}

// WorkspaceTemplatesDir returns the workspace templates directory using system defaults.
func WorkspaceTemplatesDir() (string, error) {
	p, err := NewPathResolver()
	if err != nil {
		return "", err
	}
	return p.WorkspaceTemplatesDir(), nil
}

// ResolveTemplatePath resolves a template name using system defaults.
func ResolveTemplatePath(ctx context.Context, name string) (path string, source string, err error) {
	p, err := NewPathResolver()
	if err != nil {
		return "", "", err
	}
	return p.ResolveTemplatePath(ctx, name)
}
