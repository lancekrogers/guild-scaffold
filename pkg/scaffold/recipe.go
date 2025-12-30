package scaffold

import (
	"io/fs"
	"time"
)

// Recipe represents a scaffold specification loaded from YAML
type Recipe struct {
	// ScaffoldVersion specifies the version of the scaffold format
	ScaffoldVersion string `yaml:"scaffold_version" json:"scaffold_version"`

	// TemplatesDir specifies the directory containing templates within the embedded filesystem
	TemplatesDir string `yaml:"templates_dir" json:"templates_dir"`

	// Vars contains global variables available to all templates
	Vars map[string]any `yaml:"vars,omitempty" json:"vars,omitempty"`

	// Files specifies the list of files to generate from templates
	Files []FileEntry `yaml:"files" json:"files"`
}

// FileEntry represents a single file to generate from a template
type FileEntry struct {
	// Path specifies the destination path relative to the output directory
	Path string `yaml:"path" json:"path"`

	// Template specifies the template file name within the templates directory
	Template string `yaml:"template" json:"template"`

	// With contains file-specific variables that override global vars
	With map[string]any `yaml:"with,omitempty" json:"with,omitempty"`
}

// Options configures scaffold rendering behavior
type Options struct {
	// TemplatesFS is the filesystem containing templates
	TemplatesFS fs.FS

	// ScaffoldPath is the path to the scaffold.yaml file within TemplatesFS
	ScaffoldPath string

	// Dest is the destination directory for generated files
	Dest string

	// Dry enables preview mode without writing files
	Dry bool

	// Overwrite allows overwriting existing files (normally forbidden)
	Overwrite bool

	// Vars provides additional variables to merge with recipe vars
	Vars map[string]any
}

// ParseOptions configures YAML parsing behavior
type ParseOptions struct {
	// StrictMode causes parsing to fail on unknown fields
	StrictMode bool

	// MaxFileSize limits the maximum YAML file size in bytes
	MaxFileSize int64

	// MaxParseTime limits the maximum time spent parsing
	MaxParseTime time.Duration

	// ValidateSchema enables JSON schema validation
	ValidateSchema bool

	// ValidateSemantics enables semantic validation (template existence, etc.)
	ValidateSemantics bool

	// AllowExtensions permits vendor-specific extensions
	AllowExtensions bool
}

// DefaultParseOptions provides production-ready parsing defaults
var DefaultParseOptions = ParseOptions{
	StrictMode:        true,
	MaxFileSize:       1024 * 1024, // 1MB
	MaxParseTime:      time.Second,
	ValidateSchema:    true,
	ValidateSemantics: true,
	AllowExtensions:   false,
}

// RenderContext provides template rendering context
type RenderContext struct {
	// Vars contains all variables (global + file-specific)
	Vars map[string]any

	// File contains the current file entry being processed
	File FileEntry

	// Recipe contains the complete recipe for reference
	Recipe *Recipe
}

// ScaffoldStats contains statistics about scaffold execution
type ScaffoldStats struct {
	// FilesGenerated is the number of files successfully generated
	FilesGenerated int

	// FilesSkipped is the number of files skipped (already exist)
	FilesSkipped int

	// FilesFailed is the number of files that failed to generate
	FilesFailed int

	// TotalFiles is the total number of files in the recipe
	TotalFiles int

	// Duration is the total time taken for scaffolding
	Duration time.Duration

	// TemplatesParsed is the number of unique templates parsed
	TemplatesParsed int
}
