// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package main

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-scaffold/pkg/scaffold/cli"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new project with template-driven scaffolding",
	Long: `Initialize a new Guild project using YAML-driven templates.

This command creates a new project structure based on configurable templates,
providing a flexible and maintainable approach to project initialization.

Examples:
  scaffold init my-project                    # Use default template
  scaffold init my-project --template campaign  # Use specific template
  scaffold init my-project --dry-run           # Preview without creating files
  scaffold init --list-templates              # Show available templates`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

// InitFlags represents command-line flags for the init command
type InitFlags struct {
	Template      string
	DryRun        bool
	ListTemplates bool
	Force         bool
	OutputDir     string
	ConfigFile    string
	Variables     []string
	Provider      string
	Model         string
	Interactive   bool
	Verbose       bool
}

var initFlags InitFlags

func init() {
	// Add flags to the command
	initCmd.Flags().StringVarP(&initFlags.Template, "template", "t", "",
		"Template to use for initialization (default: auto-detect)")
	initCmd.Flags().BoolVar(&initFlags.DryRun, "dry-run", false,
		"Show what would be created without actually creating files")
	initCmd.Flags().BoolVar(&initFlags.ListTemplates, "list-templates", false,
		"List available templates and exit")
	initCmd.Flags().BoolVar(&initFlags.Force, "force", false,
		"Overwrite existing files (dangerous - use with caution)")
	initCmd.Flags().StringVarP(&initFlags.OutputDir, "output", "o", ".",
		"Output directory for the project")
	initCmd.Flags().StringVar(&initFlags.ConfigFile, "config", "",
		"Custom scaffold configuration file")
	initCmd.Flags().StringSliceVarP(&initFlags.Variables, "var", "v", nil,
		"Set template variables (format: key=value)")
	initCmd.Flags().StringVar(&initFlags.Provider, "provider", "",
		"Default LLM provider for agents")
	initCmd.Flags().StringVar(&initFlags.Model, "model", "",
		"Default model for agents")
	initCmd.Flags().BoolVarP(&initFlags.Interactive, "interactive", "i", false,
		"Interactive mode with prompts for configuration")
	initCmd.Flags().BoolVar(&initFlags.Verbose, "verbose", false,
		"Enable verbose output")
}

// runInit executes the scaffold-enabled init command
func runInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Handle list templates request
	if initFlags.ListTemplates {
		return listAvailableTemplates(ctx)
	}

	// Determine project name
	projectName := "guild-project"
	if len(args) > 0 {
		projectName = args[0]
	}

	// Create CLI options from flags
	options, err := createCLIOptions(ctx, projectName)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create CLI options")
	}

	// Execute scaffolding through CLI package
	return cli.ExecuteInit(ctx, options)
}

// createCLIOptions creates CLI options from command flags
func createCLIOptions(ctx context.Context, projectName string) (*cli.InitOptions, error) {
	// Determine output directory
	outputDir, err := filepath.Abs(initFlags.OutputDir)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "invalid output directory").
			WithDetails("path", initFlags.OutputDir)
	}

	// Parse template variables
	variables, err := parseVariables(initFlags.Variables)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInvalidInput, "failed to parse variables")
	}

	// Add project name to variables (only if not already provided via --var)
	if _, ok := variables["project_name"]; !ok {
		variables["project_name"] = projectName
	}
	if _, ok := variables["campaign_name"]; !ok {
		variables["campaign_name"] = projectName
	}

	// Determine template to use
	templateName := initFlags.Template
	if templateName == "" {
		templateName = cli.DetectTemplateFromContext(ctx, outputDir)
	}

	// Create options
	options := &cli.InitOptions{
		ProjectName:     projectName,
		TemplateName:    templateName,
		OutputDirectory: outputDir,
		Variables:       variables,
		DryRun:          initFlags.DryRun,
		Force:           initFlags.Force,
		ConfigFile:      initFlags.ConfigFile,
		Provider:        initFlags.Provider,
		Model:           initFlags.Model,
		Interactive:     initFlags.Interactive,
		Verbose:         initFlags.Verbose,
	}

	return options, nil
}

// parseVariables parses key=value variable assignments
func parseVariables(varStrings []string) (map[string]interface{}, error) {
	variables := make(map[string]interface{})

	for _, varStr := range varStrings {
		parts := strings.SplitN(varStr, "=", 2)
		if len(parts) != 2 {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "invalid variable format", nil).
				WithDetails("variable", varStr).
				WithDetails("expected_format", "key=value")
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, gerror.New(gerror.ErrCodeInvalidInput, "variable key cannot be empty", nil).
				WithDetails("variable", varStr)
		}

		// Attempt to parse as different types
		variables[key] = parseVariableValue(value)
	}

	return variables, nil
}

// parseVariableValue attempts to parse a string value as appropriate type
func parseVariableValue(value string) interface{} {
	// Try boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Try integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Return as string
	return value
}

// listAvailableTemplates shows all available templates
func listAvailableTemplates(ctx context.Context) error {
	return cli.ListTemplates(ctx)
}
