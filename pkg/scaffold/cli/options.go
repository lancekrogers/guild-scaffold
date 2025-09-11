// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package cli

import (
	"fmt"
)

// InitOptions represents command-line options for the init command
type InitOptions struct {
	ProjectName     string
	TemplateName    string
	OutputDirectory string
	Variables       map[string]interface{}
	DryRun          bool
	Force           bool
	ConfigFile      string
	Provider        string
	Model           string
	Interactive     bool
	Verbose         bool
}

// ListOptions represents command-line options for the list command
type ListOptions struct {
	Verbose bool
	Format  string
}

// ValidateOptions represents command-line options for the validate command
type ValidateOptions struct {
	ScaffoldPath string
	Template     string
	Verbose      bool
	Format       string
}

// Validate checks if the InitOptions are valid
func (opts *InitOptions) Validate() error {
	if opts.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	
	if opts.OutputDirectory == "" {
		return fmt.Errorf("output directory cannot be empty")
	}
	
	return nil
}

// Validate checks if the ListOptions are valid  
func (opts *ListOptions) Validate() error {
	validFormats := map[string]bool{"table": true, "json": true, "yaml": true}
	if !validFormats[opts.Format] {
		return fmt.Errorf("invalid format %q, must be one of: table, json, yaml", opts.Format)
	}
	
	return nil
}

// Validate checks if the ValidateOptions are valid
func (opts *ValidateOptions) Validate() error {
	if opts.ScaffoldPath == "" && opts.Template == "" {
		return fmt.Errorf("must specify either scaffold path or template name")
	}
	
	validFormats := map[string]bool{"text": true, "json": true, "yaml": true}
	if !validFormats[opts.Format] {
		return fmt.Errorf("invalid format %q, must be one of: text, json, yaml", opts.Format)
	}
	
	return nil
}

// GetVariableValue safely retrieves a variable value with type assertion
func (opts *InitOptions) GetVariableValue(key string, defaultValue interface{}) interface{} {
	if value, exists := opts.Variables[key]; exists {
		return value
	}
	return defaultValue
}

// GetStringVariable retrieves a string variable with default
func (opts *InitOptions) GetStringVariable(key, defaultValue string) string {
	if value, exists := opts.Variables[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetBoolVariable retrieves a boolean variable with default  
func (opts *InitOptions) GetBoolVariable(key string, defaultValue bool) bool {
	if value, exists := opts.Variables[key]; exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// GetIntVariable retrieves an integer variable with default
func (opts *InitOptions) GetIntVariable(key string, defaultValue int) int {
	if value, exists := opts.Variables[key]; exists {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return defaultValue
}