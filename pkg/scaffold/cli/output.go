// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-scaffold/pkg/scaffold/templates"
	"gopkg.in/yaml.v3"
)

// TemplateInfo represents information about an available template
type TemplateInfo struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	UseCase     string `json:"use_case" yaml:"use_case"`
	Category    string `json:"category" yaml:"category"`
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
	Type        string `json:"type" yaml:"type"`
	Message     string `json:"message" yaml:"message"`
	Field       string `json:"field,omitempty" yaml:"field,omitempty"`
	Line        int    `json:"line,omitempty" yaml:"line,omitempty"`
	Suggestion  string `json:"suggestion,omitempty" yaml:"suggestion,omitempty"`
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
	
	templates := getAvailableTemplates()
	
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
		// Validate built-in template
		result = validateBuiltinTemplate(ctx, options.Template)
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

// DetectTemplateFromContext automatically detects the appropriate template
func DetectTemplateFromContext(ctx context.Context, outputDir string) string {
	// Check for existing project indicators
	detectors := []templateDetector{
		{
			name:     "existing_campaign",
			pattern:  ".campaign/campaign.yaml",
			template: "campaign",
		},
		{
			name:     "existing_guild_core",
			pattern:  "pkg/agent/interface.go",
			template: "guild_core_extension",
		},
		{
			name:     "go_module",
			pattern:  "go.mod",
			template: "single_agent",
		},
		{
			name:     "typescript_project",
			pattern:  "package.json",
			template: "single_agent",
		},
	}
	
	// Check each detector
	for _, detector := range detectors {
		if detector.matches(outputDir) {
			fmt.Printf("🔍 Detected %s, using %s template\n", 
				detector.name, detector.template)
			return detector.template
		}
	}
	
	// Default to campaign template
	fmt.Println("📝 No existing project detected, using campaign template")
	return "campaign"
}

// templateDetector represents a way to detect project type
type templateDetector struct {
	name     string
	pattern  string
	template string
}

// matches checks if the detector pattern matches the directory
func (d *templateDetector) matches(dir string) bool {
	path := filepath.Join(dir, d.pattern)
	_, err := os.Stat(path)
	return err == nil
}

// getAvailableTemplates returns the list of available templates
func getAvailableTemplates() []TemplateInfo {
	templates := []TemplateInfo{
		{
			Name:        "campaign",
			Description: "Complete campaign workspace with guild configuration",
			UseCase:     "New multi-project workspace with coordinated guilds",
			Category:    "workspace",
			Variables: []TemplateVariable{
				{Name: "project_name", Description: "Name of the campaign project", Type: "string", Required: true},
				{Name: "author_name", Description: "Primary author name", Type: "string", Required: false},
				{Name: "coordination_style", Description: "Guild coordination approach", Type: "string", Default: "collaborative"},
			},
		},
		{
			Name:        "guild_core_extension",
			Description: "Extension to existing guild-core repository",
			UseCase:     "Adding new features or capabilities to guild-core",
			Category:    "extension",
			Variables: []TemplateVariable{
				{Name: "extension_type", Description: "Type of extension", Type: "string", Default: "full-feature"},
				{Name: "package_name", Description: "Go package name", Type: "string", Required: true},
			},
		},
		{
			Name:        "single_agent",
			Description: "Simple single-agent project",
			UseCase:     "Rapid prototyping or simple automation tasks",
			Category:    "agent",
			Variables: []TemplateVariable{
				{Name: "agent_role", Description: "Primary role of the agent", Type: "string", Default: "assistant"},
				{Name: "agent_capabilities", Description: "Agent capabilities", Type: "array", Required: false},
			},
		},
		{
			Name:        "multi_guild",
			Description: "Multiple coordinated guilds",
			UseCase:     "Large-scale multi-team projects with complex coordination",
			Category:    "workspace",
			Variables: []TemplateVariable{
				{Name: "guild_count", Description: "Number of guilds", Type: "int", Default: 3},
				{Name: "coordination_pattern", Description: "Inter-guild coordination", Type: "string", Default: "hierarchical"},
			},
		},
		{
			Name:        "research_project",
			Description: "Research and experimentation workspace",
			UseCase:     "AI research, experimentation, and exploration",
			Category:    "research",
			Variables: []TemplateVariable{
				{Name: "research_area", Description: "Primary research focus", Type: "string", Required: false},
				{Name: "experiment_tracking", Description: "Enable experiment tracking", Type: "bool", Default: true},
			},
		},
	}
	
	// Add external templates from examples directory
	ctx := context.Background()
	externalTemplates, err := ListExternalTemplates(ctx)
	if err == nil && len(externalTemplates) > 0 {
		for _, path := range externalTemplates {
			// Extract name from path (e.g., "examples/minimal.yaml" -> "minimal")
			name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			
			// Create template info for external template
			templates = append(templates, TemplateInfo{
				Name:        path,  // Use full path as name for clarity
				Description: fmt.Sprintf("External template: %s", name),
				UseCase:     "Custom project template from examples",
				Category:    "external",
			})
		}
	}
	
	return templates
}

// displayTemplatesTable displays templates in table format
func displayTemplatesTable(templates []TemplateInfo, verbose bool) error {
	fmt.Println("📋 Available Templates:")
	fmt.Println()
	
	for _, tmpl := range templates {
		fmt.Printf("🎯 %s\n", tmpl.Name)
		fmt.Printf("   %s\n", tmpl.Description)
		fmt.Printf("   Use case: %s\n", tmpl.UseCase)
		fmt.Printf("   Category: %s\n", tmpl.Category)
		
		if verbose && len(tmpl.Variables) > 0 {
			fmt.Println("   Variables:")
			for _, variable := range tmpl.Variables {
				required := ""
				if variable.Required {
					required = " (required)"
				}
				defaultStr := ""
				if variable.Default != nil {
					defaultStr = fmt.Sprintf(" [default: %v]", variable.Default)
				}
				fmt.Printf("     • %s (%s)%s%s - %s\n", 
					variable.Name, variable.Type, required, defaultStr, variable.Description)
			}
		}
		fmt.Println()
	}
	
	fmt.Println("💡 Use --template <name> to select a specific template")
	fmt.Println("💡 Use --interactive for guided template selection")
	
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

// validateBuiltinTemplate validates a built-in template
func validateBuiltinTemplate(ctx context.Context, templateName string) ValidationResult {
	// Get embedded templates filesystem
	templateFS, err := templates.GetEmbeddedTemplatesFS()
	if err != nil {
		return ValidationResult{
			Valid:    false,
			Template: templateName,
			Errors: []ValidationError{
				{
					Type:    "filesystem_error",
					Message: fmt.Sprintf("Failed to access templates filesystem: %v", err),
				},
			},
		}
	}
	
	// Construct scaffold path
	scaffoldPath := fmt.Sprintf("scaffolds/%s/scaffold.yaml", templateName)
	
	// Try to read and validate the scaffold file
	_, err = templateFS.Open(scaffoldPath)
	if err != nil {
		return ValidationResult{
			Valid:    false,
			Template: templateName,
			Errors: []ValidationError{
				{
					Type:       "template_not_found",
					Message:    fmt.Sprintf("Template '%s' not found", templateName),
					Suggestion: "Use 'scaffold list' to see available templates",
				},
			},
		}
	}
	
	// TODO: Add more comprehensive validation once the scaffold engine supports it
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
	
	// TODO: Add comprehensive file validation once the scaffold engine supports it
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
			fmt.Printf("   • %s: %s\n", strings.Title(strings.ReplaceAll(err.Type, "_", " ")), err.Message)
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