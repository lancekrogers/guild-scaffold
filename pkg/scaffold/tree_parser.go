package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TreeFormat represents the tree-like YAML structure
// It uses a recursive structure where:
// - Directories are represented as nested maps
// - Files are stored in a special "_files" key
// - Empty directories use "_empty: true"
type TreeFormat map[string]interface{}

// TreeParser converts tree-like YAML format to standard Recipe format
type TreeParser struct {
	defaultTemplatesDir string
	defaultVersion      string
}

// NewTreeParser creates a new tree format parser
func NewTreeParser() *TreeParser {
	return &TreeParser{
		defaultTemplatesDir: "templates",
		defaultVersion:      "1.0.0",
	}
}

// ParseTreeFormat parses tree-like YAML and converts it to Recipe
func (tp *TreeParser) ParseTreeFormat(data []byte) (*Recipe, error) {
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, ErrYAMLParse("", err)
	}

	recipe := &Recipe{
		ScaffoldVersion: tp.defaultVersion,
		TemplatesDir:    tp.defaultTemplatesDir,
		Files:           []FileEntry{},
		Vars:            make(map[string]any),
		MirrorStructure: false,
	}

	// Extract metadata if present
	if version, ok := rawData["_scaffold_version"].(string); ok {
		recipe.ScaffoldVersion = version
		delete(rawData, "_scaffold_version")
	}

	if templatesDir, ok := rawData["_templates_dir"].(string); ok {
		recipe.TemplatesDir = templatesDir
		delete(rawData, "_templates_dir")
	}

	if mirrorStructure, ok := rawData["_mirror_structure"].(bool); ok {
		recipe.MirrorStructure = mirrorStructure
		delete(rawData, "_mirror_structure")
	}

	if scanTemplates, ok := rawData["_scan_templates"].(bool); ok {
		recipe.ScanTemplates = scanTemplates
		delete(rawData, "_scan_templates")
	}

	if vars, ok := rawData["_vars"].(map[string]interface{}); ok {
		recipe.Vars = vars
		delete(rawData, "_vars")
	}

	// If scan_templates is enabled, skip tree processing - files will be discovered by scanning
	if recipe.ScanTemplates {
		return recipe, nil
	}

	// Process the tree structure
	for rootName, rootContent := range rawData {
		// Skip metadata fields
		if strings.HasPrefix(rootName, "_") {
			continue
		}

		// Remove trailing slash from root directory name
		rootPath := strings.TrimSuffix(rootName, "/")

		if err := tp.processNode(rootPath, rootContent, recipe); err != nil {
			return nil, fmt.Errorf("failed to process tree structure (path=%v): %w", rootPath, err)
		}
	}

	return recipe, nil
}

// generateMirroredTemplatePath generates a template path that mirrors the output path
func (tp *TreeParser) generateMirroredTemplatePath(outputPath string) string {
	return outputPath + ".tmpl"
}

// resolveTemplatePath resolves the final template path based on mirror mode
func (tp *TreeParser) resolveTemplatePath(recipe *Recipe, outputPath string, providedTemplate string) string {
	// If template is explicitly provided and not empty/nil marker, use it
	if providedTemplate != "" && providedTemplate != "~" {
		return providedTemplate
	}

	// If template is empty/nil marker (~), return it as-is
	if providedTemplate == "~" {
		return "~"
	}

	// If mirror structure is enabled and template is empty/nil, generate mirrored path
	if recipe.MirrorStructure {
		return tp.generateMirroredTemplatePath(outputPath)
	}

	// Default: return provided template (may be empty)
	return providedTemplate
}

// processNode recursively processes tree nodes
func (tp *TreeParser) processNode(currentPath string, node interface{}, recipe *Recipe) error {
	switch v := node.(type) {
	case string:
		// Check for symlink syntax: "@target"
		if strings.HasPrefix(v, "@") {
			// This is a symlink
			target := strings.TrimPrefix(v, "@")
			recipe.Files = append(recipe.Files, FileEntry{
				Path:      currentPath,
				SymlinkTo: target,
				With:      make(map[string]any),
			})
		} else {
			// Direct file mapping: "file.txt: template.tmpl"
			templatePath := tp.resolveTemplatePath(recipe, currentPath, v)
			recipe.Files = append(recipe.Files, FileEntry{
				Path:     currentPath,
				Template: templatePath,
				With:     make(map[string]any),
			})
		}
		return nil

	case map[string]interface{}:
		// Check for special markers first
		if empty, ok := v["_empty"].(bool); ok && empty {
			// Empty directory - create a .gitkeep file
			recipe.Files = append(recipe.Files, FileEntry{
				Path:     filepath.Join(currentPath, ".gitkeep"),
				Template: "~", // Special marker for empty file
				With:     make(map[string]any),
			})
			return nil
		}

		// Process files in this directory
		if files, ok := v["_files"].(map[string]interface{}); ok {
			for fileName, template := range files {
				filePath := filepath.Join(currentPath, fileName)

				switch tmpl := template.(type) {
				case string:
					// Check for symlink syntax: "@target"
					if strings.HasPrefix(tmpl, "@") {
						// This is a symlink
						target := strings.TrimPrefix(tmpl, "@")
						recipe.Files = append(recipe.Files, FileEntry{
							Path:      filePath,
							SymlinkTo: target,
							With:      make(map[string]any),
						})
					} else {
						templatePath := tp.resolveTemplatePath(recipe, filePath, tmpl)
						recipe.Files = append(recipe.Files, FileEntry{
							Path:     filePath,
							Template: templatePath,
							With:     make(map[string]any),
						})
					}
				case nil:
					// nil value means use mirror structure if enabled
					templatePath := tp.resolveTemplatePath(recipe, filePath, "")
					recipe.Files = append(recipe.Files, FileEntry{
						Path:     filePath,
						Template: templatePath,
						With:     make(map[string]any),
					})
				case map[string]interface{}:
					// File with additional properties
					entry := FileEntry{
						Path: filePath,
						With: make(map[string]any),
					}

					// Extract template if provided
					providedTemplate := ""
					if tmplStr, ok := tmpl["template"].(string); ok {
						providedTemplate = tmplStr
					}

					// Resolve template path with mirror logic
					entry.Template = tp.resolveTemplatePath(recipe, filePath, providedTemplate)

					if with, ok := tmpl["with"].(map[string]interface{}); ok {
						entry.With = with
					}

					recipe.Files = append(recipe.Files, entry)
				default:
					return fmt.Errorf("invalid template type for file %s", fileName)
				}
			}
		}

		// Process subdirectories
		for name, content := range v {
			// Skip special keys
			if strings.HasPrefix(name, "_") {
				continue
			}

			// Remove trailing slash from directory names
			dirName := strings.TrimSuffix(name, "/")
			subPath := filepath.Join(currentPath, dirName)

			if err := tp.processNode(subPath, content, recipe); err != nil {
				return err
			}
		}

		return nil

	case map[interface{}]interface{}:
		// YAML sometimes parses maps with interface{} keys
		// Convert to string keys and reprocess
		stringMap := make(map[string]interface{})
		for k, v := range v {
			if keyStr, ok := k.(string); ok {
				stringMap[keyStr] = v
			} else {
				return fmt.Errorf("non-string key in map: %v", k)
			}
		}
		return tp.processNode(currentPath, stringMap, recipe)

	default:
		return fmt.Errorf("unexpected node type: %T", v)
	}
}

// ConvertRecipeToTree converts a standard Recipe to tree format
func (tp *TreeParser) ConvertRecipeToTree(recipe *Recipe) (TreeFormat, error) {
	tree := make(TreeFormat)

	// Add metadata
	if recipe.ScaffoldVersion != "" {
		tree["_scaffold_version"] = recipe.ScaffoldVersion
	}

	if recipe.TemplatesDir != "" {
		tree["_templates_dir"] = recipe.TemplatesDir
	}

	if recipe.MirrorStructure {
		tree["_mirror_structure"] = recipe.MirrorStructure
	}

	if len(recipe.Vars) > 0 {
		tree["_vars"] = recipe.Vars
	}

	// Build tree structure from files
	for _, file := range recipe.Files {
		tp.addFileToTree(tree, file)
	}

	return tree, nil
}

// addFileToTree adds a file entry to the tree structure
func (tp *TreeParser) addFileToTree(tree TreeFormat, file FileEntry) {
	parts := strings.Split(file.Path, string(filepath.Separator))

	// Navigate/create the tree structure
	current := tree
	for i, part := range parts {
		if i == len(parts)-1 {
			// This is the file
			if current["_files"] == nil {
				current["_files"] = make(map[string]interface{})
			}

			files := current["_files"].(map[string]interface{})

			// Handle symlinks
			if file.IsSymlink() {
				files[part] = "@" + file.SymlinkTo
			} else if len(file.With) > 0 {
				// If file has custom properties, store as object
				files[part] = map[string]interface{}{
					"template": file.Template,
					"with":     file.With,
				}
			} else {
				// Simple file mapping
				files[part] = file.Template
			}
		} else {
			// This is a directory
			dirName := part + "/"
			if current[dirName] == nil {
				current[dirName] = make(map[string]interface{})
			}

			// Move to next level
			if next, ok := current[dirName].(map[string]interface{}); ok {
				current = next
			}
		}
	}
}
