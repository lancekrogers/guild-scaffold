// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"github.com/spf13/cobra"

	"github.com/lancekrogers/guild-scaffold/pkg/scaffold/cli"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync templates from GitHub",
	Long: `Sync scaffold templates from GitHub to your local configuration directory.

Templates are downloaded to ~/.config/guild/templates/ and tracked in a registry
file for version management. Workspace-level templates in .campaign/templates/
take precedence over global templates.

Examples:
  scaffold sync                     # Sync all templates from default repo
  scaffold sync --template NAME     # Sync a specific template
  scaffold sync --list              # Show available templates from remote
  scaffold sync --check             # Check for available updates`,
	RunE: runSync,
}

// SyncFlags represents command-line flags for the sync command
type SyncFlags struct {
	Template string
	List     bool
	Check    bool
	Force    bool
	Repo     string
	Branch   string
}

var syncFlags SyncFlags

func init() {
	syncCmd.Flags().StringVarP(&syncFlags.Template, "template", "t", "",
		"Sync a specific template by name")
	syncCmd.Flags().BoolVarP(&syncFlags.List, "list", "l", false,
		"List available templates from remote repository")
	syncCmd.Flags().BoolVarP(&syncFlags.Check, "check", "c", false,
		"Check for available updates without syncing")
	syncCmd.Flags().BoolVarP(&syncFlags.Force, "force", "f", false,
		"Force sync even if local version matches remote")
	syncCmd.Flags().StringVar(&syncFlags.Repo, "repo", "",
		"Override sync repository (format: owner/repo)")
	syncCmd.Flags().StringVar(&syncFlags.Branch, "branch", "",
		"Override sync branch or tag")
}

// runSync executes the sync command
func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	options := &cli.SyncOptions{
		Template: syncFlags.Template,
		List:     syncFlags.List,
		Check:    syncFlags.Check,
		Force:    syncFlags.Force,
		Repo:     syncFlags.Repo,
		Branch:   syncFlags.Branch,
	}

	return cli.ExecuteSync(ctx, options)
}
