package scaffold

import (
	"fmt"
	"strings"

	"github.com/guild-framework/guild-core/pkg/gerror"
)

// Error codes for scaffold operations using gerror framework
const (
	ErrCodeValidation        = gerror.ErrCodeValidation
	ErrCodeTemplateNotFound  = gerror.ErrCodeNotFound
	ErrCodeTemplateRender    = gerror.ErrCodeInternal
	ErrCodeFileExists        = gerror.ErrCodeAlreadyExists
	ErrCodeFileWrite         = gerror.ErrCodeIO
	ErrCodeFileRead          = gerror.ErrCodeIO
	ErrCodeYAMLParse         = gerror.ErrCodeParsing
	ErrCodeSchemaValidation  = gerror.ErrCodeValidation
	ErrCodeTimeout           = gerror.ErrCodeTimeout
	ErrCodePermission        = gerror.ErrCodePermissionDenied
	ErrCodeInvalidPath       = gerror.ErrCodeInvalidInput
	ErrCodeRecipeNotFound    = gerror.ErrCodeNotFound
	ErrCodeVariableUndefined = gerror.ErrCodeInvalidInput
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
	Field   string               `json:"field"`
	Message string               `json:"message"`
	Value   any                  `json:"value,omitempty"`
	Code    gerror.ErrorCode     `json:"code"`
	Line    int                  `json:"line,omitempty"`
	Column  int                  `json:"column,omitempty"`
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
func (ve ValidationErrors) HasErrorCode(code gerror.ErrorCode) bool {
	for _, err := range ve {
		if err.Code == code {
			return true
		}
	}
	return false
}

// GetErrorsByCode returns all validation errors with the given code
func (ve ValidationErrors) GetErrorsByCode(code gerror.ErrorCode) ValidationErrors {
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

// Helper functions to create common scaffold errors using gerror

// ErrRecipeNotFound creates an error for missing recipe files
func ErrRecipeNotFound(path string) error {
	return gerror.New(ErrCodeRecipeNotFound, "recipe file not found", nil).
		WithDetails("path", path)
}

// ErrTemplateNotFound creates an error for missing template files
func ErrTemplateNotFound(template, templatesDir string) error {
	return gerror.New(ErrCodeTemplateNotFound, "template file not found", nil).
		WithDetails("template", template).
		WithDetails("templatesDir", templatesDir)
}

// ErrFileExists creates an error for existing files that cannot be overwritten
func ErrFileExists(path string) error {
	return gerror.New(ErrCodeFileExists, "file already exists and overwrite is disabled", nil).
		WithDetails("path", path)
}

// ErrTemplateRender creates an error for template rendering failures
func ErrTemplateRender(template string, err error) error {
	return gerror.Wrap(err, ErrCodeTemplateRender, "template rendering failed").
		WithDetails("template", template)
}

// ErrFileWrite creates an error for file writing failures
func ErrFileWrite(path string, err error) error {
	return gerror.Wrap(err, ErrCodeFileWrite, "failed to write file").
		WithDetails("path", path)
}

// ErrFileRead creates an error for file reading failures
func ErrFileRead(path string, err error) error {
	return gerror.Wrap(err, ErrCodeFileRead, "failed to read file").
		WithDetails("path", path)
}

// ErrYAMLParse creates an error for YAML parsing failures
func ErrYAMLParse(path string, err error) error {
	return gerror.Wrap(err, ErrCodeYAMLParse, "failed to parse YAML").
		WithDetails("path", path)
}

// ErrValidation creates an error for validation failures
func ErrValidation(message string) error {
	return gerror.New(ErrCodeValidation, message, nil)
}

// ErrTimeout creates an error for timeout scenarios
func ErrTimeout(operation string, duration string) error {
	return gerror.New(ErrCodeTimeout, "operation timed out", nil).
		WithDetails("operation", operation).
		WithDetails("timeout", duration)
}

// ErrPermission creates an error for permission issues
func ErrPermission(path string, err error) error {
	return gerror.Wrap(err, ErrCodePermission, "permission denied").
		WithDetails("path", path)
}

// ErrInvalidPath creates an error for invalid file paths
func ErrInvalidPath(path string, reason string) error {
	return gerror.New(ErrCodeInvalidPath, "invalid file path", nil).
		WithDetails("path", path).
		WithDetails("reason", reason)
}