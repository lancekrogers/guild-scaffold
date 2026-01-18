package scaffold

import (
	"context"
	"testing"
)

func TestSymlinkSupport(t *testing.T) {
	t.Run("FileEntry IsSymlink", func(t *testing.T) {
		// Test symlink entry
		symlinkEntry := FileEntry{
			Path:      "CLAUDE.md",
			SymlinkTo: "AGENTS.md",
		}
		if !symlinkEntry.IsSymlink() {
			t.Error("FileEntry with SymlinkTo should return true for IsSymlink()")
		}

		// Test regular entry
		regularEntry := FileEntry{
			Path:     "file.txt",
			Template: "template.tmpl",
		}
		if regularEntry.IsSymlink() {
			t.Error("FileEntry without SymlinkTo should return false for IsSymlink()")
		}
	})

	t.Run("MemoryFileSystem Symlink operations", func(t *testing.T) {
		fs := NewMemoryFileSystem()

		// Create a target file first
		err := fs.WriteFile("target.txt", []byte("target content"), 0644)
		if err != nil {
			t.Fatalf("Failed to write target file: %v", err)
		}

		// Create symlink
		err = fs.Symlink("target.txt", "link.txt")
		if err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}

		// Check if symlink exists
		if !fs.Exists("link.txt") {
			t.Error("Symlink should exist")
		}

		// Check if it's recognized as a symlink
		if !fs.IsSymlink("link.txt") {
			t.Error("Should be recognized as symlink")
		}

		// Read symlink target
		target, err := fs.ReadLink("link.txt")
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}
		if target != "target.txt" {
			t.Errorf("Expected target 'target.txt', got '%s'", target)
		}

		// Test duplicate symlink
		err = fs.Symlink("target.txt", "link.txt")
		if err == nil {
			t.Error("Should not allow duplicate symlink")
		}
	})

	t.Run("TreeParser @ prefix detection", func(t *testing.T) {
		yamlData := []byte(`
project/:
  _files:
    AGENTS.md: agents.md.tmpl
    CLAUDE.md: "@AGENTS.md"
`)
		parser := NewTreeParser()
		recipe, err := parser.ParseTreeFormat(yamlData)
		if err != nil {
			t.Fatalf("Failed to parse tree format: %v", err)
		}

		if len(recipe.Files) != 2 {
			t.Fatalf("Expected 2 files, got %d", len(recipe.Files))
		}

		// Find the symlink entry
		var symlinkEntry *FileEntry
		for i := range recipe.Files {
			if recipe.Files[i].Path == "project/CLAUDE.md" {
				symlinkEntry = &recipe.Files[i]
				break
			}
		}

		if symlinkEntry == nil {
			t.Fatal("project/CLAUDE.md entry not found")
		}

		if !symlinkEntry.IsSymlink() {
			t.Error("CLAUDE.md should be a symlink")
		}

		if symlinkEntry.SymlinkTo != "AGENTS.md" {
			t.Errorf("Expected symlink target 'AGENTS.md', got '%s'", symlinkEntry.SymlinkTo)
		}
	})

	t.Run("Renderer symlink creation", func(t *testing.T) {
		ctx := context.Background()
		fs := NewMemoryFileSystem()

		// Create a simple recipe with a symlink
		recipe := &Recipe{
			ScaffoldVersion: "1.0.0",
			TemplatesDir:    "templates",
			Files: []FileEntry{
				{
					Path:      "target.txt",
					Template:  "~", // Empty file
				},
				{
					Path:      "link.txt",
					SymlinkTo: "target.txt",
				},
			},
		}

		// Create renderer
		renderer := NewRenderer(nil, fs)

		// Render the recipe
		options := Options{
			Dest:      "output",
			Dry:       false,
			Overwrite: false,
		}

		stats, err := renderer.RenderRecipe(ctx, recipe, options)
		if err != nil {
			t.Fatalf("Failed to render recipe: %v", err)
		}

		if stats.FilesGenerated != 2 {
			t.Errorf("Expected 2 files generated, got %d", stats.FilesGenerated)
		}

		// Verify symlink was created
		if !fs.IsSymlink("link.txt") {
			t.Error("link.txt should be a symlink")
		}

		target, err := fs.ReadLink("link.txt")
		if err != nil {
			t.Fatalf("Failed to read symlink: %v", err)
		}
		if target != "target.txt" {
			t.Errorf("Expected symlink target 'target.txt', got '%s'", target)
		}
	})

	t.Run("Validator symlink validation", func(t *testing.T) {
		ctx := context.Background()
		validator := newSemanticValidator()

		// Test valid symlink
		validRecipe := &Recipe{
			ScaffoldVersion: "1.0.0",
			TemplatesDir:    "templates",
			Files: []FileEntry{
				{
					Path:      "link.txt",
					SymlinkTo: "target.txt",
				},
			},
		}

		errors := validator.Validate(ctx, validRecipe)
		if len(errors) > 0 {
			t.Errorf("Valid symlink should not have validation errors: %v", errors)
		}

		// Test invalid: empty symlink target (with explicit empty string marker)
		// Note: SymlinkTo: "" means it's not a symlink, so we need to set it to " " (whitespace)
		invalidRecipe1 := &Recipe{
			ScaffoldVersion: "1.0.0",
			TemplatesDir:    "templates",
			Files: []FileEntry{
				{
					Path:      "link.txt",
					SymlinkTo: " ", // whitespace-only is invalid
				},
			},
		}

		errors = validator.Validate(ctx, invalidRecipe1)
		hasEmptyTargetError := false
		for _, err := range errors {
			if err.Message == "symlink target cannot be empty" {
				hasEmptyTargetError = true
				break
			}
		}
		if !hasEmptyTargetError {
			t.Error("Should have error for empty symlink target")
		}

		// Test invalid: both template and symlink_to set
		invalidRecipe2 := &Recipe{
			ScaffoldVersion: "1.0.0",
			TemplatesDir:    "templates",
			Files: []FileEntry{
				{
					Path:      "link.txt",
					Template:  "template.tmpl",
					SymlinkTo: "target.txt",
				},
			},
		}

		errors = validator.Validate(ctx, invalidRecipe2)
		hasBothSetError := false
		for _, err := range errors {
			if err.Message == "cannot have both template and symlink_to set" {
				hasBothSetError = true
				break
			}
		}
		if !hasBothSetError {
			t.Error("Should have error for both template and symlink_to set")
		}
	})

	t.Run("ConvertRecipeToTree with symlinks", func(t *testing.T) {
		parser := NewTreeParser()

		recipe := &Recipe{
			ScaffoldVersion: "1.0.0",
			TemplatesDir:    "templates",
			Files: []FileEntry{
				{
					Path:     "AGENTS.md",
					Template: "agents.md.tmpl",
				},
				{
					Path:      "CLAUDE.md",
					SymlinkTo: "AGENTS.md",
				},
			},
		}

		tree, err := parser.ConvertRecipeToTree(recipe)
		if err != nil {
			t.Fatalf("Failed to convert recipe to tree: %v", err)
		}

		files, ok := tree["_files"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected _files in tree")
		}

		// Check AGENTS.md
		if files["AGENTS.md"] != "agents.md.tmpl" {
			t.Errorf("Expected AGENTS.md to have template 'agents.md.tmpl', got '%v'", files["AGENTS.md"])
		}

		// Check CLAUDE.md (should have @ prefix)
		if files["CLAUDE.md"] != "@AGENTS.md" {
			t.Errorf("Expected CLAUDE.md to be '@AGENTS.md', got '%v'", files["CLAUDE.md"])
		}
	})
}
