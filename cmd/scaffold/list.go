// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-scaffold/pkg/scaffold/cli"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "list-templates"},
	Short:   "List available templates",
	Long: `List all available scaffold templates with descriptions and use cases.

This command shows all built-in templates that can be used with the init command,
along with their descriptions and recommended use cases.

Examples:
  scaffold list                    # List all templates
  scaffold list-templates          # Same as above
  scaffold ls                      # Short alias`,
	RunE: runList,
}

// ListFlags represents command-line flags for the list command
type ListFlags struct {
	Verbose bool
	Format  string
}

var listFlags ListFlags

func init() {
	listCmd.Flags().BoolVarP(&listFlags.Verbose, "verbose", "v", false,
		"Show detailed template information")
	listCmd.Flags().StringVar(&listFlags.Format, "format", "table",
		"Output format: table, json, yaml")
}

// runList executes the list templates command
func runList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	options := &cli.ListOptions{
		Verbose: listFlags.Verbose,
		Format:  listFlags.Format,
	}

	return cli.ListTemplatesWithOptions(ctx, options)
}
