package scaffold

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"gopkg.in/yaml.v3"
)

// Parser handles YAML specification parsing and validation
type Parser interface {
	// ParseRecipe loads and parses a scaffold recipe from filesystem
	ParseRecipe(ctx context.Context, fsys fs.FS, path string) (*Recipe, error)
	
	// ValidateRecipe validates a recipe for correctness
	ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError
	
	// ParseWithOptions allows custom parsing options
	ParseWithOptions(ctx context.Context, data []byte, opts ParseOptions) (*Recipe, error)
}

// yamlParser implements the Parser interface
type yamlParser struct {
	validator *semanticValidator
	options   ParseOptions
	cache     *ParserCache
}

// NewParser creates a new YAML parser with validation
func NewParser(options ParseOptions) (Parser, error) {
	validator := newSemanticValidator()
	
	cache := &ParserCache{
		cache:   make(map[string]*cacheEntry),
		maxSize: 100,
		ttl:     5 * time.Minute,
	}
	
	return &yamlParser{
		validator: validator,
		options:   options,
		cache:     cache,
	}, nil
}

// ParseRecipe loads and parses a scaffold recipe from filesystem
func (p *yamlParser) ParseRecipe(ctx context.Context, fsys fs.FS, path string) (*Recipe, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", getFilesystemID(fsys), path)
	if cached, found := p.cache.Get(cacheKey); found {
		return cached, nil
	}
	
	// Read file with size limit
	data, err := p.readWithLimit(fsys, path)
	if err != nil {
		return nil, ErrFileRead(path, err)
	}
	
	// Parse with timeout
	recipe, err := p.parseWithTimeout(ctx, data, path)
	if err != nil {
		return nil, err
	}
	
	// Cache successful parse
	p.cache.Set(cacheKey, recipe)
	
	return recipe, nil
}

// ValidateRecipe validates a recipe for correctness
func (p *yamlParser) ValidateRecipe(ctx context.Context, recipe *Recipe) []ValidationError {
	return p.validator.Validate(ctx, recipe)
}

// ParseWithOptions allows custom parsing options
func (p *yamlParser) ParseWithOptions(ctx context.Context, data []byte, opts ParseOptions) (*Recipe, error) {
	// Create temporary parser with custom options
	tempParser := &yamlParser{
		validator: p.validator,
		options:   opts,
		cache:     p.cache,
	}
	
	return tempParser.parseWithTimeout(ctx, data, "")
}

// readWithLimit reads a file with size constraints
func (p *yamlParser) readWithLimit(fsys fs.FS, path string) ([]byte, error) {
	// Check file size first
	info, err := fs.Stat(fsys, path)
	if err != nil {
		return nil, err
	}
	
	if info.Size() > p.options.MaxFileSize {
		return nil, gerror.New(ErrCodeValidation, "file too large", nil).
			WithDetails("size", info.Size()).
			WithDetails("maxSize", p.options.MaxFileSize).
			WithDetails("path", path)
	}
	
	return fs.ReadFile(fsys, path)
}

// parseWithTimeout parses YAML with timeout protection
func (p *yamlParser) parseWithTimeout(ctx context.Context, data []byte, filename string) (*Recipe, error) {
	// Create timeout context
	parseCtx, cancel := context.WithTimeout(ctx, p.options.MaxParseTime)
	defer cancel()
	
	resultChan := make(chan parseResult, 1)
	
	go func() {
		recipe, err := p.parseYAML(data, filename)
		resultChan <- parseResult{recipe: recipe, err: err}
	}()
	
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		
		// Validate if enabled
		if p.options.ValidateSemantics {
			if validationErrors := p.ValidateRecipe(parseCtx, result.recipe); len(validationErrors) > 0 {
				// Format first error message for visibility
				msg := "recipe validation failed"
				if len(validationErrors) > 0 {
					msg = validationErrors[0].Message
					if len(validationErrors) > 1 {
						msg += fmt.Sprintf(" (and %d more errors)", len(validationErrors)-1)
					}
				}
				return nil, gerror.New(ErrCodeValidation, msg, nil).
					WithDetails("errors", validationErrors)
			}
		}
		
		return result.recipe, nil
		
	case <-parseCtx.Done():
		return nil, ErrTimeout("YAML parsing", p.options.MaxParseTime.String())
	}
}

// parseResult holds the result of async parsing
type parseResult struct {
	recipe *Recipe
	err    error
}

// parseYAML performs the actual YAML parsing with error enhancement
func (p *yamlParser) parseYAML(data []byte, filename string) (*Recipe, error) {
	// First check if this is a tree-like format
	if isTreeFormat(data) {
		treeParser := NewTreeParser()
		recipe, err := treeParser.ParseTreeFormat(data)
		if err != nil {
			return nil, gerror.Wrap(err, ErrCodeYAMLParse, "failed to parse tree format").
				WithDetails("file", filename)
		}
		return recipe, nil
	}
	
	// Standard YAML parsing
	var recipe Recipe
	
	// Configure YAML decoder
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(p.options.StrictMode)
	
	// Parse with detailed error information
	if err := decoder.Decode(&recipe); err != nil {
		return nil, p.enhanceYAMLError(err, data, filename)
	}
	
	// Validate required fields
	if err := p.validateRequiredFields(&recipe); err != nil {
		return nil, err
	}
	
	// Apply defaults
	p.applyDefaults(&recipe)
	
	return &recipe, nil
}

// isTreeFormat checks if the YAML data is in tree-like format
func isTreeFormat(data []byte) bool {
	// Quick check for tree format indicators
	dataStr := string(data)
	
	// Tree format has directories with trailing slashes and _files markers
	hasTreeIndicators := strings.Contains(dataStr, "/:") && 
		(strings.Contains(dataStr, "_files:") || strings.Contains(dataStr, "_empty:"))
	
	// Standard format has a top-level files: array
	hasStandardFormat := regexp.MustCompile(`(?m)^files:\s*$`).MatchString(dataStr)
	
	return hasTreeIndicators && !hasStandardFormat
}

// enhanceYAMLError adds context and line information to YAML errors
func (p *yamlParser) enhanceYAMLError(err error, data []byte, filename string) error {
	// Extract line information from YAML error
	yamlErr, ok := err.(*yaml.TypeError)
	if !ok {
		return ErrYAMLParse(filename, err)
	}
	
	lines := strings.Split(string(data), "\n")
	
	var enhanced []string
	for _, errStr := range yamlErr.Errors {
		enhanced = append(enhanced, p.addLineContext(errStr, lines))
	}
	
	enhancedErr := &EnhancedYAMLError{
		OriginalError:    err,
		EnhancedMessages: enhanced,
		LineContext:      p.extractRelevantLines(yamlErr, lines),
		File:             filename,
	}
	
	return gerror.Wrap(enhancedErr, ErrCodeYAMLParse, "YAML parsing failed").
		WithDetails("file", filename)
}

// addLineContext adds line numbers and context to error messages
func (p *yamlParser) addLineContext(errStr string, lines []string) string {
	// Try to extract line number from error message
	lineRegex := regexp.MustCompile(`line (\d+)`)
	matches := lineRegex.FindStringSubmatch(errStr)
	
	if len(matches) >= 2 {
		return errStr // Already has line info
	}
	
	return errStr
}

// extractRelevantLines extracts lines around errors for context
func (p *yamlParser) extractRelevantLines(yamlErr *yaml.TypeError, lines []string) []string {
	var context []string
	
	// For now, return first few lines as context
	maxLines := 5
	for i, line := range lines {
		if i >= maxLines {
			break
		}
		context = append(context, fmt.Sprintf("%d: %s", i+1, line))
	}
	
	return context
}

// validateRequiredFields checks that all required fields are present
func (p *yamlParser) validateRequiredFields(recipe *Recipe) error {
	var errors []ValidationError
	
	if recipe.ScaffoldVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "scaffold_version",
			Message: "scaffold version is required",
			Code:    ErrCodeValidation,
		})
	}
	
	// TemplatesDir is optional - it can be empty or set to a specific directory
	// Removed the requirement for templates_dir
	
	if len(recipe.Files) == 0 {
		errors = append(errors, ValidationError{
			Field:   "files",
			Message: "at least one file entry is required",
			Code:    ErrCodeValidation,
		})
	}
	
	// Validate individual file entries
	for i, file := range recipe.Files {
		if file.Path == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].path", i),
				Message: "file path is required",
				Code:    ErrCodeValidation,
				Value:   file.Path,
			})
		}
		
		if file.Template == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].template", i),
				Message: "template name is required",
				Code:    ErrCodeValidation,
				Value:   file.Template,
			})
		}
		
		// Validate path safety
		if err := p.validatePathSafety(file.Path); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("files[%d].path", i),
				Message: err.Error(),
				Code:    ErrCodeInvalidPath,
				Value:   file.Path,
			})
		}
	}
	
	if len(errors) > 0 {
		return gerror.New(ErrCodeValidation, "validation failed", nil).
			WithDetails("errors", ValidationErrors(errors))
	}
	
	return nil
}

// validatePathSafety ensures file paths are safe and don't escape the destination
func (p *yamlParser) validatePathSafety(path string) error {
	// Check for absolute paths
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	
	// Check for path traversal attempts
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal attempts are not allowed")
	}
	
	// Check for reserved names on Windows
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	baseName := strings.ToUpper(filepath.Base(path))
	for _, res := range reserved {
		if baseName == res || strings.HasPrefix(baseName, res+".") {
			return fmt.Errorf("reserved filename: %s", path)
		}
	}
	
	return nil
}

// applyDefaults sets default values for optional fields
func (p *yamlParser) applyDefaults(recipe *Recipe) {
	// Initialize Vars if nil
	if recipe.Vars == nil {
		recipe.Vars = make(map[string]any)
	}
	
	// Initialize With maps for file entries
	for i := range recipe.Files {
		if recipe.Files[i].With == nil {
			recipe.Files[i].With = make(map[string]any)
		}
	}
}

// ParserCache provides caching for parsed recipes
type ParserCache struct {
	cache   map[string]*cacheEntry
	mu      sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// cacheEntry represents a cached parse result
type cacheEntry struct {
	recipe    *Recipe
	timestamp time.Time
	hits      int64
}

// Get retrieves a cached recipe if it exists and is still valid
func (pc *ParserCache) Get(key string) (*Recipe, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	entry, exists := pc.cache[key]
	if !exists {
		return nil, false
	}
	
	// Check TTL
	if time.Since(entry.timestamp) > pc.ttl {
		return nil, false
	}
	
	atomic.AddInt64(&entry.hits, 1)
	return entry.recipe, true
}

// Set stores a recipe in the cache
func (pc *ParserCache) Set(key string, recipe *Recipe) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Evict old entries if at capacity
	if len(pc.cache) >= pc.maxSize {
		pc.evictLRU()
	}
	
	pc.cache[key] = &cacheEntry{
		recipe:    recipe,
		timestamp: time.Now(),
		hits:      1,
	}
}

// evictLRU removes the least recently used entry
func (pc *ParserCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time = time.Now()
	
	for key, entry := range pc.cache {
		if entry.timestamp.Before(oldestTime) {
			oldestTime = entry.timestamp
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(pc.cache, oldestKey)
	}
}

// getFilesystemID generates a cache key for filesystem instances
// This is a simple implementation - in production you might want something more sophisticated
func getFilesystemID(fsys fs.FS) string {
	return fmt.Sprintf("%p", fsys)
}