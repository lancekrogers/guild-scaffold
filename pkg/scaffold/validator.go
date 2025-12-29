package scaffold

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// semanticValidator performs semantic validation of recipes
type semanticValidator struct {
	templateChecker TemplateChecker
	pathValidator   PathValidator
	variableChecker VariableChecker
}

// newSemanticValidator creates a new semantic validator
func newSemanticValidator() *semanticValidator {
	return &semanticValidator{
		templateChecker: &templateChecker{},
		pathValidator:   &pathValidator{},
		variableChecker: &variableChecker{},
	}
}

// Validate performs comprehensive semantic validation
func (sv *semanticValidator) Validate(ctx context.Context, recipe *Recipe) []ValidationError {
	var errors []ValidationError
	
	// Validate templates exist (if we have a filesystem)
	if sv.templateChecker != nil {
		templateErrors := sv.templateChecker.CheckTemplates(ctx, recipe)
		errors = append(errors, templateErrors...)
	}
	
	// Validate paths are safe
	pathErrors := sv.pathValidator.CheckPaths(recipe)
	errors = append(errors, pathErrors...)
	
	// Validate variable references
	varErrors := sv.variableChecker.CheckVariables(recipe)
	errors = append(errors, varErrors...)
	
	// Validate scaffold version format
	versionErrors := sv.validateScaffoldVersion(recipe)
	errors = append(errors, versionErrors...)
	
	// Validate template directory
	dirErrors := sv.validateTemplatesDirectory(recipe)
	errors = append(errors, dirErrors...)
	
	return errors
}

// SetTemplateFS sets the filesystem for template validation
func (sv *semanticValidator) SetTemplateFS(fsys fs.FS) {
	if tc, ok := sv.templateChecker.(*templateChecker); ok {
		tc.templateFS = fsys
	}
}

// validateScaffoldVersion validates the scaffold version format
func (sv *semanticValidator) validateScaffoldVersion(recipe *Recipe) []ValidationError {
	var errors []ValidationError
	
	// Check semantic version format (simplified)
	versionRegex := regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	
	if !versionRegex.MatchString(recipe.ScaffoldVersion) {
		errors = append(errors, ValidationError{
			Field:   "scaffold_version",
			Message: "must be a valid semantic version (e.g., '1.0.0', '0.1.0-alpha')",
			Value:   recipe.ScaffoldVersion,
			Code:    ErrCodeValidation,
		})
	}
	
	return errors
}

// validateTemplatesDirectory validates the templates directory path
func (sv *semanticValidator) validateTemplatesDirectory(recipe *Recipe) []ValidationError {
	var errors []ValidationError
	
	// Check for invalid characters
	if strings.ContainsAny(recipe.TemplatesDir, `<>:"|?*`) {
		errors = append(errors, ValidationError{
			Field:   "templates_dir",
			Message: "contains invalid characters",
			Value:   recipe.TemplatesDir,
			Code:    ErrCodeInvalidPath,
		})
	}
	
	// Check for path traversal
	if strings.Contains(recipe.TemplatesDir, "..") {
		errors = append(errors, ValidationError{
			Field:   "templates_dir",
			Message: "path traversal not allowed",
			Value:   recipe.TemplatesDir,
			Code:    ErrCodeInvalidPath,
		})
	}
	
	return errors
}

// TemplateChecker validates template existence and syntax
type TemplateChecker interface {
	CheckTemplates(ctx context.Context, recipe *Recipe) []ValidationError
}

// templateChecker implements template validation
type templateChecker struct {
	templateFS fs.FS
}

// CheckTemplates validates that all referenced templates exist and are valid
func (tc *templateChecker) CheckTemplates(ctx context.Context, recipe *Recipe) []ValidationError {
	var errors []ValidationError

	if tc.templateFS == nil {
		// Cannot validate without filesystem - skip
		return errors
	}

	templatesSeen := make(map[string]bool)

	for i, file := range recipe.Files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			errors = append(errors, ValidationError{
				Field:   "validation",
				Message: "validation cancelled",
				Code:    "VALIDATION_CANCELLED",
			})
			return errors
		default:
		}

		// Skip empty templates (e.g., .gitkeep files that have no content)
		// Both "" and "~" are markers for empty files
		if file.Template == "" || file.Template == "~" {
			continue
		}

		templatePath := filepath.Join(recipe.TemplatesDir, file.Template)

		// Check if template path is actually a directory (skip validation for directories)
		if info, err := fs.Stat(tc.templateFS, templatePath); err == nil && info.IsDir() {
			// Skip directories - this can happen with tree format conversions
			continue
		}

		// Check if template exists
		if _, err := fs.Stat(tc.templateFS, templatePath); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].template", i),
				Message: fmt.Sprintf("template not found: %s", file.Template),
				Value:   file.Template,
				Code:    ErrCodeTemplateNotFound,
			})
			continue
		}

		// Check template syntax if not already checked
		if !templatesSeen[file.Template] {
			templatesSeen[file.Template] = true

			if syntaxErrors := tc.validateTemplateSyntax(templatePath, file.Template); len(syntaxErrors) > 0 {
				errors = append(errors, syntaxErrors...)
			}
		}
	}

	return errors
}

// validateTemplateSyntax checks if a template has valid Go template syntax
func (tc *templateChecker) validateTemplateSyntax(templatePath, templateName string) []ValidationError {
	var errors []ValidationError
	
	// Read template content
	content, err := fs.ReadFile(tc.templateFS, templatePath)
	if err != nil {
		errors = append(errors, ValidationError{
			Field:   "template",
			Message: fmt.Sprintf("failed to read template: %v", err),
			Value:   templateName,
			Code:    ErrCodeFileRead,
		})
		return errors
	}
	
	// Try to parse template
	tmpl := template.New(templateName).Funcs(getValidationTemplateFuncMap())
	if _, err := tmpl.Parse(string(content)); err != nil {
		errors = append(errors, ValidationError{
			Field:   "template",
			Message: fmt.Sprintf("invalid template syntax: %v", err),
			Value:   templateName,
			Code:    ErrCodeTemplateRender,
		})
	}
	
	return errors
}

// PathValidator validates file paths
type PathValidator interface {
	CheckPaths(recipe *Recipe) []ValidationError
}

// pathValidator implements path validation
type pathValidator struct{}

// CheckPaths validates that all file paths are safe and valid
func (pv *pathValidator) CheckPaths(recipe *Recipe) []ValidationError {
	var errors []ValidationError
	
	pathsSeen := make(map[string]int)
	
	for i, file := range recipe.Files {
		// Check for duplicate paths
		if prevIndex, exists := pathsSeen[file.Path]; exists {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].path", i),
				Message: fmt.Sprintf("duplicate path found at files[%d]", prevIndex),
				Value:   file.Path,
				Code:    ErrCodeValidation,
			})
		}
		pathsSeen[file.Path] = i
		
		// Validate path safety
		if pathErrors := pv.validatePathSafety(file.Path, i); len(pathErrors) > 0 {
			errors = append(errors, pathErrors...)
		}
		
		// Validate path format
		if formatErrors := pv.validatePathFormat(file.Path, i); len(formatErrors) > 0 {
			errors = append(errors, formatErrors...)
		}
	}
	
	return errors
}

// validatePathSafety ensures paths are safe
func (pv *pathValidator) validatePathSafety(path string, index int) []ValidationError {
	var errors []ValidationError
	
	// Check for absolute paths
	if filepath.IsAbs(path) {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("files[%d].path", index),
			Message: "absolute paths are not allowed",
			Value:   path,
			Code:    ErrCodeInvalidPath,
		})
	}
	
	// Check for path traversal
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") || strings.HasPrefix(cleanPath, "../") {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("files[%d].path", index),
			Message: "path traversal attempts are not allowed",
			Value:   path,
			Code:    ErrCodeInvalidPath,
		})
	}
	
	return errors
}

// validatePathFormat validates path formatting
func (pv *pathValidator) validatePathFormat(path string, index int) []ValidationError {
	var errors []ValidationError
	
	// Check for empty path
	if strings.TrimSpace(path) == "" {
		errors = append(errors, ValidationError{
			Field:   fmt.Sprintf("files[%d].path", index),
			Message: "path cannot be empty",
			Value:   path,
			Code:    ErrCodeValidation,
		})
	}
	
	// Check for invalid characters (Windows restrictions)
	invalidChars := `<>:"|?*`
	for _, char := range invalidChars {
		if strings.ContainsRune(path, char) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].path", index),
				Message: fmt.Sprintf("path contains invalid character: %c", char),
				Value:   path,
				Code:    ErrCodeInvalidPath,
			})
		}
	}
	
	// Check for reserved names (Windows)
	baseName := strings.ToUpper(filepath.Base(path))
	reserved := []string{"CON", "PRN", "AUX", "NUL"}
	for i := 1; i <= 9; i++ {
		reserved = append(reserved, fmt.Sprintf("COM%d", i), fmt.Sprintf("LPT%d", i))
	}
	
	for _, res := range reserved {
		if baseName == res || strings.HasPrefix(baseName, res+".") {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].path", index),
				Message: fmt.Sprintf("reserved filename: %s", baseName),
				Value:   path,
				Code:    ErrCodeInvalidPath,
			})
		}
	}
	
	return errors
}

// VariableChecker validates variable references in templates
type VariableChecker interface {
	CheckVariables(recipe *Recipe) []ValidationError
}

// variableChecker implements variable validation
type variableChecker struct{}

// CheckVariables validates variable usage and references
func (vc *variableChecker) CheckVariables(recipe *Recipe) []ValidationError {
	var errors []ValidationError
	
	// Collect all defined variables
	definedVars := make(map[string]bool)
	
	// Add global variables
	for key := range recipe.Vars {
		definedVars[key] = true
	}
	
	// Check each file entry
	for i, file := range recipe.Files {
		// Validate variable names in With map
		for key, value := range file.With {
			if varErrors := vc.validateVariableName(key, fmt.Sprintf("files[%d].with", i)); len(varErrors) > 0 {
				errors = append(errors, varErrors...)
			}
			
			if valueErrors := vc.validateVariableValue(value, fmt.Sprintf("files[%d].with.%s", i, key)); len(valueErrors) > 0 {
				errors = append(errors, valueErrors...)
			}
		}
	}
	
	// Validate global variable names and values
	for key, value := range recipe.Vars {
		if varErrors := vc.validateVariableName(key, "vars"); len(varErrors) > 0 {
			errors = append(errors, varErrors...)
		}
		
		if valueErrors := vc.validateVariableValue(value, fmt.Sprintf("vars.%s", key)); len(valueErrors) > 0 {
			errors = append(errors, valueErrors...)
		}
	}
	
	return errors
}

// validateVariableName validates variable naming conventions
func (vc *variableChecker) validateVariableName(name, field string) []ValidationError {
	var errors []ValidationError
	
	// Check for empty name
	if strings.TrimSpace(name) == "" {
		errors = append(errors, ValidationError{
			Field:   field,
			Message: "variable name cannot be empty",
			Value:   name,
			Code:    ErrCodeValidation,
		})
		return errors
	}
	
	// Check for valid identifier (letters, numbers, underscores, must start with letter or underscore)
	validName := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validName.MatchString(name) {
		errors = append(errors, ValidationError{
			Field:   field,
			Message: "variable name must be a valid identifier (letters, numbers, underscores, must start with letter or underscore)",
			Value:   name,
			Code:    ErrCodeValidation,
		})
	}
	
	// Check for reserved template variable names
	reserved := []string{"vars", "with", "file", "recipe"}
	for _, res := range reserved {
		if strings.EqualFold(name, res) {
			errors = append(errors, ValidationError{
				Field:   field,
				Message: fmt.Sprintf("variable name '%s' is reserved", res),
				Value:   name,
				Code:    ErrCodeValidation,
			})
		}
	}
	
	return errors
}

// validateVariableValue validates variable values
func (vc *variableChecker) validateVariableValue(value any, field string) []ValidationError {
	var errors []ValidationError
	
	// Check for complex types that might not serialize well
	switch v := value.(type) {
	case map[string]any:
		// Recursively validate nested maps
		for key, nestedValue := range v {
			nestedField := fmt.Sprintf("%s.%s", field, key)
			if keyErrors := vc.validateVariableName(key, nestedField); len(keyErrors) > 0 {
				errors = append(errors, keyErrors...)
			}
			if valueErrors := vc.validateVariableValue(nestedValue, nestedField); len(valueErrors) > 0 {
				errors = append(errors, valueErrors...)
			}
		}
	case []any:
		// Validate array elements
		for i, item := range v {
			nestedField := fmt.Sprintf("%s[%d]", field, i)
			if itemErrors := vc.validateVariableValue(item, nestedField); len(itemErrors) > 0 {
				errors = append(errors, itemErrors...)
			}
		}
	case string, int, int32, int64, float32, float64, bool:
		// These are fine
	case nil:
		// Nil is acceptable
	default:
		errors = append(errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("unsupported variable type: %T", value),
			Value:   value,
			Code:    ErrCodeValidation,
		})
	}
	
	return errors
}

// getValidationTemplateFuncMap returns the template function map for validation.
// This must include ALL functions from the renderer's getTemplateFuncMap() to
// ensure templates that render successfully also validate successfully.
func getValidationTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		// String manipulation
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"title":     strings.Title,
		"trimSpace": strings.TrimSpace,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"split":     strings.Split,
		"join": func(sep string, items []string) string {
			return strings.Join(items, sep)
		},

		// Utilities
		"default": func(defaultVal, val any) any {
			if val == nil || val == "" {
				return defaultVal
			}
			return val
		},
		"empty": func(val any) bool {
			if val == nil {
				return true
			}
			switch v := val.(type) {
			case string:
				return v == ""
			case []any:
				return len(v) == 0
			case map[string]any:
				return len(v) == 0
			default:
				return false
			}
		},
		"not": func(val bool) bool {
			return !val
		},

		// Type checking
		"isString": func(val any) bool {
			_, ok := val.(string)
			return ok
		},
		"isMap": func(val any) bool {
			_, ok := val.(map[string]any)
			return ok
		},
		"isList": func(val any) bool {
			_, ok := val.([]any)
			return ok
		},

		// Path manipulation
		"pathBase":  filepath.Base,
		"pathDir":   filepath.Dir,
		"pathExt":   filepath.Ext,
		"pathJoin":  filepath.Join,
		"pathClean": filepath.Clean,

		// Date/time functions (stubs for validation)
		"now": func() string {
			return ""
		},
		"date": func(format string) string {
			return ""
		},
		"dateISO": func() string {
			return ""
		},

		// Guild-specific functions
		"campaignHash": func(name string) string {
			return ""
		},
		"quote": func(s string) string {
			return `"` + s + `"`
		},
		"indent": func(spaces int, text string) string {
			return text
		},

		// YAML/JSON functions
		"toYAML": func(v any) string {
			return ""
		},
		"toJSON": func(v any) string {
			return "{}"
		},
	}
}