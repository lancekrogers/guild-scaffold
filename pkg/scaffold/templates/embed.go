// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package templates

import (
	"fmt"
	"embed"
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
