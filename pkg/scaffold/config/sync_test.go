// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyncConfig(t *testing.T) {
	cfg := NewSyncConfig()

	assert.Equal(t, DefaultSyncRepo, cfg.SyncRepo)
	assert.Equal(t, DefaultSyncBranch, cfg.SyncBranch)
	assert.Equal(t, DefaultLibraryPath, cfg.LibraryPath)
	assert.Nil(t, cfg.LastSync)
}

func TestSyncConfig_EffectiveSyncRepo(t *testing.T) {
	tests := []struct {
		name     string
		syncRepo string
		want     string
	}{
		{
			name:     "uses default when empty",
			syncRepo: "",
			want:     DefaultSyncRepo,
		},
		{
			name:     "uses custom when set",
			syncRepo: "github.com/custom/repo",
			want:     "github.com/custom/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SyncConfig{SyncRepo: tt.syncRepo}
			assert.Equal(t, tt.want, cfg.EffectiveSyncRepo())
		})
	}
}

func TestSyncConfig_EffectiveSyncBranch(t *testing.T) {
	tests := []struct {
		name       string
		syncBranch string
		want       string
	}{
		{
			name:       "uses default when empty",
			syncBranch: "",
			want:       DefaultSyncBranch,
		},
		{
			name:       "uses custom when set",
			syncBranch: "develop",
			want:       "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SyncConfig{SyncBranch: tt.syncBranch}
			assert.Equal(t, tt.want, cfg.EffectiveSyncBranch())
		})
	}
}

func TestSyncConfig_EffectiveLibraryPath(t *testing.T) {
	tests := []struct {
		name        string
		libraryPath string
		want        string
	}{
		{
			name:        "uses default when empty",
			libraryPath: "",
			want:        DefaultLibraryPath,
		},
		{
			name:        "uses custom when set",
			libraryPath: "templates",
			want:        "templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SyncConfig{LibraryPath: tt.libraryPath}
			assert.Equal(t, tt.want, cfg.EffectiveLibraryPath())
		})
	}
}

func TestNewRegistryFile(t *testing.T) {
	rf := NewRegistryFile()

	assert.NotNil(t, rf.Templates)
	assert.Empty(t, rf.Templates)
	assert.Nil(t, rf.SyncConfig)
}

func TestRegistryFile_GetTemplate(t *testing.T) {
	rf := &RegistryFile{
		Templates: []TemplateVersion{
			{Name: "template-a", Version: "1.0.0"},
			{Name: "template-b", Version: "2.0.0"},
		},
	}

	t.Run("finds existing template", func(t *testing.T) {
		tv := rf.GetTemplate("template-a")
		require.NotNil(t, tv)
		assert.Equal(t, "1.0.0", tv.Version)
	})

	t.Run("returns nil for non-existent template", func(t *testing.T) {
		tv := rf.GetTemplate("does-not-exist")
		assert.Nil(t, tv)
	})
}

func TestRegistryFile_SetTemplate(t *testing.T) {
	t.Run("adds new template", func(t *testing.T) {
		rf := NewRegistryFile()
		rf.SetTemplate(TemplateVersion{Name: "new-template", Version: "1.0.0"})

		assert.Len(t, rf.Templates, 1)
		assert.Equal(t, "new-template", rf.Templates[0].Name)
	})

	t.Run("updates existing template", func(t *testing.T) {
		rf := &RegistryFile{
			Templates: []TemplateVersion{
				{Name: "template-a", Version: "1.0.0"},
			},
		}

		rf.SetTemplate(TemplateVersion{Name: "template-a", Version: "2.0.0"})

		assert.Len(t, rf.Templates, 1)
		assert.Equal(t, "2.0.0", rf.Templates[0].Version)
	})
}

func TestRegistryFile_RemoveTemplate(t *testing.T) {
	t.Run("removes existing template", func(t *testing.T) {
		rf := &RegistryFile{
			Templates: []TemplateVersion{
				{Name: "template-a", Version: "1.0.0"},
				{Name: "template-b", Version: "2.0.0"},
			},
		}

		rf.RemoveTemplate("template-a")

		assert.Len(t, rf.Templates, 1)
		assert.Equal(t, "template-b", rf.Templates[0].Name)
	})

	t.Run("no-op for non-existent template", func(t *testing.T) {
		rf := &RegistryFile{
			Templates: []TemplateVersion{
				{Name: "template-a", Version: "1.0.0"},
			},
		}

		rf.RemoveTemplate("does-not-exist")

		assert.Len(t, rf.Templates, 1)
	})
}

func TestTemplateVersion(t *testing.T) {
	now := time.Now()
	tv := TemplateVersion{
		Name:     "test-template",
		Version:  "1.0.0",
		Commit:   "abc123",
		SyncedAt: now,
		Source:   "github.com/test/repo",
	}

	assert.Equal(t, "test-template", tv.Name)
	assert.Equal(t, "1.0.0", tv.Version)
	assert.Equal(t, "abc123", tv.Commit)
	assert.Equal(t, now, tv.SyncedAt)
	assert.Equal(t, "github.com/test/repo", tv.Source)
}

func TestNewGitHubSyncer(t *testing.T) {
	t.Run("with nil config uses defaults", func(t *testing.T) {
		syncer, err := NewGitHubSyncer(nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, syncer)
		assert.Equal(t, DefaultSyncRepo, syncer.config.SyncRepo)
	})

	t.Run("with custom config", func(t *testing.T) {
		cfg := &SyncConfig{SyncRepo: "custom/repo"}
		syncer, err := NewGitHubSyncer(cfg, nil)
		require.NoError(t, err)
		assert.Equal(t, "custom/repo", syncer.config.SyncRepo)
	})
}

func TestGitHubSyncer_NotImplemented(t *testing.T) {
	syncer, err := NewGitHubSyncer(nil, nil)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Sync returns not implemented", func(t *testing.T) {
		_, err := syncer.Sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("SyncTemplate returns not implemented", func(t *testing.T) {
		_, err := syncer.SyncTemplate(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("ListRemote returns not implemented", func(t *testing.T) {
		_, err := syncer.ListRemote(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})

	t.Run("CheckUpdates returns not implemented", func(t *testing.T) {
		_, err := syncer.CheckUpdates(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})
}

func TestGitHubSyncer_ContextCancellation(t *testing.T) {
	syncer, err := NewGitHubSyncer(nil, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("Sync respects context cancellation", func(t *testing.T) {
		_, err := syncer.Sync(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("SyncTemplate respects context cancellation", func(t *testing.T) {
		_, err := syncer.SyncTemplate(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("ListRemote respects context cancellation", func(t *testing.T) {
		_, err := syncer.ListRemote(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})

	t.Run("CheckUpdates respects context cancellation", func(t *testing.T) {
		_, err := syncer.CheckUpdates(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cancelled")
	})
}

func TestSyncResult(t *testing.T) {
	result := SyncResult{
		Template:   "test-template",
		Action:     "updated",
		OldVersion: "1.0.0",
		NewVersion: "2.0.0",
		Error:      nil,
	}

	assert.Equal(t, "test-template", result.Template)
	assert.Equal(t, "updated", result.Action)
	assert.Equal(t, "1.0.0", result.OldVersion)
	assert.Equal(t, "2.0.0", result.NewVersion)
	assert.Nil(t, result.Error)
}

func TestRemoteTemplate(t *testing.T) {
	rt := RemoteTemplate{
		Name:        "test-template",
		Version:     "1.0.0",
		Description: "A test template",
		Path:        "library/test-template",
	}

	assert.Equal(t, "test-template", rt.Name)
	assert.Equal(t, "1.0.0", rt.Version)
	assert.Equal(t, "A test template", rt.Description)
	assert.Equal(t, "library/test-template", rt.Path)
}

func TestUpdateInfo(t *testing.T) {
	ui := UpdateInfo{
		Name:             "test-template",
		CurrentVersion:   "1.0.0",
		AvailableVersion: "2.0.0",
		UpdateAvailable:  true,
	}

	assert.Equal(t, "test-template", ui.Name)
	assert.Equal(t, "1.0.0", ui.CurrentVersion)
	assert.Equal(t, "2.0.0", ui.AvailableVersion)
	assert.True(t, ui.UpdateAvailable)
}
