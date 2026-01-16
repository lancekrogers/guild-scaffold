package scaffold

import (
	"fmt"
	"strings"
)

// Error codes for scaffold operations (stdlib-compatible)
const (
	ErrCodeValidation        = "SCAFFOLD-2000"
	ErrCodeTemplateNotFound  = "SCAFFOLD-3001"
	ErrCodeTemplateRender    = "SCAFFOLD-1000"
	ErrCodeFileExists        = "SCAFFOLD-3002"
	ErrCodeFileWrite         = "SCAFFOLD-8000"
	ErrCodeFileRead          = "SCAFFOLD-8000"
	ErrCodeYAMLParse         = "SCAFFOLD-8001"
	ErrCodeSchemaValidation  = "SCAFFOLD-2000"
	ErrCodeTimeout           = "SCAFFOLD-1002"
	ErrCodePermission        = "SCAFFOLD-9001"
	ErrCodeInvalidPath       = "SCAFFOLD-2001"
	ErrCodeRecipeNotFound    = "SCAFFOLD-3001"
	ErrCodeVariableUndefined = "SCAFFOLD-2001"
	ErrCodeCancelled         = "SCAFFOLD-1003"
	ErrCodeInternal          = "SCAFFOLD-1000"
	ErrCodeIO                = "SCAFFOLD-8000"
	ErrCodeParsing           = "SCAFFOLD-8001"
	ErrCodeNotFound          = "SCAFFOLD-3001"
	ErrCodeAlreadyExists     = "SCAFFOLD-3002"
	ErrCodeNotImplemented    = "SCAFFOLD-1006"
	ErrCodeConnection        = "SCAFFOLD-3004"
	ErrCodeExternal          = "SCAFFOLD-7000"
	ErrCodeInvalidInput      = "SCAFFOLD-2001"
)

// ScaffoldError represents scaffold-specific errors with rich context
type ScaffoldError struct {
	Code    string
	Message string
	File    string
	Line    int
	Column  int
	Context map[string]any
	Cause   error
}

// Error implements the error interface
func (e *ScaffoldError) Error() string {
	if e.File != "" && e.Line > 0 {
		return fmt.Sprintf("%s: %s (at %s:%d:%d)", e.Code, e.Message, e.File, e.Line, e.Column)
	}
	if e.File != "" {
		return fmt.Sprintf("%s: %s (in %s)", e.Code, e.Message, e.File)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error chain support
func (e *ScaffoldError) Unwrap() error {
	return e.Cause
}

// NewScaffoldError creates a new scaffold error with context
func NewScaffoldError(code, message string) *ScaffoldError {
	return &ScaffoldError{
		Code:    code,
		Message: message,
		Context: make(map[string]any),
	}
}

// WithFile adds file context to the error
func (e *ScaffoldError) WithFile(filename string) *ScaffoldError {
	e.File = filename
	return e
}

// WithPosition adds line/column position to the error
func (e *ScaffoldError) WithPosition(line, column int) *ScaffoldError {
	e.Line = line
	e.Column = column
	return e
}

// WithContext adds arbitrary context to the error
func (e *ScaffoldError) WithContext(key string, value any) *ScaffoldError {
	e.Context[key] = value
	return e
}

// WithCause wraps an underlying error
func (e *ScaffoldError) WithCause(err error) *ScaffoldError {
	e.Cause = err
	return e
}

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
	Code    string `json:"code"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

// Error implements the error interface
func (ve ValidationError) Error() string {
	location := ""
	if ve.Line > 0 {
		if ve.Column > 0 {
			location = fmt.Sprintf(" (line %d, col %d)", ve.Line, ve.Column)
		} else {
			location = fmt.Sprintf(" (line %d)", ve.Line)
		}
	}

	if ve.Value != nil {
		return fmt.Sprintf("%s: %s (value: %v)%s", ve.Field, ve.Message, ve.Value, location)
	}
	return fmt.Sprintf("%s: %s%s", ve.Field, ve.Message, location)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed with no specific errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("validation failed with %d errors:\n", len(ve)))

	for i, err := range ve {
		buf.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}

	return buf.String()
}

// HasErrorCode checks if any validation error has the given code
func (ve ValidationErrors) HasErrorCode(code string) bool {
	for _, err := range ve {
		if err.Code == code {
			return true
		}
	}
	return false
}

// GetErrorsByCode returns all validation errors with the given code
func (ve ValidationErrors) GetErrorsByCode(code string) ValidationErrors {
	var filtered ValidationErrors
	for _, err := range ve {
		if err.Code == code {
			filtered = append(filtered, err)
		}
	}
	return filtered
}

// EnhancedYAMLError provides enhanced YAML parsing errors with context
type EnhancedYAMLError struct {
	OriginalError    error
	EnhancedMessages []string
	LineContext      []string
	File             string
}

// Error implements the error interface
func (e *EnhancedYAMLError) Error() string {
	var buf strings.Builder

	if e.File != "" {
		buf.WriteString(fmt.Sprintf("YAML parsing error in %s:\n", e.File))
	} else {
		buf.WriteString("YAML parsing error:\n")
	}

	for _, msg := range e.EnhancedMessages {
		buf.WriteString("  ")
		buf.WriteString(msg)
		buf.WriteString("\n")
	}

	if len(e.LineContext) > 0 {
		buf.WriteString("\nContext:\n")
		for _, line := range e.LineContext {
			buf.WriteString("  ")
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

// Unwrap returns the original error
func (e *EnhancedYAMLError) Unwrap() error {
	return e.OriginalError
}

// SchemaValidationError represents JSON schema validation failures
type SchemaValidationError struct {
	Errors []string
	Count  int
	Schema string
}

// Error implements the error interface
func (e *SchemaValidationError) Error() string {
	if e.Count == 1 {
		return fmt.Sprintf("schema validation error: %s", e.Errors[0])
	}

	return fmt.Sprintf("schema validation failed with %d errors:\n%s",
		e.Count, strings.Join(e.Errors, "\n"))
}

// Helper functions to create common scaffold errors using stdlib

// ErrRecipeNotFound creates an error for missing recipe files
func ErrRecipeNotFound(path string) error {
	return fmt.Errorf("[%s] recipe file not found: path=%s", ErrCodeRecipeNotFound, path)
}

// ErrTemplateNotFound creates an error for missing template files
func ErrTemplateNotFound(template, templatesDir string) error {
	return fmt.Errorf("[%s] template file not found: template=%s, templatesDir=%s", ErrCodeTemplateNotFound, template, templatesDir)
}

// ErrFileExists creates an error for existing files that cannot be overwritten
func ErrFileExists(path string) error {
	return fmt.Errorf("[%s] file already exists and overwrite is disabled: path=%s", ErrCodeFileExists, path)
}

// ErrTemplateRender creates an error for template rendering failures
func ErrTemplateRender(template string, err error) error {
	return fmt.Errorf("[%s] template rendering failed (template=%s): %w", ErrCodeTemplateRender, template, err)
}

// ErrFileWrite creates an error for file writing failures
func ErrFileWrite(path string, err error) error {
	return fmt.Errorf("[%s] failed to write file (path=%s): %w", ErrCodeFileWrite, path, err)
}

// ErrFileRead creates an error for file reading failures
func ErrFileRead(path string, err error) error {
	return fmt.Errorf("[%s] failed to read file (path=%s): %w", ErrCodeFileRead, path, err)
}

// ErrYAMLParse creates an error for YAML parsing failures
func ErrYAMLParse(path string, err error) error {
	return fmt.Errorf("[%s] failed to parse YAML (path=%s): %w", ErrCodeYAMLParse, path, err)
}

// ErrValidation creates an error for validation failures
func ErrValidation(message string) error {
	return fmt.Errorf("[%s] %s", ErrCodeValidation, message)
}

// ErrTimeout creates an error for timeout scenarios
func ErrTimeout(operation string, duration string) error {
	return fmt.Errorf("[%s] operation timed out: operation=%s, timeout=%s", ErrCodeTimeout, operation, duration)
}

// ErrPermission creates an error for permission issues
func ErrPermission(path string, err error) error {
	return fmt.Errorf("[%s] permission denied (path=%s): %w", ErrCodePermission, path, err)
}

// ErrInvalidPath creates an error for invalid file paths
func ErrInvalidPath(path string, reason string) error {
	return fmt.Errorf("[%s] invalid file path: path=%s, reason=%s", ErrCodeInvalidPath, path, reason)
}
