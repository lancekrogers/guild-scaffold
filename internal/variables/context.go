package variables

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Context represents the execution context for variable handlers
type Context struct {
	// Project information
	ProjectName string
	ProjectPath string

	// User information
	Author   string
	Email    string
	Username string

	// System information
	OS       string
	Arch     string
	Hostname string

	// Time information
	Timestamp time.Time
	Year      int

	// Custom data
	Custom map[string]any
}

// NewContext creates a new variable context with system defaults
func NewContext() *Context {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}

	now := time.Now()

	return &Context{
		ProjectName: filepath.Base(mustGetwd()),
		ProjectPath: mustGetwd(),
		Author:      getGitUser(),
		Email:       getGitEmail(),
		Username:    username,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Hostname:    hostname,
		Timestamp:   now,
		Year:        now.Year(),
		Custom:      make(map[string]any),
	}
}

// WithProjectName sets the project name
func (c *Context) WithProjectName(name string) *Context {
	c.ProjectName = name
	return c
}

// WithProjectPath sets the project path
func (c *Context) WithProjectPath(path string) *Context {
	c.ProjectPath = path
	return c
}

// WithAuthor sets the author name
func (c *Context) WithAuthor(author string) *Context {
	c.Author = author
	return c
}

// WithEmail sets the email
func (c *Context) WithEmail(email string) *Context {
	c.Email = email
	return c
}

// WithCustom adds custom data to the context
func (c *Context) WithCustom(key string, value any) *Context {
	c.Custom[key] = value
	return c
}

// ToMap converts the context to a map for use with handlers
func (c *Context) ToMap() map[string]any {
	m := map[string]any{
		"project_name": c.ProjectName,
		"project_path": c.ProjectPath,
		"author":       c.Author,
		"email":        c.Email,
		"username":     c.Username,
		"os":           c.OS,
		"arch":         c.Arch,
		"hostname":     c.Hostname,
		"timestamp":    c.Timestamp,
		"year":         c.Year,
	}

	// Add custom data
	for k, v := range c.Custom {
		m[k] = v
	}

	return m
}

// mustGetwd gets the current working directory or returns "."
func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// getGitUser attempts to get the git user name
func getGitUser() string {
	// Try git config first
	if gitUser := os.Getenv("GIT_AUTHOR_NAME"); gitUser != "" {
		return gitUser
	}

	// Try system user
	if user := os.Getenv("USER"); user != "" {
		return user
	}

	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}

	return "Author"
}

// getGitEmail attempts to get the git email
func getGitEmail() string {
	if gitEmail := os.Getenv("GIT_AUTHOR_EMAIL"); gitEmail != "" {
		return gitEmail
	}

	if email := os.Getenv("EMAIL"); email != "" {
		return email
	}

	return "email@example.com"
}
