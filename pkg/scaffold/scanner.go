package scaffold

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// ScanTemplatesDir scans a templates directory and generates FileEntry list.
// It walks the directory structure and applies the following rules:
//   - Files ending in .tmpl: Create a template entry (strip .tmpl suffix for output path)
//   - .gitkeep files: Create an empty file entry (marks directory for creation)
//   - Other files: Copied as static files (no template processing)
func ScanTemplatesDir(fsys fs.FS, templatesDir string) ([]FileEntry, error) {
	var entries []FileEntry

	err := fs.WalkDir(fsys, templatesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path from templates directory
		relPath, err := filepath.Rel(templatesDir, path)
		if err != nil {
			return err
		}

		// Skip the root templates directory itself
		if relPath == "." {
			return nil
		}

		// Skip directories - they're implicitly created when files are written
		if d.IsDir() {
			return nil
		}

		name := d.Name()

		// Handle .gitkeep files - mark directory for creation with empty file
		if name == ".gitkeep" {
			entries = append(entries, FileEntry{
				Path:     relPath,
				Template: "~", // Special marker for empty file
				With:     make(map[string]any),
			})
			return nil
		}

		// Handle template files - strip .tmpl suffix for output path
		if strings.HasSuffix(name, ".tmpl") {
			outputPath := strings.TrimSuffix(relPath, ".tmpl")
			entries = append(entries, FileEntry{
				Path:     outputPath,
				Template: relPath, // Keep relative path within templates dir
				With:     make(map[string]any),
			})
			return nil
		}

		// Handle static files - copy as-is (use special marker for static copy)
		entries = append(entries, FileEntry{
			Path:     relPath,
			Template: "=" + relPath, // "=" prefix indicates static copy
			With:     make(map[string]any),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return entries, nil
}
