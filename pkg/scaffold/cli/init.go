// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-scaffold/pkg/scaffold"
	"github.com/guild-framework/guild-scaffold/pkg/scaffold/templates"
)

// ExecuteInit executes the scaffold initialization process
func ExecuteInit(ctx context.Context, options *InitOptions) error {
	if err := options.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid options")
	}
	
	// Interactive mode handling
	if options.Interactive {
		if err := runInteractiveConfiguration(ctx, options); err != nil {
			return gerror.Wrap(err, gerror.ErrCodeInternal, "interactive configuration failed")
		}
	}
	
	// Create scaffold configuration
	scaffoldOpts, err := createScaffoldOptions(ctx, options)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold configuration")
	}
	
	// Get embedded templates filesystem
	templateFS, err := templates.GetEmbeddedTemplatesFS()
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get templates filesystem")
	}
	
	// Create scaffold engine
	fsys, err := scaffold.NewOSFileSystem(options.OutputDirectory)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create filesystem")
	}
	
	engine, err := scaffold.NewEngine(templateFS, fsys)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}
	
	// Determine scaffold path
	scaffoldPath := fmt.Sprintf("%s/scaffold.yaml", options.TemplateName)
	
	// Load recipe
	recipe, err := engine.LoadRecipeFS(ctx, templateFS, scaffoldPath)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "failed to load scaffold recipe").
			WithDetails("template", options.TemplateName).
			WithDetails("path", scaffoldPath)
	}
	
	// Merge variables with recipe variables
	mergeVariables(recipe, options.Variables)
	
	if options.DryRun {
		return executeDryRun(ctx, engine, recipe, scaffoldOpts, options)
	}
	
	return executeScaffolding(ctx, engine, recipe, scaffoldOpts, options)
}

// createScaffoldOptions creates scaffold.Options from CLI options
func createScaffoldOptions(ctx context.Context, options *InitOptions) (scaffold.Options, error) {
	templateFS, err := templates.GetEmbeddedTemplatesFS()
	if err != nil {
		return scaffold.Options{}, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to get templates filesystem")
	}
	
	return scaffold.Options{
		TemplatesFS:  templateFS,
		ScaffoldPath: fmt.Sprintf("%s/scaffold.yaml", options.TemplateName),
		Dest:         options.OutputDirectory,
		Dry:          options.DryRun,
		Overwrite:    options.Force,
		Vars:         options.Variables,
	}, nil
}

// mergeVariables merges CLI variables into the recipe variables
func mergeVariables(recipe *scaffold.Recipe, variables map[string]interface{}) {
	if recipe.Vars == nil {
		recipe.Vars = make(map[string]any)
	}
	
	for key, value := range variables {
		recipe.Vars[key] = value
	}
}

// executeDryRun performs a dry run of the scaffolding process
func executeDryRun(ctx context.Context, engine scaffold.Engine, recipe *scaffold.Recipe, scaffoldOpts scaffold.Options, options *InitOptions) error {
	fmt.Println("🔍 Dry Run Mode - Preview of changes")
	fmt.Println("====================================")
	fmt.Println()
	
	// Execute dry run
	stats, err := engine.DryRun(ctx, recipe, scaffoldOpts)
	if err != nil {
		return handleScaffoldingError(err)
	}
	
	// Display results
	displayDryRunResults(stats, recipe, options)
	
	fmt.Println("💡 Use the command without --dry-run to execute this plan")
	return nil
}

// executeScaffolding performs the actual scaffolding operation
func executeScaffolding(ctx context.Context, engine scaffold.Engine, recipe *scaffold.Recipe, scaffoldOpts scaffold.Options, options *InitOptions) error {
	fmt.Println("🚀 Executing Scaffold Operation")
	fmt.Println("===============================")
	fmt.Println()
	
	startTime := time.Now()
	
	// Check if output directory exists and is not empty (unless force is used)
	if !options.Force {
		if err := checkOutputDirectory(options.OutputDirectory); err != nil {
			return err
		}
	}
	
	// Execute scaffolding
	stats, err := engine.RenderFS(ctx, recipe, scaffoldOpts)
	if err != nil {
		return handleScaffoldingError(err)
	}
	
	duration := time.Since(startTime)
	
	// Display results
	displayScaffoldingResults(stats, duration, options)
	
	// Show next steps
	showNextSteps(options)
	
	return nil
}

// checkOutputDirectory validates the output directory
func checkOutputDirectory(outputDir string) error {
	// Check if directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		// Directory doesn't exist, that's fine
		return nil
	} else if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to check output directory")
	}
	
	// Directory exists, check if it's empty
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeIO, "failed to read output directory")
	}
	
	if len(entries) > 0 {
		return gerror.New(gerror.ErrCodeAlreadyExists, "output directory is not empty", nil).
			WithDetails("directory", outputDir).
			WithDetails("suggestion", "use --force to overwrite existing files")
	}
	
	return nil
}

// displayDryRunResults displays the results of a dry run
func displayDryRunResults(stats *scaffold.ScaffoldStats, recipe *scaffold.Recipe, options *InitOptions) {
	fmt.Printf("📊 Execution Plan Summary:\n")
	fmt.Printf("   Template: %s\n", options.TemplateName)
	fmt.Printf("   Output directory: %s\n", options.OutputDirectory)
	fmt.Printf("   Total files: %d\n", stats.TotalFiles)
	fmt.Printf("   Templates to parse: %d\n", stats.TemplatesParsed)
	fmt.Println()
	
	// Show variables
	if len(recipe.Vars) > 0 {
		fmt.Println("🔧 Template Variables:")
		for key, value := range recipe.Vars {
			fmt.Printf("   %s = %v\n", key, value)
		}
		fmt.Println()
	}
	
	// Show files that would be created
	fmt.Println("📁 Files to be created:")
	for _, file := range recipe.Files {
		outputPath := filepath.Join(options.OutputDirectory, file.Path)
		status := "✅ new"
		
		// Check if file would be overwritten
		if _, err := os.Stat(outputPath); err == nil {
			if options.Force {
				status = "⚠️  would overwrite"
			} else {
				status = "❌ exists (use --force)"
			}
		}
		
		fmt.Printf("   %s %s\n", status, file.Path)
	}
	fmt.Println()
}

// displayScaffoldingResults displays the results of scaffolding
func displayScaffoldingResults(stats *scaffold.ScaffoldStats, duration time.Duration, options *InitOptions) {
	fmt.Println()
	fmt.Println("✅ Scaffold Operation Complete!")
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Files generated: %d\n", stats.FilesGenerated)
	fmt.Printf("   Files skipped: %d\n", stats.FilesSkipped)
	if stats.FilesFailed > 0 {
		fmt.Printf("   Files failed: %d\n", stats.FilesFailed)
	}
	fmt.Printf("   Templates parsed: %d\n", stats.TemplatesParsed)
	fmt.Println()
}

// handleScaffoldingError provides detailed error handling
func handleScaffoldingError(err error) error {
	fmt.Println("❌ Scaffold Operation Failed")
	fmt.Println("===========================")
	fmt.Println()
	
	fmt.Printf("Error: %v\n", err)
	
	// Provide recovery suggestions based on error type
	fmt.Println()
	fmt.Println("💡 Suggested Actions:")
	
	// Check for common error patterns
	errStr := err.Error()
	switch {
	case contains(errStr, "file exists") || contains(errStr, "already exists"):
		fmt.Println("   • Use --force to overwrite existing files")
		fmt.Println("   • Use --dry-run to preview changes")
		fmt.Println("   • Choose a different output directory")
		
	case contains(errStr, "permission denied") || contains(errStr, "access denied"):
		fmt.Println("   • Check directory permissions")
		fmt.Println("   • Run with appropriate user privileges")
		fmt.Println("   • Verify output directory is writable")
		
	case contains(errStr, "template") || contains(errStr, "parsing"):
		fmt.Println("   • Check template syntax and variables")
		fmt.Println("   • Use --var key=value to set missing variables")
		fmt.Println("   • Try with --list-templates to see available templates")
		
	case contains(errStr, "not found") || contains(errStr, "no such"):
		fmt.Println("   • Verify the template name is correct")
		fmt.Println("   • Use --list-templates to see available templates")
		fmt.Println("   • Check if the output directory path is valid")
		
	default:
		fmt.Println("   • Use --dry-run to preview the operation")
		fmt.Println("   • Check file permissions and disk space")
		fmt.Println("   • Try with --verbose for more details")
	}
	
	fmt.Println()
	return err
}

// showNextSteps provides guidance for what to do after scaffolding
func showNextSteps(options *InitOptions) {
	fmt.Println("🎯 Next Steps:")
	
	// Project-specific guidance based on template
	switch options.TemplateName {
	case "campaign":
		fmt.Println("   1. Review generated configuration files:")
		fmt.Println("      • .campaign/campaign.yaml - Main workspace config")
		fmt.Println("      • .campaign/guilds/ - Guild team configurations") 
		fmt.Println("      • commissions/ - Project specification templates")
		fmt.Println()
		fmt.Println("   2. Initialize the campaign:")
		fmt.Printf("      cd %s\n", options.OutputDirectory)
		fmt.Println("      guild campaign start")
		
	case "guild_core_extension":
		fmt.Println("   1. Review integration points:")
		fmt.Println("      • pkg/ - New package implementations")
		fmt.Println("      • cmd/ - Command extensions")
		fmt.Println("      • internal/ - Internal utilities")
		fmt.Println()
		fmt.Println("   2. Run tests to verify integration:")
		fmt.Printf("      cd %s\n", options.OutputDirectory)
		fmt.Println("      make test")
		
	default:
		fmt.Println("   1. Review the generated project structure")
		fmt.Println("   2. Customize configuration files as needed")
		fmt.Printf("      cd %s\n", options.OutputDirectory)
		fmt.Println("      # Edit configuration files")
	}
	
	fmt.Println()
	fmt.Println("   📚 Documentation:")
	fmt.Println("      • README.md - Project overview and setup")
	fmt.Println("      • Check generated docs/ directory for details")
	fmt.Println()
	fmt.Println("Happy building! 🎉")
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 findSubstring(s, substr)))
}

// findSubstring checks if substr exists in s
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}