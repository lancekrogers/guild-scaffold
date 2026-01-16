// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lancekrogers/guild-scaffold/pkg/scaffold/config"
	"gopkg.in/yaml.v3"
)

// SyncOptions contains options for the sync command.
type SyncOptions struct {
	// Template is a specific template to sync (empty for all)
	Template string

	// List shows available templates from remote without syncing
	List bool

	// Check checks for updates without syncing
	Check bool

	// Force syncs even if versions match
	Force bool

	// Repo overrides the sync repository
	Repo string

	// Branch overrides the sync branch
	Branch string
}

// ExecuteSync executes the sync command.
func ExecuteSync(ctx context.Context, options *SyncOptions) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled: %w", err)
	}

	// Create path resolver
	paths, err := config.NewPathResolver()
	if err != nil {
		return fmt.Errorf("failed to create path resolver: %w", err)
	}

	// Build sync config
	syncConfig := config.NewSyncConfig()
	if options.Repo != "" {
		syncConfig.SyncRepo = options.Repo
	}
	if options.Branch != "" {
		syncConfig.SyncBranch = options.Branch
	}

	// Handle different modes
	if options.List {
		return executeListRemote(ctx, syncConfig)
	}

	if options.Check {
		return executeCheckUpdates(ctx, paths, syncConfig)
	}

	// Ensure config directories exist
	if err := paths.EnsureGlobalDirs(ctx); err != nil {
		return fmt.Errorf("failed to create config directories: %w", err)
	}

	// Execute sync
	if options.Template != "" {
		return executeSyncTemplate(ctx, paths, syncConfig, options.Template, options.Force)
	}

	return executeSyncAll(ctx, paths, syncConfig, options.Force)
}

// executeListRemote lists available templates from the remote repository.
func executeListRemote(ctx context.Context, cfg *config.SyncConfig) error {
	fmt.Println("Fetching available templates from", cfg.EffectiveSyncRepo())
	fmt.Println()

	templates, err := fetchRemoteTemplates(ctx, cfg)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		fmt.Println("No templates found in remote repository")
		return nil
	}

	fmt.Printf("Available templates (%d):\n", len(templates))
	fmt.Println()
	for _, t := range templates {
		fmt.Printf("  %s\n", t.Name)
		if t.Description != "" {
			fmt.Printf("    %s\n", t.Description)
		}
		if t.Version != "" {
			fmt.Printf("    Version: %s\n", t.Version)
		}
		fmt.Println()
	}

	return nil
}

// executeCheckUpdates checks for available updates without syncing.
func executeCheckUpdates(ctx context.Context, paths *config.PathResolver, cfg *config.SyncConfig) error {
	fmt.Println("Checking for updates...")
	fmt.Println()

	// Load local registry
	localRegistry, err := loadLocalRegistry(paths)
	if err != nil {
		// No local registry is fine - just means nothing is synced
		localRegistry = config.NewRegistryFile()
	}

	// Fetch remote templates
	remoteTemplates, err := fetchRemoteTemplates(ctx, cfg)
	if err != nil {
		return err
	}

	if len(remoteTemplates) == 0 {
		fmt.Println("No templates found in remote repository")
		return nil
	}

	// Compare versions
	updatesAvailable := false
	for _, remote := range remoteTemplates {
		local := localRegistry.GetTemplate(remote.Name)
		if local == nil {
			fmt.Printf("  [NEW] %s (%s)\n", remote.Name, remote.Version)
			updatesAvailable = true
		} else if local.Version != remote.Version {
			fmt.Printf("  [UPDATE] %s: %s -> %s\n", remote.Name, local.Version, remote.Version)
			updatesAvailable = true
		}
	}

	if !updatesAvailable {
		fmt.Println("All templates are up to date")
	} else {
		fmt.Println()
		fmt.Println("Run 'scaffold sync' to download updates")
	}

	return nil
}

// executeSyncTemplate syncs a specific template.
func executeSyncTemplate(ctx context.Context, paths *config.PathResolver, cfg *config.SyncConfig, name string, force bool) error {
	fmt.Printf("Syncing template: %s\n", name)

	// Fetch remote templates
	remoteTemplates, err := fetchRemoteTemplates(ctx, cfg)
	if err != nil {
		return err
	}

	// Find the requested template
	var target *config.RemoteTemplate
	for _, t := range remoteTemplates {
		if t.Name == name {
			target = &t
			break
		}
	}

	if target == nil {
		return fmt.Errorf("template not found in remote repository: name=%s, repo=%s", name, cfg.EffectiveSyncRepo())
	}

	// Load local registry
	localRegistry, err := loadLocalRegistry(paths)
	if err != nil {
		localRegistry = config.NewRegistryFile()
	}

	// Check if update needed
	local := localRegistry.GetTemplate(name)
	if !force && local != nil && local.Version == target.Version {
		fmt.Printf("  Template %s is already up to date (%s)\n", name, local.Version)
		return nil
	}

	// Download and extract template
	if err := downloadTemplate(ctx, cfg, target, paths.GlobalTemplatesDir()); err != nil {
		return err
	}

	// Update registry
	localRegistry.SetTemplate(config.TemplateVersion{
		Name:     target.Name,
		Version:  target.Version,
		SyncedAt: time.Now(),
		Source:   cfg.EffectiveSyncRepo(),
	})

	if err := saveLocalRegistry(paths, localRegistry); err != nil {
		return err
	}

	fmt.Printf("  Successfully synced %s (%s)\n", name, target.Version)
	return nil
}

// executeSyncAll syncs all templates from the remote repository.
func executeSyncAll(ctx context.Context, paths *config.PathResolver, cfg *config.SyncConfig, force bool) error {
	fmt.Println("Syncing templates from", cfg.EffectiveSyncRepo())
	fmt.Println()

	// Fetch remote templates
	remoteTemplates, err := fetchRemoteTemplates(ctx, cfg)
	if err != nil {
		return err
	}

	if len(remoteTemplates) == 0 {
		fmt.Println("No templates found in remote repository")
		return nil
	}

	// Load local registry
	localRegistry, err := loadLocalRegistry(paths)
	if err != nil {
		localRegistry = config.NewRegistryFile()
	}

	synced := 0
	skipped := 0
	for _, remote := range remoteTemplates {
		local := localRegistry.GetTemplate(remote.Name)

		// Skip if up to date and not forced
		if !force && local != nil && local.Version == remote.Version {
			fmt.Printf("  [skip] %s (%s) - up to date\n", remote.Name, local.Version)
			skipped++
			continue
		}

		action := "added"
		if local != nil {
			action = "updated"
		}

		// Download and extract template
		if err := downloadTemplate(ctx, cfg, &remote, paths.GlobalTemplatesDir()); err != nil {
			fmt.Printf("  [error] %s: %v\n", remote.Name, err)
			continue
		}

		// Update registry
		localRegistry.SetTemplate(config.TemplateVersion{
			Name:     remote.Name,
			Version:  remote.Version,
			SyncedAt: time.Now(),
			Source:   cfg.EffectiveSyncRepo(),
		})

		fmt.Printf("  [%s] %s (%s)\n", action, remote.Name, remote.Version)
		synced++
	}

	if err := saveLocalRegistry(paths, localRegistry); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("Sync complete: %d synced, %d skipped\n", synced, skipped)
	fmt.Printf("Templates stored in: %s\n", paths.GlobalTemplatesDir())

	return nil
}

// fetchRemoteTemplates fetches the list of available templates from GitHub.
func fetchRemoteTemplates(ctx context.Context, cfg *config.SyncConfig) ([]config.RemoteTemplate, error) {
	// Parse repo from config
	repo := cfg.EffectiveSyncRepo()
	branch := cfg.EffectiveSyncBranch()
	libraryPath := cfg.EffectiveLibraryPath()

	// Strip github.com prefix if present
	repo = strings.TrimPrefix(repo, "github.com/")

	// Construct API URL for contents
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", repo, libraryPath, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "guild-scaffold")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote templates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("library path not found: repo=%s, path=%s, branch=%s", repo, libraryPath, branch)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected response from GitHub: status=%d", resp.StatusCode)
	}

	// Parse response
	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	// Filter to directories only (templates are directories)
	var templates []config.RemoteTemplate
	for _, item := range contents {
		if item.Type == "dir" {
			// Try to fetch scaffold.yaml for metadata
			t := config.RemoteTemplate{
				Name: item.Name,
				Path: item.Path,
			}

			// Attempt to fetch scaffold.yaml for version/description
			scaffoldURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/scaffold.yaml",
				repo, branch, item.Path)
			if meta, err := fetchScaffoldMeta(ctx, scaffoldURL); err == nil {
				t.Version = meta.Version
				t.Description = meta.Description
			}

			templates = append(templates, t)
		}
	}

	return templates, nil
}

// scaffoldMeta represents metadata from scaffold.yaml
type scaffoldMeta struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// fetchScaffoldMeta fetches metadata from a scaffold.yaml file.
func fetchScaffoldMeta(ctx context.Context, url string) (*scaffoldMeta, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "guild-scaffold")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var meta scaffoldMeta
	if err := yaml.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// downloadTemplate downloads and extracts a template to the target directory.
func downloadTemplate(ctx context.Context, cfg *config.SyncConfig, template *config.RemoteTemplate, targetDir string) error {
	repo := strings.TrimPrefix(cfg.EffectiveSyncRepo(), "github.com/")
	branch := cfg.EffectiveSyncBranch()

	// Download the tarball for the repo
	tarURL := fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", repo, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", tarURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "guild-scaffold")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download tarball: status=%d", resp.StatusCode)
	}

	// Extract the specific template directory
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decompress tarball: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Template destination
	templateDest := filepath.Join(targetDir, template.Name)

	// Create template directory
	if err := os.MkdirAll(templateDest, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	// The tarball has a root directory like "owner-repo-hash/"
	// We need to extract files from "root/library/template-name/" to our destination
	templatePrefix := "/" + template.Path + "/"

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tarball: %w", err)
		}

		// Skip the root directory prefix
		name := header.Name
		slashIdx := strings.Index(name, "/")
		if slashIdx >= 0 {
			name = name[slashIdx:]
		}

		// Check if this file is in our template directory
		if !strings.HasPrefix(name, templatePrefix) {
			continue
		}

		// Get relative path within template
		relPath := strings.TrimPrefix(name, templatePrefix)
		if relPath == "" {
			continue
		}

		destPath := filepath.Join(templateDest, relPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			f, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			f.Close()

			// Set file permissions
			if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
				// Non-fatal, continue
			}
		}
	}

	return nil
}

// loadLocalRegistry loads the local template registry.
func loadLocalRegistry(paths *config.PathResolver) (*config.RegistryFile, error) {
	registryPath := paths.GlobalRegistryFile()

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, err
	}

	var registry config.RegistryFile
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, err
	}

	return &registry, nil
}

// saveLocalRegistry saves the local template registry.
func saveLocalRegistry(paths *config.PathResolver, registry *config.RegistryFile) error {
	registryPath := paths.GlobalRegistryFile()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(registryPath), 0755); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	data, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}
