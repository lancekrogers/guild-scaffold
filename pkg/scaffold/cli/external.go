// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-scaffold/pkg/scaffold"
)

// LoadExternalTemplate loads a template from an external file path
func LoadExternalTemplate(ctx context.Context, templatePath string, options *InitOptions) error {
	// Check if the template path exists
	if _, err := os.Stat(templatePath); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeNotFound, "template file not found").
			WithDetails("path", templatePath)
	}

	// Get the directory containing the template
	templateDir := filepath.Dir(templatePath)

	// Create filesystem for the template directory (for loading the YAML)
	templateDirFS := os.DirFS(templateDir)

	// Create output filesystem
	outputFS, err := scaffold.NewOSFileSystem(options.OutputDirectory)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create output filesystem")
	}

	// Load the recipe first to get the templates_dir
	templateFile := filepath.Base(templatePath)
	data, err := fs.ReadFile(templateDirFS, templateFile)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to read template file").
			WithDetails("path", templatePath)
	}

	// Parse recipe to get templates_dir
	parser, err := scaffold.NewParser(scaffold.DefaultParseOptions)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create parser")
	}

	recipe, err := parser.ParseWithOptions(ctx, data, scaffold.DefaultParseOptions)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to parse template").
			WithDetails("path", templatePath)
	}

	// Set up the correct template filesystem by searching multiple paths
	var templateFS fs.FS
	var foundPath string

	if recipe.TemplatesDir != "" {
		// Search paths in order of preference
		searchPaths := []string{
			filepath.Join(templateDir, recipe.TemplatesDir),                    // {yaml_dir}/{templates_dir}
			filepath.Join(templateDir, "templates", recipe.TemplatesDir),       // {yaml_dir}/templates/{templates_dir}
			filepath.Join(templateDir, "..", "templates", recipe.TemplatesDir), // {yaml_dir}/../templates/{templates_dir}
		}

		for _, path := range searchPaths {
			if info, err := os.Stat(path); err == nil && info.IsDir() {
				foundPath = path
				break
			}
		}

		if foundPath != "" {
			templateFS = os.DirFS(foundPath)
		} else {
			// If no path found, use the base directory and let validation catch the error
			templateFS = templateDirFS
		}
	} else {
		// If templates_dir is empty, templates are in the same directory as YAML
		templateFS = templateDirFS
	}

	// Create scaffold engine with correct template filesystem
	engine, err := scaffold.NewEngine(templateFS, outputFS)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}

	// Since we've already resolved the templates directory into the filesystem,
	// clear it from the recipe so the validator doesn't prepend it again
	if foundPath != "" {
		recipe.TemplatesDir = ""
	}

	// Merge variables
	mergeVariables(recipe, options.Variables)

	// Validate the recipe (optional - could be skipped for external templates)
	// Validation will check if templates exist and are valid
	if validationErrors := engine.ValidateRecipe(ctx, recipe); len(validationErrors) > 0 {
		// Format first error message for visibility
		msg := "recipe validation failed"
		if len(validationErrors) > 0 {
			msg = validationErrors[0].Message
			if len(validationErrors) > 1 {
				msg += fmt.Sprintf(" (and %d more errors)", len(validationErrors)-1)
			}
		}
		return gerror.New(gerror.ErrCodeValidation, msg, nil).
			WithDetails("errors", validationErrors)
	}

	// Create scaffold options with correct template filesystem
	scaffoldOpts := scaffold.Options{
		TemplatesFS:  templateFS,
		ScaffoldPath: templateFile,
		Dest:         options.OutputDirectory,
		Dry:          options.DryRun,
		Overwrite:    options.Force,
		Vars:         options.Variables,
	}

	if options.DryRun {
		return executeDryRun(ctx, engine, recipe, scaffoldOpts, options)
	}

	return executeScaffolding(ctx, engine, recipe, scaffoldOpts, options)
}

// IsExternalTemplate checks if the template name refers to an external file
func IsExternalTemplate(templateName string) bool {
	// Check if it's a file path (contains path separator or ends with .yaml/.yml)
	return strings.Contains(templateName, string(filepath.Separator)) ||
		strings.HasSuffix(templateName, ".yaml") ||
		strings.HasSuffix(templateName, ".yml")
}

// ResolveTemplatePath resolves the template path.
// Simplified to only check:
// 1. Absolute paths (used as-is)
// 2. Relative paths (resolved to absolute)
func ResolveTemplatePath(templateName string) (string, error) {
	// If it's already an absolute path, use it
	if filepath.IsAbs(templateName) {
		if _, err := os.Stat(templateName); err == nil {
			return templateName, nil
		}
		return "", gerror.New(gerror.ErrCodeNotFound, "template file not found", nil).
			WithDetails("path", templateName)
	}

	// Check relative to current directory
	if _, err := os.Stat(templateName); err == nil {
		return filepath.Abs(templateName)
	}

	return "", gerror.New(gerror.ErrCodeNotFound, "template file not found", nil).
		WithDetails("template", templateName).
		WithDetails("suggestion", "use 'scaffold list' to see synced templates")
}
