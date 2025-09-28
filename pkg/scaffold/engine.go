package scaffold

import (
	"context"
	"io/fs"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Engine provides the main scaffold operations interface
type Engine interface {
	// LoadRecipeFS loads and parses a scaffold recipe from filesystem
	LoadRecipeFS(ctx context.Context, fsys fs.FS, path string) (*Recipe, error)
	
	// RenderFS renders all files from a recipe to the filesystem
	RenderFS(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error)
	
	// ValidateRecipe validates a recipe for correctness
	ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError
	
	// SetTemplateFS sets the filesystem containing templates
	SetTemplateFS(fsys fs.FS)
	
	// DryRun performs a dry run without writing files
	DryRun(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error)
}

// ScaffoldEngine implements the Engine interface
type ScaffoldEngine struct {
	parser   Parser
	renderer Renderer
	fileSystem FileSystem
}

// NewEngine creates a new scaffold engine
func NewEngine(templateFS fs.FS, fileSystem FileSystem) (Engine, error) {
	parser, err := NewParser(DefaultParseOptions)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create parser")
	}
	
	renderer := NewRenderer(templateFS, fileSystem)
	
	// Set template filesystem for validation
	if validator, ok := parser.(*yamlParser); ok {
		validator.validator.SetTemplateFS(templateFS)
	}
	
	return &ScaffoldEngine{
		parser:     parser,
		renderer:   renderer,
		fileSystem: fileSystem,
	}, nil
}

// LoadRecipeFS loads and parses a scaffold recipe from filesystem
func (se *ScaffoldEngine) LoadRecipeFS(ctx context.Context, fsys fs.FS, path string) (*Recipe, error) {
	return se.parser.ParseRecipe(ctx, fsys, path)
}

// RenderFS renders all files from a recipe to the filesystem
func (se *ScaffoldEngine) RenderFS(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error) {
	// Validate recipe first
	if validationErrors := se.ValidateRecipe(ctx, recipe); len(validationErrors) > 0 {
		return nil, gerror.New(ErrCodeValidation, "recipe validation failed", nil).
			WithDetails("errors", validationErrors)
	}
	
	// Ensure destination directory exists
	// Note: if fileSystem is OSFileSystem with basePath set to options.Dest,
	// we should create the root directory (".")
	if err := se.fileSystem.MkdirAll(".", 0755); err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeIO, "failed to create destination directory").
			WithDetails("dest", options.Dest)
	}
	
	// Set template filesystem
	if options.TemplatesFS != nil {
		se.renderer.SetTemplateFS(options.TemplatesFS)
	}
	
	// Render all files
	return se.renderer.RenderRecipe(ctx, recipe, options)
}

// ValidateRecipe validates a recipe for correctness
func (se *ScaffoldEngine) ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError {
	return se.parser.ValidateRecipe(ctx, recipe)
}

// SetTemplateFS sets the filesystem containing templates
func (se *ScaffoldEngine) SetTemplateFS(fsys fs.FS) {
	se.renderer.SetTemplateFS(fsys)
	
	// Also update parser validation
	if parser, ok := se.parser.(*yamlParser); ok {
		parser.validator.SetTemplateFS(fsys)
	}
}

// DryRun performs a dry run without writing files
func (se *ScaffoldEngine) DryRun(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error) {
	// Set dry run flag
	dryOptions := options
	dryOptions.Dry = true
	
	return se.RenderFS(ctx, recipe, dryOptions)
}

// ScaffoldFromFS is a convenience function that loads and renders a scaffold in one operation
func ScaffoldFromFS(ctx context.Context, templateFS fs.FS, scaffoldPath string, options Options) (*ScaffoldStats, error) {
	// Create file system
	fileSystem, err := NewOSFileSystem(options.Dest)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create filesystem")
	}
	
	// Create engine
	engine, err := NewEngine(templateFS, fileSystem)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}
	
	// Load recipe
	recipe, err := engine.LoadRecipeFS(ctx, templateFS, scaffoldPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to load recipe")
	}
	
	// Set templates filesystem in options
	options.TemplatesFS = templateFS
	
	// Render
	return engine.RenderFS(ctx, recipe, options)
}

// ScaffoldFromEmbedFS is a convenience function for embedded filesystems
func ScaffoldFromEmbedFS(ctx context.Context, embedFS fs.FS, scaffoldPath string, dest string, vars map[string]any) (*ScaffoldStats, error) {
	options := Options{
		TemplatesFS: embedFS,
		Dest:        dest,
		Vars:        vars,
		Dry:         false,
		Overwrite:   false,
	}
	
	return ScaffoldFromFS(ctx, embedFS, scaffoldPath, options)
}

// DryRunFromFS is a convenience function for dry runs
func DryRunFromFS(ctx context.Context, templateFS fs.FS, scaffoldPath string, options Options) (*ScaffoldStats, error) {
	// Force dry run
	options.Dry = true
	
	return ScaffoldFromFS(ctx, templateFS, scaffoldPath, options)
}

// ValidateFromFS is a convenience function that only validates a recipe
func ValidateFromFS(ctx context.Context, templateFS fs.FS, scaffoldPath string) ([]ValidationError, error) {
	// Create memory filesystem for validation (no actual writes)
	memFS := NewMemoryFileSystem()
	
	// Create engine
	engine, err := NewEngine(templateFS, memFS)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}
	
	// Load recipe
	recipe, err := engine.LoadRecipeFS(ctx, templateFS, scaffoldPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to load recipe")
	}
	
	// Validate
	return engine.ValidateRecipe(ctx, recipe), nil
}

// ScaffoldInfo contains metadata about a scaffold
type ScaffoldInfo struct {
	ScaffoldVersion string            `json:"scaffold_version"`
	TemplatesDir    string            `json:"templates_dir"`
	FileCount       int               `json:"file_count"`
	Variables       map[string]any    `json:"variables"`
	Templates       []string          `json:"templates"`
	EstimatedSize   int64             `json:"estimated_size_bytes"`
	LoadTime        time.Duration     `json:"load_time"`
}

// GetScaffoldInfo returns metadata about a scaffold without rendering
func GetScaffoldInfo(ctx context.Context, templateFS fs.FS, scaffoldPath string) (*ScaffoldInfo, error) {
	start := time.Now()
	
	// Create memory filesystem for info gathering
	memFS := NewMemoryFileSystem()
	
	// Create engine
	engine, err := NewEngine(templateFS, memFS)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to create scaffold engine")
	}
	
	// Load recipe
	recipe, err := engine.LoadRecipeFS(ctx, templateFS, scaffoldPath)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeParsing, "failed to load recipe")
	}
	
	loadTime := time.Since(start)
	
	// Collect unique templates
	templateSet := make(map[string]bool)
	for _, file := range recipe.Files {
		templateSet[file.Template] = true
	}
	
	templates := make([]string, 0, len(templateSet))
	for template := range templateSet {
		templates = append(templates, template)
	}
	
	// Estimate size by reading all templates
	var estimatedSize int64
	for template := range templateSet {
		if data, err := fs.ReadFile(templateFS, template); err == nil {
			estimatedSize += int64(len(data))
		}
	}
	
	return &ScaffoldInfo{
		ScaffoldVersion: recipe.ScaffoldVersion,
		TemplatesDir:    recipe.TemplatesDir,
		FileCount:       len(recipe.Files),
		Variables:       recipe.Vars,
		Templates:       templates,
		EstimatedSize:   estimatedSize,
		LoadTime:        loadTime,
	}, nil
}