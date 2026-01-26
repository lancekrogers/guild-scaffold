// Copyright (c) 2025 Lance Rogers
// SPDX-License-Identifier: MIT

package templates

import (
	"embed"
	"fmt"
	"io/fs"
)

// EmbeddedTemplates contains the embedded template files
//
//go:embed templates
var EmbeddedTemplates embed.FS

// GetEmbeddedTemplatesFS returns the embedded templates filesystem
func GetEmbeddedTemplatesFS() (fs.FS, error) {
	// Return the subdirectory that contains the actual templates
	subFS, err := fs.Sub(EmbeddedTemplates, "templates")
	if err != nil {
		return nil, fmt.Errorf("failed to get templates subdirectory: %w", err)
	}
	return subFS, nil
}

// GetTemplatesFS returns the full embedded filesystem including the templates directory
func GetTemplatesFS() fs.FS {
	return EmbeddedTemplates
}
