// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/lancekrogers/guild-scaffold/pkg/scaffold"
	"gopkg.in/yaml.v3"
)

// TemplateInfo represents information about an available template
type TemplateInfo struct {
	Name        string             `json:"name" yaml:"name"`
	Description string             `json:"description" yaml:"description"`
	Category    string             `json:"category" yaml:"category"`
	Source      string             `json:"source" yaml:"source"`
	Builtin     bool               `json:"builtin,omitempty" yaml:"builtin,omitempty"`
	Variables   []TemplateVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// TemplateVariable represents a configurable variable in a template
type TemplateVariable struct {
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description" yaml:"description"`
	Type        string      `json:"type" yaml:"type"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool        `json:"required" yaml:"required"`
}

// ValidationResult represents the result of template validation
type ValidationResult struct {
	Valid    bool              `json:"valid" yaml:"valid"`
	Template string            `json:"template" yaml:"template"`
	Errors   []ValidationError `json:"errors,omitempty" yaml:"errors,omitempty"`
	Warnings []string          `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Type       string `json:"type" yaml:"type"`
	Message    string `json:"message" yaml:"message"`
	Field      string `json:"field,omitempty" yaml:"field,omitempty"`
	Line       int    `json:"line,omitempty" yaml:"line,omitempty"`
	Suggestion string `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
}

// ListTemplates lists available templates with basic output
func ListTemplates(ctx context.Context) error {
	options := &ListOptions{
		Verbose: false,
		Format:  "table",
	}

	return ListTemplatesWithOptions(ctx, options)
}

// ListTemplatesWithOptions lists available templates with specified options
func ListTemplatesWithOptions(ctx context.Context, options *ListOptions) error {
	if err := options.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid list options")
	}

	templates, err := getAvailableTemplates(ctx)
	if err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to load templates")
	}

	switch options.Format {
	case "table":
		return displayTemplatesTable(templates, options.Verbose)
	case "json":
		return displayTemplatesJSON(templates)
	case "yaml":
		return displayTemplatesYAML(templates)
	default:
		return gerror.New(gerror.ErrCodeInvalidInput, "unsupported format", nil).WithDetails("format", options.Format)
	}
}

// ValidateScaffold validates a scaffold configuration
func ValidateScaffold(ctx context.Context, options *ValidateOptions) error {
	if err := options.Validate(); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeValidation, "invalid validate options")
	}

	var result ValidationResult

	if options.Template != "" {
		// Validate template from registry
		result = validateRegistryTemplate(ctx, options.Template)
	} else {
		// Validate file
		result = validateScaffoldFile(ctx, options.ScaffoldPath)
	}

	switch options.Format {
	case "text":
		return displayValidationText(result, options.Verbose)
	case "json":
		return displayValidationJSON(result)
	case "yaml":
		return displayValidationYAML(result)
	default:
		return gerror.New(gerror.ErrCodeInvalidInput, "unsupported format", nil).WithDetails("format", options.Format)
	}
}

// getAvailableTemplates loads templates from the registry
func getAvailableTemplates(ctx context.Context) ([]TemplateInfo, error) {
	loader, err := scaffold.NewRegistryLoader()
	if err != nil {
		return nil, err
	}

	registry, err := loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	entries := registry.List()
	templates := make([]TemplateInfo, 0, len(entries))

	for _, entry := range entries {
		info := TemplateInfo{
			Name:        entry.Name,
			Description: entry.Description,
			Category:    entry.Category,
			Source:      entry.Source,
			Builtin:     entry.Builtin,
		}

		// Try to load variables from scaffold definition
		if def, _, err := scaffold.ResolveScaffold(ctx, entry); err == nil {
			info.Variables = convertVariables(def.Variables)
		}

		templates = append(templates, info)
	}

	return templates, nil
}

// convertVariables converts scaffold variables to template variables
func convertVariables(vars map[string]scaffold.VariableDefinition) []TemplateVariable {
	result := make([]TemplateVariable, 0, len(vars))
	for name, def := range vars {
		result = append(result, TemplateVariable{
			Name:        name,
			Description: def.Description,
			Type:        def.Type,
			Default:     def.Default,
			Required:    def.Required,
		})
	}
	return result
}

// displayTemplatesTable displays templates in table format
func displayTemplatesTable(templates []TemplateInfo, verbose bool) error {
	if len(templates) == 0 {
		fmt.Println("No templates found.")
		fmt.Println()
		fmt.Println("💡 To get started, sync templates from GitHub:")
		fmt.Println("   scaffold sync")
		fmt.Println()
		fmt.Println("Or create workspace templates in:")
		fmt.Println("   .campaign/templates/")
		return nil
	}

	fmt.Println("📋 Available Templates:")
	fmt.Println()

	for _, tmpl := range templates {
		// Format source indicator
		var sourceTag string
		switch tmpl.Source {
		case "workspace":
			sourceTag = "[workspace]"
		case "global":
			sourceTag = "[global]"
		default:
			sourceTag = fmt.Sprintf("[%s]", tmpl.Source)
		}

		fmt.Printf("  %s %s\n", tmpl.Name, sourceTag)
		if tmpl.Description != "" {
			fmt.Printf("    %s\n", tmpl.Description)
		}

		if verbose {
			if tmpl.Category != "" {
				fmt.Printf("    Category: %s\n", tmpl.Category)
			}
			if len(tmpl.Variables) > 0 {
				fmt.Println("    Variables:")
				for _, variable := range tmpl.Variables {
					required := ""
					if variable.Required {
						required = " (required)"
					}
					defaultStr := ""
					if variable.Default != nil {
						defaultStr = fmt.Sprintf(" [default: %v]", variable.Default)
					}
					fmt.Printf("      • %s (%s)%s%s\n",
						variable.Name, variable.Type, required, defaultStr)
					if variable.Description != "" {
						fmt.Printf("        %s\n", variable.Description)
					}
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("💡 Use 'scaffold init --template <name>' to create a project")
	if !verbose {
		fmt.Println("💡 Use --verbose for variable details")
	}
	fmt.Println("💡 Use 'scaffold sync' to download/update templates")

	return nil
}

// displayTemplatesJSON displays templates in JSON format
func displayTemplatesJSON(templates []TemplateInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"templates": templates,
	})
}

// displayTemplatesYAML displays templates in YAML format
func displayTemplatesYAML(templates []TemplateInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(map[string]interface{}{
		"templates": templates,
	})
}

// validateRegistryTemplate validates a template from the registry
func validateRegistryTemplate(ctx context.Context, templateName string) ValidationResult {
	loader, err := scaffold.NewRegistryLoader()
	if err != nil {
		return ValidationResult{
			Valid:    false,
			Template: templateName,
			Errors: []ValidationError{
				{
					Type:    "loader_error",
					Message: fmt.Sprintf("Failed to create registry loader: %v", err),
				},
			},
		}
	}

	entry, err := loader.FindScaffold(ctx, templateName)
	if err != nil {
		return ValidationResult{
			Valid:    false,
			Template: templateName,
			Errors: []ValidationError{
				{
					Type:       "template_not_found",
					Message:    fmt.Sprintf("Template '%s' not found in registry", templateName),
					Suggestion: "Use 'scaffold list' to see available templates",
				},
			},
		}
	}

	def, _, err := scaffold.ResolveScaffold(ctx, entry)
	if err != nil {
		return ValidationResult{
			Valid:    false,
			Template: templateName,
			Errors: []ValidationError{
				{
					Type:    "resolution_error",
					Message: fmt.Sprintf("Failed to resolve scaffold: %v", err),
				},
			},
		}
	}

	if err := scaffold.ValidateScaffoldDefinition(def); err != nil {
		return ValidationResult{
			Valid:    false,
			Template: templateName,
			Errors: []ValidationError{
				{
					Type:    "validation_error",
					Message: err.Error(),
				},
			},
		}
	}

	return ValidationResult{
		Valid:    true,
		Template: templateName,
	}
}

// validateScaffoldFile validates a scaffold file
func validateScaffoldFile(ctx context.Context, filePath string) ValidationResult {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return ValidationResult{
			Valid:    false,
			Template: filePath,
			Errors: []ValidationError{
				{
					Type:    "file_not_found",
					Message: fmt.Sprintf("Scaffold file not found: %s", filePath),
				},
			},
		}
	}

	def, err := scaffold.LoadScaffoldDefinition(ctx, filePath)
	if err != nil {
		return ValidationResult{
			Valid:    false,
			Template: filePath,
			Errors: []ValidationError{
				{
					Type:    "parse_error",
					Message: fmt.Sprintf("Failed to parse scaffold: %v", err),
				},
			},
		}
	}

	if err := scaffold.ValidateScaffoldDefinition(def); err != nil {
		return ValidationResult{
			Valid:    false,
			Template: filePath,
			Errors: []ValidationError{
				{
					Type:    "validation_error",
					Message: err.Error(),
				},
			},
		}
	}

	return ValidationResult{
		Valid:    true,
		Template: filePath,
	}
}

// displayValidationText displays validation results in text format
func displayValidationText(result ValidationResult, verbose bool) error {
	if result.Valid {
		fmt.Printf("✅ Template '%s' is valid\n", result.Template)

		if len(result.Warnings) > 0 {
			fmt.Println("\n⚠️  Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("   • %s\n", warning)
			}
		}
	} else {
		fmt.Printf("❌ Template '%s' validation failed\n", result.Template)
		fmt.Println("\nErrors:")

		for _, err := range result.Errors {
			errType := strings.ReplaceAll(err.Type, "_", " ")
			fmt.Printf("   • %s: %s\n", strings.Title(errType), err.Message)
			if err.Field != "" {
				fmt.Printf("     Field: %s\n", err.Field)
			}
			if err.Line > 0 {
				fmt.Printf("     Line: %d\n", err.Line)
			}
			if err.Suggestion != "" {
				fmt.Printf("     💡 %s\n", err.Suggestion)
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("   • %s\n", warning)
			}
		}
	}

	return nil
}

// displayValidationJSON displays validation results in JSON format
func displayValidationJSON(result ValidationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// displayValidationYAML displays validation results in YAML format
func displayValidationYAML(result ValidationResult) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(result)
}

// DetectTemplateFromContext detects the appropriate template based on project context.
// Returns the builtin scaffold name if no specific context is detected.
func DetectTemplateFromContext(ctx context.Context, outputDir string) string {
	// For now, just return the builtin scaffold
	// Future: detect based on existing files (go.mod, package.json, .campaign/, etc.)
	return scaffold.BuiltinScaffoldName
}
