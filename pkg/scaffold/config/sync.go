// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package config

import (
	"context"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"gopkg.in/yaml.v3"
)

// Default sync configuration
const (
	// DefaultSyncRepo is the default GitHub repository to sync from
	DefaultSyncRepo = "github.com/guild-framework/guild-scaffold"

	// DefaultSyncBranch is the default branch to sync from
	DefaultSyncBranch = "main"

	// DefaultLibraryPath is the path within the repo containing templates
	DefaultLibraryPath = "library"
)

// SyncConfig holds configuration for template synchronization.
// Stored in ~/.config/guild/config.yaml
type SyncConfig struct {
	// SyncRepo is the GitHub repository to sync templates from
	// Format: "github.com/owner/repo" or "owner/repo"
	SyncRepo string `yaml:"sync_repo,omitempty" json:"sync_repo,omitempty"`

	// SyncBranch is the branch or tag to sync from
	SyncBranch string `yaml:"sync_branch,omitempty" json:"sync_branch,omitempty"`

	// LibraryPath is the path within the repo containing templates
	LibraryPath string `yaml:"library_path,omitempty" json:"library_path,omitempty"`

	// LastSync is the timestamp of the last successful sync
	LastSync *time.Time `yaml:"last_sync,omitempty" json:"last_sync,omitempty"`
}

// NewSyncConfig returns a SyncConfig with default values.
func NewSyncConfig() *SyncConfig {
	return &SyncConfig{
		SyncRepo:    DefaultSyncRepo,
		SyncBranch:  DefaultSyncBranch,
		LibraryPath: DefaultLibraryPath,
	}
}

// EffectiveSyncRepo returns the sync repo, using default if not set.
func (c *SyncConfig) EffectiveSyncRepo() string {
	if c.SyncRepo != "" {
		return c.SyncRepo
	}
	return DefaultSyncRepo
}

// EffectiveSyncBranch returns the sync branch, using default if not set.
func (c *SyncConfig) EffectiveSyncBranch() string {
	if c.SyncBranch != "" {
		return c.SyncBranch
	}
	return DefaultSyncBranch
}

// EffectiveLibraryPath returns the library path, using default if not set.
func (c *SyncConfig) EffectiveLibraryPath() string {
	if c.LibraryPath != "" {
		return c.LibraryPath
	}
	return DefaultLibraryPath
}

// TemplateVersion tracks version information for a synced template.
type TemplateVersion struct {
	// Name is the template name
	Name string `yaml:"name" json:"name"`

	// Version is the template version (from scaffold.yaml)
	Version string `yaml:"version" json:"version"`

	// Commit is the git commit hash when synced
	Commit string `yaml:"commit,omitempty" json:"commit,omitempty"`

	// SyncedAt is when the template was synced
	SyncedAt time.Time `yaml:"synced_at" json:"synced_at"`

	// Source is the repository the template was synced from
	Source string `yaml:"source,omitempty" json:"source,omitempty"`
}

// RegistryFile represents the registry.yaml file in the templates directory.
// Tracks synced templates and their versions.
type RegistryFile struct {
	// Templates lists all synced templates with version info
	Templates []TemplateVersion `yaml:"templates" json:"templates"`

	// SyncConfig contains sync configuration (optional override per directory)
	SyncConfig *SyncConfig `yaml:"sync_config,omitempty" json:"sync_config,omitempty"`
}

// NewRegistryFile creates an empty registry file.
func NewRegistryFile() *RegistryFile {
	return &RegistryFile{
		Templates: make([]TemplateVersion, 0),
	}
}

// GetTemplate returns version info for a template by name.
func (r *RegistryFile) GetTemplate(name string) *TemplateVersion {
	for i := range r.Templates {
		if r.Templates[i].Name == name {
			return &r.Templates[i]
		}
	}
	return nil
}

// SetTemplate updates or adds a template version.
func (r *RegistryFile) SetTemplate(tv TemplateVersion) {
	for i := range r.Templates {
		if r.Templates[i].Name == tv.Name {
			r.Templates[i] = tv
			return
		}
	}
	r.Templates = append(r.Templates, tv)
}

// RemoveTemplate removes a template from the registry.
func (r *RegistryFile) RemoveTemplate(name string) {
	for i := range r.Templates {
		if r.Templates[i].Name == name {
			r.Templates = append(r.Templates[:i], r.Templates[i+1:]...)
			return
		}
	}
}

// SyncResult represents the result of a sync operation.
type SyncResult struct {
	// Template is the template name
	Template string

	// Action is what was done: "added", "updated", "unchanged", "removed"
	Action string

	// OldVersion is the previous version (for updates)
	OldVersion string

	// NewVersion is the new version
	NewVersion string

	// Error is any error that occurred
	Error error
}

// Syncer handles template synchronization from GitHub.
type Syncer interface {
	// Sync synchronizes all templates from the configured source.
	Sync(ctx context.Context) ([]SyncResult, error)

	// SyncTemplate synchronizes a specific template.
	SyncTemplate(ctx context.Context, name string) (*SyncResult, error)

	// ListRemote lists available templates from the remote source.
	ListRemote(ctx context.Context) ([]RemoteTemplate, error)

	// CheckUpdates checks for available updates without syncing.
	CheckUpdates(ctx context.Context) ([]UpdateInfo, error)
}

// RemoteTemplate represents a template available in the remote repository.
type RemoteTemplate struct {
	// Name is the template name
	Name string

	// Version is the template version
	Version string

	// Description is the template description
	Description string

	// Path is the path within the repository
	Path string
}

// UpdateInfo represents information about an available update.
type UpdateInfo struct {
	// Name is the template name
	Name string

	// CurrentVersion is the locally installed version
	CurrentVersion string

	// AvailableVersion is the version available remotely
	AvailableVersion string

	// UpdateAvailable is true if the remote version is newer
	UpdateAvailable bool
}

// GitHubSyncer implements Syncer using GitHub as the source.
// This is a placeholder - full implementation in Phase 002.
type GitHubSyncer struct {
	// config is the sync configuration
	config *SyncConfig

	// paths provides path resolution
	paths *PathResolver

	// registry is the local template registry
	registry *RegistryFile
}

// NewGitHubSyncer creates a new GitHub-based syncer.
func NewGitHubSyncer(config *SyncConfig, paths *PathResolver) (*GitHubSyncer, error) {
	if config == nil {
		config = NewSyncConfig()
	}
	if paths == nil {
		var err error
		paths, err = NewPathResolver()
		if err != nil {
			return nil, err
		}
	}

	return &GitHubSyncer{
		config:   config,
		paths:    paths,
		registry: NewRegistryFile(),
	}, nil
}

// Sync synchronizes all templates from GitHub.
// Placeholder - full implementation in Phase 002.
func (s *GitHubSyncer) Sync(ctx context.Context) ([]SyncResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// TODO: Implement in Phase 002
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "sync not yet implemented", nil).
		WithDetails("hint", "run 'scaffold sync' after Phase 002 is complete")
}

// SyncTemplate synchronizes a specific template from GitHub.
// Placeholder - full implementation in Phase 002.
func (s *GitHubSyncer) SyncTemplate(ctx context.Context, name string) (*SyncResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// TODO: Implement in Phase 002
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "sync not yet implemented", nil).
		WithDetails("template", name)
}

// ListRemote lists available templates from the remote repository.
// Placeholder - full implementation in Phase 002.
func (s *GitHubSyncer) ListRemote(ctx context.Context) ([]RemoteTemplate, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// TODO: Implement in Phase 002
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "list remote not yet implemented", nil)
}

// CheckUpdates checks for available updates without syncing.
// Placeholder - full implementation in Phase 002.
func (s *GitHubSyncer) CheckUpdates(ctx context.Context) ([]UpdateInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeCancelled, "context cancelled")
	}

	// TODO: Implement in Phase 002
	return nil, gerror.New(gerror.ErrCodeNotImplemented, "check updates not yet implemented", nil)
}

// MarshalYAML implements yaml.Marshaler for RegistryFile.
func (r *RegistryFile) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(r)
}

// UnmarshalYAML implements yaml.Unmarshaler for RegistryFile.
func (r *RegistryFile) UnmarshalYAML(data []byte) error {
	type alias RegistryFile
	return yaml.Unmarshal(data, (*alias)(r))
}
