package scaffold

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

// Renderer handles template rendering operations
type Renderer interface {
	// RenderTemplate renders a single template with the given context
	RenderTemplate(ctx context.Context, templateName string, context RenderContext) ([]byte, error)

	// RenderRecipe renders all files defined in a recipe
	RenderRecipe(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error)

	// SetTemplateFS sets the filesystem containing templates
	SetTemplateFS(fsys fs.FS)
}

// templateRenderer implements the Renderer interface
type templateRenderer struct {
	templateFS    fs.FS
	templateCache *TemplateCache
	funcMap       template.FuncMap
	fileSystem    FileSystem
}

// NewRenderer creates a new template renderer
func NewRenderer(fsys fs.FS, fileSystem FileSystem) Renderer {
	return &templateRenderer{
		templateFS:    fsys,
		templateCache: NewTemplateCache(100, 10*time.Minute),
		funcMap:       getTemplateFuncMap(),
		fileSystem:    fileSystem,
	}
}

// SetTemplateFS sets the filesystem containing templates
func (tr *templateRenderer) SetTemplateFS(fsys fs.FS) {
	tr.templateFS = fsys
	// Clear cache when filesystem changes
	tr.templateCache.Clear()
}

// RenderTemplate renders a single template with the given context
func (tr *templateRenderer) RenderTemplate(ctx context.Context, templateName string, context RenderContext) ([]byte, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled before rendering: %w", err)
	}

	// Get or parse template
	tmpl, err := tr.getTemplate(ctx, templateName, context.Recipe.TemplatesDir)
	if err != nil {
		return nil, ErrTemplateRender(templateName, err)
	}

	// Prepare template data
	data := tr.prepareTemplateData(context)

	// Render template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, ErrTemplateRender(templateName, err)
	}

	return buf.Bytes(), nil
}

// RenderRecipe renders all files defined in a recipe
func (tr *templateRenderer) RenderRecipe(ctx context.Context, recipe *Recipe, options Options) (*ScaffoldStats, error) {
	stats := &ScaffoldStats{
		TotalFiles:   len(recipe.Files),
		CreatedFiles: []string{},
		CreatedDirs:  []string{},
		SkippedPaths: []string{},
	}

	startTime := time.Now()
	defer func() {
		stats.Duration = time.Since(startTime)
	}()

	// Track unique templates for stats
	templatesUsed := make(map[string]bool)

	// Track created directories to avoid duplicates
	createdDirsMap := make(map[string]bool)

	// Process each file
	for i, file := range recipe.Files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return stats, fmt.Errorf("rendering cancelled: %w", ctx.Err())
		default:
		}

		// Track template usage (only for non-empty templates)
		if file.Template != "" && file.Template != "~" {
			templatesUsed[file.Template] = true
		}

		// Check if file already exists
		// Note: file.Path is relative, and fileSystem already has basePath set
		if tr.fileExists(file.Path) && !options.Overwrite {
			stats.FilesSkipped++
			stats.SkippedPaths = append(stats.SkippedPaths, file.Path)
			continue
		}

		// Handle symlinks separately
		if file.IsSymlink() {
			// Write symlink (or skip in dry run)
			if !options.Dry {
				// Remove existing file if overwrite is enabled
				if options.Overwrite && tr.fileExists(file.Path) {
					if err := tr.fileSystem.Remove(file.Path); err != nil {
						stats.FilesFailed++
						return stats, fmt.Errorf("failed to remove existing file for symlink overwrite (filePath=%v): %w", file.Path, err)
					}
				}

				// Track parent directory creation
				dir := filepath.Dir(file.Path)
				if dir != "." && dir != "" && !createdDirsMap[dir] {
					createdDirsMap[dir] = true
					stats.CreatedDirs = append(stats.CreatedDirs, dir)
				}

				// Create the symlink
				if err := tr.createSymlink(ctx, file.Path, file.SymlinkTo); err != nil {
					stats.FilesFailed++
					return stats, fmt.Errorf("failed to create symlink (filePath=%v, target=%v): %w", file.Path, file.SymlinkTo, err)
				}
			}

			stats.FilesGenerated++
			stats.CreatedFiles = append(stats.CreatedFiles, file.Path)
			continue
		}

		// Prepare render context
		renderCtx := tr.prepareRenderContext(recipe, file, options)

		var content []byte
		var err error

		// Handle empty template markers (for .gitkeep files and empty directories)
		if file.Template == "" || file.Template == "~" {
			content = []byte{}
		} else {
			// Render template
			content, err = tr.RenderTemplate(ctx, file.Template, renderCtx)
			if err != nil {
				stats.FilesFailed++

				// In dry run or continue on error, log and continue
				if options.Dry {
					continue
				}

				return stats, fmt.Errorf("failed to render file (fileIndex=%v): %w", i, err)
			}
		}

		// Write file (or skip in dry run)
		if !options.Dry {
			// Remove existing file if overwrite is enabled
			if options.Overwrite && tr.fileExists(file.Path) {
				if err := tr.fileSystem.Remove(file.Path); err != nil {
					stats.FilesFailed++
					return stats, fmt.Errorf("failed to remove existing file for overwrite (filePath=%v): %w", file.Path, err)
				}
			}

			// Track parent directory creation
			dir := filepath.Dir(file.Path)
			if dir != "." && dir != "" && !createdDirsMap[dir] {
				createdDirsMap[dir] = true
				stats.CreatedDirs = append(stats.CreatedDirs, dir)
			}

			// Use relative path since fileSystem has basePath configured
			if err := tr.writeFile(ctx, file.Path, content); err != nil {
				stats.FilesFailed++
				return stats, fmt.Errorf("failed to write file (filePath=%v): %w", file.Path, err)
			}
		}

		stats.FilesGenerated++
		stats.CreatedFiles = append(stats.CreatedFiles, file.Path)
	}

	stats.TemplatesParsed = len(templatesUsed)
	return stats, nil
}

// getTemplate retrieves or parses a template
func (tr *templateRenderer) getTemplate(ctx context.Context, templateName, templatesDir string) (*template.Template, error) {
	// Check cache first
	if cached := tr.templateCache.Get(templateName); cached != nil {
		return cached, nil
	}

	// Read template file
	templatePath := filepath.Join(templatesDir, templateName)
	content, err := fs.ReadFile(tr.templateFS, templatePath)
	if err != nil {
		return nil, ErrTemplateNotFound(templateName, templatesDir)
	}

	// Parse template
	tmpl := template.New(templateName).Funcs(tr.funcMap)
	parsedTemplate, err := tmpl.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template (template=%v): %w", templateName, err)
	}

	// Cache parsed template
	tr.templateCache.Set(templateName, parsedTemplate)

	return parsedTemplate, nil
}

// prepareRenderContext creates a render context for a file
func (tr *templateRenderer) prepareRenderContext(recipe *Recipe, file FileEntry, options Options) RenderContext {
	// Merge variables: recipe vars + options vars + file vars
	vars := make(map[string]any)

	// Start with recipe vars
	for k, v := range recipe.Vars {
		vars[k] = v
	}

	// Override with options vars
	for k, v := range options.Vars {
		vars[k] = v
	}

	// Override with file-specific vars
	for k, v := range file.With {
		vars[k] = v
	}

	return RenderContext{
		Vars:   vars,
		File:   file,
		Recipe: recipe,
	}
}

// prepareTemplateData creates the data structure passed to templates
func (tr *templateRenderer) prepareTemplateData(context RenderContext) map[string]any {
	return map[string]any{
		"vars":   context.Vars,
		"with":   context.File.With,
		"file":   context.File,
		"recipe": context.Recipe,
	}
}

// fileExists checks if a file already exists
func (tr *templateRenderer) fileExists(path string) bool {
	_, err := tr.fileSystem.Stat(path)
	return err == nil
}

// writeFile writes content to a file, creating directories as needed
func (tr *templateRenderer) writeFile(ctx context.Context, path string, content []byte) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before writing file: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := tr.fileSystem.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory (directory=%v): %w", dir, err)
	}

	// Write file
	if err := tr.fileSystem.WriteFile(path, content, 0644); err != nil {
		return ErrFileWrite(path, err)
	}

	return nil
}

// createSymlink creates a symbolic link, creating parent directories as needed
func (tr *templateRenderer) createSymlink(ctx context.Context, linkPath, target string) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before creating symlink: %w", err)
	}

	// Create directory if needed
	dir := filepath.Dir(linkPath)
	if dir != "." && dir != "" {
		if err := tr.fileSystem.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for symlink (directory=%v): %w", dir, err)
		}
	}

	// Create symlink
	if err := tr.fileSystem.Symlink(target, linkPath); err != nil {
		return fmt.Errorf("failed to create symlink (linkPath=%v, target=%v): %w", linkPath, target, err)
	}

	return nil
}

// TemplateCache provides caching for parsed templates
type TemplateCache struct {
	cache   map[string]*templateCacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// templateCacheEntry represents a cached template
type templateCacheEntry struct {
	template  *template.Template
	timestamp time.Time
	hits      int64
}

// NewTemplateCache creates a new template cache
func NewTemplateCache(maxSize int, ttl time.Duration) *TemplateCache {
	return &TemplateCache{
		cache:   make(map[string]*templateCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a cached template
func (tc *TemplateCache) Get(name string) *template.Template {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	entry, exists := tc.cache[name]
	if !exists {
		return nil
	}

	// Check TTL
	if time.Since(entry.timestamp) > tc.ttl {
		return nil
	}

	entry.hits++
	return entry.template
}

// Set caches a parsed template
func (tc *TemplateCache) Set(name string, tmpl *template.Template) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Evict if at capacity
	if len(tc.cache) >= tc.maxSize {
		tc.evictLRU()
	}

	tc.cache[name] = &templateCacheEntry{
		template:  tmpl,
		timestamp: time.Now(),
		hits:      1,
	}
}

// Clear clears all cached templates
func (tc *TemplateCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache = make(map[string]*templateCacheEntry)
}

// evictLRU removes the least recently used template
func (tc *TemplateCache) evictLRU() {
	var oldestName string
	var oldestTime time.Time = time.Now()

	for name, entry := range tc.cache {
		if entry.timestamp.Before(oldestTime) {
			oldestTime = entry.timestamp
			oldestName = name
		}
	}

	if oldestName != "" {
		delete(tc.cache, oldestName)
	}
}

// Enhanced template function map with additional helpers
func getTemplateFuncMap() template.FuncMap {
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

		// Date/time functions
		"now": func() time.Time {
			return time.Now()
		},
		"date": func(format string) string {
			return time.Now().Format(format)
		},
		"dateISO": func() string {
			return time.Now().Format(time.RFC3339)
		},

		// Guild-specific functions
		"campaignHash": func(name string) string {
			// Generate a simple hash for campaign names
			return generateSimpleHash(name)
		},
		"quote": func(s string) string {
			return `"` + s + `"`
		},
		"indent": func(spaces int, text string) string {
			indent := strings.Repeat(" ", spaces)
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if strings.TrimSpace(line) != "" {
					lines[i] = indent + line
				}
			}
			return strings.Join(lines, "\n")
		},

		// YAML/JSON functions
		"toYAML": func(v any) string {
			// Simple YAML serialization for basic types
			return convertToYAML(v)
		},
		"toJSON": func(v any) string {
			// Simple JSON serialization
			return convertToJSON(v)
		},
	}
}

// Helper functions for template functions

// generateSimpleHash creates a simple hash for campaign names
func generateSimpleHash(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	hash := hex.EncodeToString(hasher.Sum(nil))
	// Return first 8 characters for readability
	return hash[:8]
}

// convertToYAML converts basic types to YAML-like string representation
func convertToYAML(v any) string {
	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	case string:
		// Simple string escaping - for full YAML, we'd need a proper library
		if strings.Contains(val, "\n") || strings.Contains(val, ":") || strings.Contains(val, "#") {
			return fmt.Sprintf("%q", val)
		}
		return val
	case []any:
		var items []string
		for _, item := range val {
			items = append(items, "- "+convertToYAML(item))
		}
		return strings.Join(items, "\n")
	case map[string]any:
		var items []string
		for key, value := range val {
			items = append(items, fmt.Sprintf("%s: %s", key, convertToYAML(value)))
		}
		return strings.Join(items, "\n")
	default:
		return fmt.Sprintf("%v", val)
	}
}

// convertToJSON converts values to JSON string representation
func convertToJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
