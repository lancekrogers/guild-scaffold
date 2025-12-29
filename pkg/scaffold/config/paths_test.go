// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathResolver_GlobalConfigDir(t *testing.T) {
	tests := []struct {
		name          string
		homeDir       string
		xdgConfigHome string
		want          string
	}{
		{
			name:          "default uses .config under home",
			homeDir:       "/home/user",
			xdgConfigHome: "",
			want:          "/home/user/.config/guild",
		},
		{
			name:          "respects XDG_CONFIG_HOME",
			homeDir:       "/home/user",
			xdgConfigHome: "/custom/config",
			want:          "/custom/config/guild",
		},
		{
			name:          "XDG takes precedence over home",
			homeDir:       "/home/user",
			xdgConfigHome: "/xdg/config",
			want:          "/xdg/config/guild",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPathResolverWithPaths(tt.homeDir, "/work", tt.xdgConfigHome)
			got := p.GlobalConfigDir()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPathResolver_GlobalTemplatesDir(t *testing.T) {
	p := NewPathResolverWithPaths("/home/user", "/work", "")
	got := p.GlobalTemplatesDir()
	assert.Equal(t, "/home/user/.config/guild/templates", got)
}

func TestPathResolver_GlobalConfigFile(t *testing.T) {
	p := NewPathResolverWithPaths("/home/user", "/work", "")
	got := p.GlobalConfigFile()
	assert.Equal(t, "/home/user/.config/guild/config.yaml", got)
}

func TestPathResolver_GlobalRegistryFile(t *testing.T) {
	p := NewPathResolverWithPaths("/home/user", "/work", "")
	got := p.GlobalRegistryFile()
	assert.Equal(t, "/home/user/.config/guild/templates/registry.yaml", got)
}

func TestPathResolver_WorkspaceDir(t *testing.T) {
	p := NewPathResolverWithPaths("/home/user", "/project/root", "")
	got := p.WorkspaceDir()
	assert.Equal(t, "/project/root/.campaign", got)
}

func TestPathResolver_WorkspaceTemplatesDir(t *testing.T) {
	p := NewPathResolverWithPaths("/home/user", "/project/root", "")
	got := p.WorkspaceTemplatesDir()
	assert.Equal(t, "/project/root/.campaign/templates", got)
}

func TestPathResolver_WorkspaceRegistryFile(t *testing.T) {
	p := NewPathResolverWithPaths("/home/user", "/project/root", "")
	got := p.WorkspaceRegistryFile()
	assert.Equal(t, "/project/root/.campaign/templates/registry.yaml", got)
}

func TestPathResolver_ResolveTemplatePath(t *testing.T) {
	// Create temp directories for testing
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	workDir := filepath.Join(tempDir, "project")

	// Create global templates dir with a template
	globalTemplatesDir := filepath.Join(homeDir, ".config", "guild", "templates")
	globalTemplate := filepath.Join(globalTemplatesDir, "global-only")
	require.NoError(t, os.MkdirAll(globalTemplate, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(globalTemplate, "scaffold.yaml"), []byte("name: global-only"), 0644))

	// Create workspace templates dir with a template
	workspaceTemplatesDir := filepath.Join(workDir, ".campaign", "templates")
	workspaceTemplate := filepath.Join(workspaceTemplatesDir, "workspace-only")
	require.NoError(t, os.MkdirAll(workspaceTemplate, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(workspaceTemplate, "scaffold.yaml"), []byte("name: workspace-only"), 0644))

	// Create a template that exists in both (workspace should win)
	sharedGlobal := filepath.Join(globalTemplatesDir, "shared")
	require.NoError(t, os.MkdirAll(sharedGlobal, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedGlobal, "scaffold.yaml"), []byte("name: shared"), 0644))

	sharedWorkspace := filepath.Join(workspaceTemplatesDir, "shared")
	require.NoError(t, os.MkdirAll(sharedWorkspace, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sharedWorkspace, "scaffold.yaml"), []byte("name: shared"), 0644))

	p := NewPathResolverWithPaths(homeDir, workDir, "")
	ctx := context.Background()

	tests := []struct {
		name       string
		template   string
		wantPath   string
		wantSource string
		wantErr    bool
	}{
		{
			name:       "finds global-only template",
			template:   "global-only",
			wantPath:   globalTemplate,
			wantSource: "global",
			wantErr:    false,
		},
		{
			name:       "finds workspace-only template",
			template:   "workspace-only",
			wantPath:   workspaceTemplate,
			wantSource: "workspace",
			wantErr:    false,
		},
		{
			name:       "workspace takes precedence for shared template",
			template:   "shared",
			wantPath:   sharedWorkspace,
			wantSource: "workspace",
			wantErr:    false,
		},
		{
			name:     "returns error for non-existent template",
			template: "does-not-exist",
			wantErr:  true,
		},
		{
			name:     "returns error for empty name",
			template: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, source, err := p.ResolveTemplatePath(ctx, tt.template)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantSource, source)
		})
	}
}

func TestPathResolver_ResolveTemplatePath_ContextCancelled(t *testing.T) {
	p := NewPathResolverWithPaths("/home", "/work", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := p.ResolveTemplatePath(ctx, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestPathResolver_EnsureGlobalDirs(t *testing.T) {
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")

	p := NewPathResolverWithPaths(homeDir, "/work", "")
	ctx := context.Background()

	err := p.EnsureGlobalDirs(ctx)
	require.NoError(t, err)

	// Verify directories were created
	assert.DirExists(t, p.GlobalConfigDir())
	assert.DirExists(t, p.GlobalTemplatesDir())
}

func TestPathResolver_EnsureGlobalDirs_ContextCancelled(t *testing.T) {
	p := NewPathResolverWithPaths("/home", "/work", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := p.EnsureGlobalDirs(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestPathResolver_ListTemplates(t *testing.T) {
	// Create temp directories for testing
	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	workDir := filepath.Join(tempDir, "project")

	// Create global templates
	globalTemplatesDir := filepath.Join(homeDir, ".config", "guild", "templates")
	for _, name := range []string{"template-a", "template-b"} {
		dir := filepath.Join(globalTemplatesDir, name)
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "scaffold.yaml"), []byte("name: "+name), 0644))
	}

	// Create workspace template (including one that overrides global)
	workspaceTemplatesDir := filepath.Join(workDir, ".campaign", "templates")
	for _, name := range []string{"template-b", "template-c"} {
		dir := filepath.Join(workspaceTemplatesDir, name)
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "scaffold.yaml"), []byte("name: "+name), 0644))
	}

	p := NewPathResolverWithPaths(homeDir, workDir, "")
	ctx := context.Background()

	templates, err := p.ListTemplates(ctx)
	require.NoError(t, err)

	// Should have 3 templates: template-a (global), template-b (workspace), template-c (workspace)
	assert.Len(t, templates, 3)

	// Create map for easier assertions
	byName := make(map[string]TemplateInfo)
	for _, t := range templates {
		byName[t.Name] = t
	}

	assert.Equal(t, "global", byName["template-a"].Source)
	assert.Equal(t, "workspace", byName["template-b"].Source) // workspace takes precedence
	assert.Equal(t, "workspace", byName["template-c"].Source)
}

func TestPathResolver_ListTemplates_ContextCancelled(t *testing.T) {
	p := NewPathResolverWithPaths("/home", "/work", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.ListTemplates(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestNewPathResolver(t *testing.T) {
	p, err := NewPathResolver()
	require.NoError(t, err)
	assert.NotNil(t, p)

	// Should use real system paths
	home, _ := os.UserHomeDir()
	assert.Contains(t, p.GlobalConfigDir(), home)
}

func TestConvenienceFunctions(t *testing.T) {
	// These use system defaults, so just verify they don't error
	t.Run("GlobalConfigDir", func(t *testing.T) {
		path, err := GlobalConfigDir()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("GlobalTemplatesDir", func(t *testing.T) {
		path, err := GlobalTemplatesDir()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})

	t.Run("WorkspaceTemplatesDir", func(t *testing.T) {
		path, err := WorkspaceTemplatesDir()
		require.NoError(t, err)
		assert.NotEmpty(t, path)
	})
}
