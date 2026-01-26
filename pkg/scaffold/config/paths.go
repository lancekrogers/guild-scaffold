// Copyright (c) 2025 Lance Rogers
// SPDX-License-Identifier: MIT

// Package config provides configuration directory management for guild-scaffold.
// It implements a two-tier template system:
//   - Global templates: ~/.config/guild/templates/
//   - Workspace templates: .campaign/templates/ (takes precedence)
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Environment variable names for configuration overrides
const (
	// EnvGlobalDir overrides the global configuration directory
	// Default: ~/.config/guild/ (or XDG_CONFIG_HOME/guild/)
	EnvGlobalDir = "GUILD_SCAFFOLD_GLOBAL_DIR"

	// EnvWorkspaceDir overrides the workspace directory name
	// Default: .campaign
	EnvWorkspaceDir = "GUILD_SCAFFOLD_WORKSPACE_DIR"
)

// Directory names and paths
const (
	// DefaultConfigDirName is the default config directory under XDG config
	DefaultConfigDirName = "guild"

	// TemplatesDirName is the name of the templates subdirectory
	TemplatesDirName = "templates"

	// ConfigFileName is the name of the main config file
	ConfigFileName = "config.yaml"

	// RegistryFileName is the name of the template registry file
	RegistryFileName = "registry.yaml"

	// DefaultWorkspaceDirName is the default workspace directory name
	DefaultWorkspaceDirName = ".campaign"
)

// ConfigDirName returns the effective config directory name.
// Uses GUILD_SCAFFOLD_GLOBAL_DIR if set, otherwise returns default.
// Deprecated: Use PathResolver.GlobalConfigDir() for full path resolution.
var ConfigDirName = DefaultConfigDirName

// WorkspaceDirName returns the effective workspace directory name.
// Uses GUILD_SCAFFOLD_WORKSPACE_DIR if set, otherwise returns default.
// Deprecated: Use PathResolver.WorkspaceDir() for full path resolution.
var WorkspaceDirName = DefaultWorkspaceDirName

func init() {
	// Note: These are kept for backward compatibility with code that references
	// the constants directly. New code should use PathResolver methods.
}

// PathResolver provides path resolution for scaffold configuration.
// It abstracts filesystem access for testability.
//
// Path resolution supports environment variable overrides:
//   - GUILD_SCAFFOLD_GLOBAL_DIR: Override the entire global config directory
//   - GUILD_SCAFFOLD_WORKSPACE_DIR: Override the workspace directory name
//   - XDG_CONFIG_HOME: Standard XDG override for config location
type PathResolver struct {
	// homeDir is the user's home directory
	homeDir string

	// workDir is the current working directory
	workDir string

	// xdgConfigHome overrides XDG_CONFIG_HOME if set
	xdgConfigHome string

	// globalDirOverride overrides the entire global directory if set
	globalDirOverride string

	// workspaceDirName overrides the workspace directory name if set
	workspaceDirName string
}

// NewPathResolver creates a new path resolver with system defaults.
// Reads configuration from environment variables:
//   - GUILD_SCAFFOLD_GLOBAL_DIR: Override entire global directory
//   - GUILD_SCAFFOLD_WORKSPACE_DIR: Override workspace directory name
//   - XDG_CONFIG_HOME: Standard XDG config home override
func NewPathResolver() (*PathResolver, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Read workspace dir override, default to .campaign
	workspaceDir := os.Getenv(EnvWorkspaceDir)
	if workspaceDir == "" {
		workspaceDir = DefaultWorkspaceDirName
	}

	return &PathResolver{
		homeDir:           home,
		workDir:           wd,
		xdgConfigHome:     os.Getenv("XDG_CONFIG_HOME"),
		globalDirOverride: os.Getenv(EnvGlobalDir),
		workspaceDirName:  workspaceDir,
	}, nil
}

// NewPathResolverWithPaths creates a path resolver with explicit paths for testing.
// Deprecated: Use NewPathResolverWithConfig for full configuration control.
func NewPathResolverWithPaths(homeDir, workDir, xdgConfigHome string) *PathResolver {
	return &PathResolver{
		homeDir:          homeDir,
		workDir:          workDir,
		xdgConfigHome:    xdgConfigHome,
		workspaceDirName: DefaultWorkspaceDirName,
	}
}

// PathResolverConfig holds configuration for creating a PathResolver.
type PathResolverConfig struct {
	// HomeDir is the user's home directory
	HomeDir string

	// WorkDir is the current working directory
	WorkDir string

	// XDGConfigHome overrides XDG_CONFIG_HOME
	XDGConfigHome string

	// GlobalDirOverride overrides the entire global directory path
	GlobalDirOverride string

	// WorkspaceDirName overrides the workspace directory name (default: .campaign)
	WorkspaceDirName string
}

// NewPathResolverWithConfig creates a path resolver with full configuration control.
func NewPathResolverWithConfig(cfg PathResolverConfig) *PathResolver {
	workspaceDir := cfg.WorkspaceDirName
	if workspaceDir == "" {
		workspaceDir = DefaultWorkspaceDirName
	}

	return &PathResolver{
		homeDir:           cfg.HomeDir,
		workDir:           cfg.WorkDir,
		xdgConfigHome:     cfg.XDGConfigHome,
		globalDirOverride: cfg.GlobalDirOverride,
		workspaceDirName:  workspaceDir,
	}
}

// GlobalConfigDir returns the global configuration directory.
// Resolution order:
//  1. GUILD_SCAFFOLD_GLOBAL_DIR environment variable (if set)
//  2. XDG_CONFIG_HOME/guild/ (if XDG_CONFIG_HOME is set)
//  3. ~/.config/guild/ (default)
func (p *PathResolver) GlobalConfigDir() string {
	// Check for explicit override first
	if p.globalDirOverride != "" {
		return expandHome(p.globalDirOverride, p.homeDir)
	}

	// Fall back to XDG-compliant path
	base := p.xdgConfigHome
	if base == "" {
		base = filepath.Join(p.homeDir, ".config")
	}
	return filepath.Join(base, DefaultConfigDirName)
}

// expandHome expands ~ to the home directory in a path.
func expandHome(path, homeDir string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(homeDir, path[1:])
	}
	return path
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
// Uses GUILD_SCAFFOLD_WORKSPACE_DIR if set, otherwise defaults to .campaign/
// relative to workDir.
func (p *PathResolver) WorkspaceDir() string {
	return filepath.Join(p.workDir, p.workspaceDirName)
}

// WorkspaceDirNameValue returns the configured workspace directory name.
func (p *PathResolver) WorkspaceDirNameValue() string {
	return p.workspaceDirName
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
// Returns the path and source ("workspace" or "global")
// Returns ErrNotFound if template doesn't exist in either location.
func (p *PathResolver) ResolveTemplatePath(ctx context.Context, name string) (path string, source string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", fmt.Errorf("context cancelled: %w", err)
	}

	if name == "" {
		return "", "", fmt.Errorf("template name cannot be empty")
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

	return "", "", fmt.Errorf("template not found: name=%s, searched=%v", name, []string{workspacePath, globalPath})
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
		return fmt.Errorf("context cancelled: %w", err)
	}

	dirs := []string{
		p.GlobalConfigDir(),
		p.GlobalTemplatesDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory (path=%v): %w", dir, err)
		}
	}

	return nil
}

// ListTemplates returns all available templates from both workspace and global.
// Workspace templates take precedence over global templates with the same name.
func (p *PathResolver) ListTemplates(ctx context.Context) ([]TemplateInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
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
