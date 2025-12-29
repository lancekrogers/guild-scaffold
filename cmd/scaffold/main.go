// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Guild Scaffold - Template-driven project initialization",
	Long: `Guild Scaffold creates structured projects using YAML-driven templates.

This is the standalone scaffold tool that can be used independently or
integrated with the main guild CLI for enhanced project initialization.

Examples:
  scaffold init my-project                    # Use default template
  scaffold init my-project --template campaign  # Use specific template
  scaffold init my-project --dry-run           # Preview without creating files
  scaffold list-templates                     # Show available templates`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Register commands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(syncCmd)
}

func main() {
	ctx := context.Background()
	
	// Set up error handling
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}