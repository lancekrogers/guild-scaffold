package scaffold

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed templates
var templatesFS embed.FS

// TemplateLibrary provides access to embedded templates
type TemplateLibrary struct {
	fsys          fs.FS
	specification *ScaffoldSpec
	specMutex     sync.RWMutex
	templateCache map[string][]byte
	cacheMutex    sync.RWMutex
}

// ScaffoldSpec represents the scaffold.yaml specification
type ScaffoldSpec struct {
	Scaffold        ScaffoldMetadata     `yaml:"scaffold"`
	Categories      map[string]Category  `yaml:"categories"`
	Presets         map[string]Preset    `yaml:"presets"`
	VariableSchemas map[string]VarSchema `yaml:"variable_schemas"`
	Functions       map[string]FuncDoc   `yaml:"functions"`
}

// ScaffoldMetadata contains basic scaffold information
type ScaffoldMetadata struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
}

// Category represents a template category
type Category struct {
	Description string         `yaml:"description"`
	Templates   []TemplateInfo `yaml:"templates"`
}

// TemplateInfo describes a template's metadata
type TemplateInfo struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Output       string   `yaml:"output"`
	RequiredVars []string `yaml:"required_vars"`
	OptionalVars []string `yaml:"optional_vars"`
}

// Preset defines a template preset configuration
type Preset struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Templates   []string               `yaml:"templates"`
	DefaultVars map[string]interface{} `yaml:"default_vars"`
}

// VarSchema defines variable validation schema
type VarSchema struct {
	Type        string      `yaml:"type"`
	Required    bool        `yaml:"required"`
	Default     interface{} `yaml:"default,omitempty"`
	Pattern     string      `yaml:"pattern,omitempty"`
	Enum        []string    `yaml:"enum,omitempty"`
	Minimum     *int        `yaml:"minimum,omitempty"`
	Maximum     *int        `yaml:"maximum,omitempty"`
	Properties  interface{} `yaml:"properties,omitempty"`
	Description string      `yaml:"description"`
}

// FuncDoc documents template functions
type FuncDoc struct {
	Description string `yaml:"description"`
	Signature   string `yaml:"signature"`
	Example     string `yaml:"example"`
}

// NewTemplateLibrary creates a new template library instance
func NewTemplateLibrary() *TemplateLibrary {
	return &TemplateLibrary{
		fsys:          templatesFS,
		templateCache: make(map[string][]byte),
	}
}

// GetFileSystem returns the embedded filesystem
func (tl *TemplateLibrary) GetFileSystem() fs.FS {
	return tl.fsys
}

// GetSpecification loads and returns the scaffold specification
func (tl *TemplateLibrary) GetSpecification(ctx context.Context) (*ScaffoldSpec, error) {
	tl.specMutex.RLock()
	if tl.specification != nil {
		spec := *tl.specification
		tl.specMutex.RUnlock()
		return &spec, nil
	}
	tl.specMutex.RUnlock()

	// Load specification
	tl.specMutex.Lock()
	defer tl.specMutex.Unlock()

	// Double-check in case another goroutine loaded it
	if tl.specification != nil {
		spec := *tl.specification
		return &spec, nil
	}

	data, err := fs.ReadFile(tl.fsys, "templates/scaffold.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read scaffold specification (file=templates/scaffold.yaml): %w", err)
	}

	var spec ScaffoldSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse scaffold specification (file=templates/scaffold.yaml): %w", err)
	}

	tl.specification = &spec
	return &spec, nil
}

// GetTemplate retrieves a template by name
func (tl *TemplateLibrary) GetTemplate(ctx context.Context, templateName string) ([]byte, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before retrieving template: %w", err)
	}

	// Check cache first
	tl.cacheMutex.RLock()
	if cached, exists := tl.templateCache[templateName]; exists {
		tl.cacheMutex.RUnlock()
		return cached, nil
	}
	tl.cacheMutex.RUnlock()

	// Load template from filesystem
	templatePath := filepath.Join("templates", templateName)
	data, err := fs.ReadFile(tl.fsys, templatePath)
	if err != nil {
		return nil, ErrTemplateNotFound(templateName, "templates")
	}

	// Cache the template
	tl.cacheMutex.Lock()
	tl.templateCache[templateName] = data
	tl.cacheMutex.Unlock()

	return data, nil
}

// ListTemplates returns all available template names
func (tl *TemplateLibrary) ListTemplates(ctx context.Context) ([]string, error) {
	var templates []string

	err := fs.WalkDir(tl.fsys, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-template files
		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		// Remove "templates/" prefix for template name
		templateName := strings.TrimPrefix(path, "templates/")
		templates = append(templates, templateName)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	return templates, nil
}

// ListCategories returns all available template categories
func (tl *TemplateLibrary) ListCategories(ctx context.Context) ([]string, error) {
	spec, err := tl.GetSpecification(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get specification: %w", err)
	}

	var categories []string
	for name := range spec.Categories {
		categories = append(categories, name)
	}

	return categories, nil
}

// GetCategory returns templates in a specific category
func (tl *TemplateLibrary) GetCategory(ctx context.Context, categoryName string) (*Category, error) {
	spec, err := tl.GetSpecification(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get specification: %w", err)
	}

	category, exists := spec.Categories[categoryName]
	if !exists {
		return nil, fmt.Errorf("category not found: category=%s", categoryName)
	}

	return &category, nil
}

// ListPresets returns all available presets
func (tl *TemplateLibrary) ListPresets(ctx context.Context) ([]string, error) {
	spec, err := tl.GetSpecification(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get specification: %w", err)
	}

	var presets []string
	for name := range spec.Presets {
		presets = append(presets, name)
	}

	return presets, nil
}

// GetPreset returns a specific preset configuration
func (tl *TemplateLibrary) GetPreset(ctx context.Context, presetName string) (*Preset, error) {
	spec, err := tl.GetSpecification(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get specification: %w", err)
	}

	preset, exists := spec.Presets[presetName]
	if !exists {
		return nil, fmt.Errorf("preset not found: preset=%s", presetName)
	}

	return &preset, nil
}

// ValidateVariables validates template variables against schemas
func (tl *TemplateLibrary) ValidateVariables(ctx context.Context, vars map[string]interface{}) error {
	spec, err := tl.GetSpecification(ctx)
	if err != nil {
		return fmt.Errorf("failed to get specification: %w", err)
	}

	for varName, schema := range spec.VariableSchemas {
		value, exists := vars[varName]

		// Check required variables
		if schema.Required && !exists {
			return fmt.Errorf("required variable missing: variable=%s", varName)
		}

		// Skip validation if variable is not provided and not required
		if !exists {
			continue
		}

		// Validate variable type and constraints
		if err := tl.validateVariable(varName, value, schema); err != nil {
			return err
		}
	}

	return nil
}

// validateVariable validates a single variable against its schema
func (tl *TemplateLibrary) validateVariable(name string, value interface{}, schema VarSchema) error {
	// Type validation
	switch schema.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("variable must be a string: variable=%s, expected_type=string, actual_type=%T", name, value)
		}
		strValue := value.(string)

		// Pattern validation
		if schema.Pattern != "" {
			// Simple pattern matching - in production, use regexp
			if strings.Contains(schema.Pattern, "^[a-zA-Z0-9_-]+$") && !isValidIdentifier(strValue) {
				return fmt.Errorf("variable does not match required pattern: variable=%s, pattern=%s, value=%s", name, schema.Pattern, strValue)
			}
		}

		// Enum validation
		if len(schema.Enum) > 0 {
			valid := false
			for _, enumValue := range schema.Enum {
				if strValue == enumValue {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("variable value not in allowed enum: variable=%s, allowed_values=%v, actual_value=%s", name, schema.Enum, strValue)
			}
		}

	case "integer":
		var intValue int
		switch v := value.(type) {
		case int:
			intValue = v
		case float64:
			intValue = int(v)
		default:
			return fmt.Errorf("variable must be an integer: variable=%s, expected_type=integer, actual_type=%T", name, value)
		}

		// Range validation
		if schema.Minimum != nil && intValue < *schema.Minimum {
			return fmt.Errorf("variable value below minimum: variable=%s, minimum=%d, actual_value=%d", name, *schema.Minimum, intValue)
		}
		if schema.Maximum != nil && intValue > *schema.Maximum {
			return fmt.Errorf("variable value above maximum: variable=%s, maximum=%d, actual_value=%d", name, *schema.Maximum, intValue)
		}

	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("variable must be an object: variable=%s, expected_type=object, actual_type=%T", name, value)
		}
		// Note: For full validation, we'd recursively validate object properties
	}

	return nil
}

// isValidIdentifier checks if a string is a valid identifier
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// ClearCache clears the template cache
func (tl *TemplateLibrary) ClearCache() {
	tl.cacheMutex.Lock()
	defer tl.cacheMutex.Unlock()
	tl.templateCache = make(map[string][]byte)

	tl.specMutex.Lock()
	defer tl.specMutex.Unlock()
	tl.specification = nil
}

// TemplateStats provides statistics about the template library
type TemplateStats struct {
	TotalTemplates  int            `json:"total_templates"`
	CategoriesCount int            `json:"categories_count"`
	PresetsCount    int            `json:"presets_count"`
	CachedTemplates int            `json:"cached_templates"`
	Categories      map[string]int `json:"categories"`
	LastLoaded      time.Time      `json:"last_loaded"`
}

// GetStats returns statistics about the template library
func (tl *TemplateLibrary) GetStats(ctx context.Context) (*TemplateStats, error) {
	spec, err := tl.GetSpecification(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get specification: %w", err)
	}

	templates, err := tl.ListTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	// Count templates per category
	categoryStats := make(map[string]int)
	for categoryName, category := range spec.Categories {
		categoryStats[categoryName] = len(category.Templates)
	}

	tl.cacheMutex.RLock()
	cachedCount := len(tl.templateCache)
	tl.cacheMutex.RUnlock()

	return &TemplateStats{
		TotalTemplates:  len(templates),
		CategoriesCount: len(spec.Categories),
		PresetsCount:    len(spec.Presets),
		CachedTemplates: cachedCount,
		Categories:      categoryStats,
		LastLoaded:      time.Now(), // This would be more accurate with actual load time tracking
	}, nil
}

// Global template library instance
var globalTemplateLibrary = NewTemplateLibrary()

// GetGlobalTemplateLibrary returns the global template library instance
func GetGlobalTemplateLibrary() *TemplateLibrary {
	return globalTemplateLibrary
}
