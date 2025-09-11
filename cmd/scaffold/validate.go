// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-scaffold/pkg/scaffold/cli"
)

var validateCmd = &cobra.Command{
	Use:   "validate [scaffold-file]",
	Short: "Validate a scaffold configuration file",
	Long: `Validate a scaffold configuration file for syntax and semantic correctness.

This command checks scaffold.yaml files for proper structure, required fields,
template existence, and other validation rules.

Examples:
  scaffold validate                        # Validate ./scaffold.yaml
  scaffold validate path/to/scaffold.yaml  # Validate specific file
  scaffold validate --template campaign    # Validate built-in template`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

// ValidateFlags represents command-line flags for the validate command
type ValidateFlags struct {
	Template string
	Verbose  bool
	Format   string
}

var validateFlags ValidateFlags

func init() {
	validateCmd.Flags().StringVarP(&validateFlags.Template, "template", "t", "",
		"Validate a built-in template instead of a file")
	validateCmd.Flags().BoolVarP(&validateFlags.Verbose, "verbose", "v", false,
		"Show detailed validation information")
	validateCmd.Flags().StringVar(&validateFlags.Format, "format", "text",
		"Output format: text, json, yaml")
}

// runValidate executes the validate command
func runValidate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	
	var scaffoldPath string
	if len(args) > 0 {
		scaffoldPath = args[0]
	} else {
		scaffoldPath = "scaffold.yaml" // Default
	}
	
	options := &cli.ValidateOptions{
		ScaffoldPath: scaffoldPath,
		Template:     validateFlags.Template,
		Verbose:      validateFlags.Verbose,
		Format:       validateFlags.Format,
	}
	
	return cli.ValidateScaffold(ctx, options)
}