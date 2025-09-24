package variables

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// VariableHandler is a function that generates content for a special variable
// path is the destination path where the file will be created
// context contains any additional data needed for generation
type VariableHandler func(path string, context map[string]any) ([]byte, error)

// Handlers is the registry of all available variable handlers
// Variables in YAML are referenced as {VARIABLE_NAME}
var Handlers = map[string]VariableHandler{
	"DATABASE_FILE":   createDatabaseFile,
	"SOCKET_FILE":     createSocketFile,
	"UUID_FILE":       generateUUIDFile,
	"TIMESTAMP_FILE":  createTimestampFile,
	"ENV_FILE":        createEnvFile,
	"CONFIG_FILE":     createConfigFile,
	"SECRET_FILE":     generateSecretFile,
	"LICENSE_FILE":    createLicenseFile,
	"GITKEEP":         createGitkeepFile,
	"EMPTY":           createEmptyFile,
}

// createDatabaseFile creates an SQLite database with initial schema
func createDatabaseFile(path string, ctx map[string]any) ([]byte, error) {
	schema := `-- Generated database schema
-- Created: %s
-- Path: %s

CREATE TABLE IF NOT EXISTS metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT UNIQUE NOT NULL,
    value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    level TEXT NOT NULL,
    message TEXT NOT NULL,
    context TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_logs_created_at ON logs(created_at);
CREATE INDEX idx_logs_level ON logs(level);

-- Insert initial metadata
INSERT INTO metadata (key, value) VALUES 
    ('version', '1.0.0'),
    ('created', '%s'),
    ('path', '%s');
`
	
	now := time.Now().Format(time.RFC3339)
	content := fmt.Sprintf(schema, now, path, now, path)
	return []byte(content), nil
}

// createSocketFile creates a socket configuration file
func createSocketFile(path string, ctx map[string]any) ([]byte, error) {
	config := `# Socket Configuration
# Generated: %s
# Path: %s

# Socket type: unix, tcp, udp
type: unix

# Socket address
address: %s

# Permissions (for unix sockets)
permissions: 0600

# Buffer sizes
read_buffer: 4096
write_buffer: 4096

# Timeouts (in seconds)
connect_timeout: 10
read_timeout: 30
write_timeout: 30

# Max connections (for server sockets)
max_connections: 100

# Keep-alive settings
keep_alive: true
keep_alive_period: 30
`
	
	now := time.Now().Format(time.RFC3339)
	// For unix sockets, use the file path; for network, could be from context
	address := path
	if proto, ok := ctx["protocol"].(string); ok && proto != "unix" {
		if addr, ok := ctx["address"].(string); ok {
			address = addr
		} else {
			address = "localhost:8080"
		}
	}
	
	content := fmt.Sprintf(config, now, path, address)
	return []byte(content), nil
}

// generateUUIDFile creates a file containing a new UUID
func generateUUIDFile(path string, ctx map[string]any) ([]byte, error) {
	// Generate UUID v4
	uuid := make([]byte, 16)
	if _, err := rand.Read(uuid); err != nil {
		return nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
	
	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	
	// Format as string
	uuidStr := fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
	
	return []byte(uuidStr), nil
}

// createTimestampFile creates a file with current timestamp information
func createTimestampFile(path string, ctx map[string]any) ([]byte, error) {
	now := time.Now()
	
	content := fmt.Sprintf(`# Timestamp File
# Generated at: %s

unix: %d
iso8601: %s
rfc3339: %s
rfc822: %s
year: %d
month: %s
day: %d
hour: %d
minute: %d
second: %d
timezone: %s
`,
		now.Format(time.RFC3339),
		now.Unix(),
		now.Format("2006-01-02T15:04:05"),
		now.Format(time.RFC3339),
		now.Format(time.RFC822),
		now.Year(),
		now.Month().String(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Location().String(),
	)
	
	return []byte(content), nil
}

// createEnvFile creates an environment variable template file
func createEnvFile(path string, ctx map[string]any) ([]byte, error) {
	projectName := "myproject"
	if name, ok := ctx["project_name"].(string); ok {
		projectName = name
	}
	
	content := fmt.Sprintf(`# Environment Configuration
# Project: %s
# Generated: %s

# Application
APP_NAME=%s
APP_ENV=development
APP_DEBUG=true
APP_PORT=8080

# Database
DATABASE_URL=postgres://user:password@localhost:5432/%s_db
DATABASE_MAX_CONNECTIONS=25
DATABASE_SSL_MODE=disable

# Redis
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=

# API Keys
API_KEY=your-api-key-here
SECRET_KEY=%s

# External Services
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_REGION=us-east-1

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Feature Flags
FEATURE_NEW_UI=false
FEATURE_BETA_API=false
`,
		projectName,
		time.Now().Format(time.RFC3339),
		projectName,
		projectName,
		generateRandomSecret(32),
	)
	
	return []byte(content), nil
}

// createConfigFile creates a configuration file template
func createConfigFile(path string, ctx map[string]any) ([]byte, error) {
	ext := filepath.Ext(path)
	
	switch ext {
	case ".yaml", ".yml":
		return createYAMLConfig(path, ctx)
	case ".json":
		return createJSONConfig(path, ctx)
	case ".toml":
		return createTOMLConfig(path, ctx)
	default:
		return createYAMLConfig(path, ctx)
	}
}

// createYAMLConfig creates a YAML configuration file
func createYAMLConfig(path string, ctx map[string]any) ([]byte, error) {
	projectName := "myproject"
	if name, ok := ctx["project_name"].(string); ok {
		projectName = name
	}
	
	content := fmt.Sprintf(`# Configuration File
# Generated: %s

app:
  name: %s
  version: 1.0.0
  environment: development
  
server:
  host: localhost
  port: 8080
  timeout: 30s
  
database:
  driver: postgres
  host: localhost
  port: 5432
  name: %s_db
  user: dbuser
  sslmode: disable
  
logging:
  level: info
  format: json
  output: stdout
  
cache:
  driver: redis
  host: localhost
  port: 6379
  ttl: 3600
`,
		time.Now().Format(time.RFC3339),
		projectName,
		projectName,
	)
	
	return []byte(content), nil
}

// createJSONConfig creates a JSON configuration file
func createJSONConfig(path string, ctx map[string]any) ([]byte, error) {
	projectName := "myproject"
	if name, ok := ctx["project_name"].(string); ok {
		projectName = name
	}
	
	content := fmt.Sprintf(`{
  "_comment": "Generated: %s",
  "app": {
    "name": "%s",
    "version": "1.0.0",
    "environment": "development"
  },
  "server": {
    "host": "localhost",
    "port": 8080,
    "timeout": "30s"
  },
  "database": {
    "driver": "postgres",
    "host": "localhost",
    "port": 5432,
    "name": "%s_db",
    "user": "dbuser",
    "sslmode": "disable"
  },
  "logging": {
    "level": "info",
    "format": "json",
    "output": "stdout"
  }
}`,
		time.Now().Format(time.RFC3339),
		projectName,
		projectName,
	)
	
	return []byte(content), nil
}

// createTOMLConfig creates a TOML configuration file
func createTOMLConfig(path string, ctx map[string]any) ([]byte, error) {
	projectName := "myproject"
	if name, ok := ctx["project_name"].(string); ok {
		projectName = name
	}
	
	content := fmt.Sprintf(`# Configuration File
# Generated: %s

[app]
name = "%s"
version = "1.0.0"
environment = "development"

[server]
host = "localhost"
port = 8080
timeout = "30s"

[database]
driver = "postgres"
host = "localhost"
port = 5432
name = "%s_db"
user = "dbuser"
sslmode = "disable"

[logging]
level = "info"
format = "json"
output = "stdout"
`,
		time.Now().Format(time.RFC3339),
		projectName,
		projectName,
	)
	
	return []byte(content), nil
}

// generateSecretFile generates a file with random secret/key
func generateSecretFile(path string, ctx map[string]any) ([]byte, error) {
	length := 32
	if l, ok := ctx["length"].(int); ok && l > 0 {
		length = l
	}
	
	secret := generateRandomSecret(length)
	return []byte(secret), nil
}

// createLicenseFile creates a license file
func createLicenseFile(path string, ctx map[string]any) ([]byte, error) {
	licenseType := "MIT"
	if lt, ok := ctx["license"].(string); ok {
		licenseType = lt
	}
	
	author := "Your Name"
	if a, ok := ctx["author"].(string); ok {
		author = a
	}
	
	year := time.Now().Year()
	
	var content string
	switch licenseType {
	case "MIT":
		content = fmt.Sprintf(`MIT License

Copyright (c) %d %s

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`, year, author)
		
	case "Apache-2.0", "Apache":
		content = fmt.Sprintf(`Copyright %d %s

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`, year, author)
		
	default:
		content = fmt.Sprintf("Copyright (c) %d %s\nAll rights reserved.", year, author)
	}
	
	return []byte(content), nil
}

// createGitkeepFile creates an empty .gitkeep file
func createGitkeepFile(path string, ctx map[string]any) ([]byte, error) {
	return []byte(""), nil
}

// createEmptyFile creates an empty file
func createEmptyFile(path string, ctx map[string]any) ([]byte, error) {
	return []byte(""), nil
}

// generateRandomSecret generates a random secret string
func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to less secure but functional
		for i := range bytes {
			bytes[i] = byte(i + int(time.Now().UnixNano()))
		}
	}
	return hex.EncodeToString(bytes)[:length]
}

// RegisterHandler registers a new variable handler
func RegisterHandler(name string, handler VariableHandler) {
	Handlers[name] = handler
}

// GetHandler retrieves a handler by name
func GetHandler(name string) (VariableHandler, bool) {
	handler, ok := Handlers[name]
	return handler, ok
}

// IsVariable checks if a string is a variable reference
func IsVariable(value string) bool {
	return len(value) > 2 && value[0] == '{' && value[len(value)-1] == '}'
}

// ExtractVariableName extracts the variable name from a reference like {VAR_NAME}
func ExtractVariableName(value string) string {
	if !IsVariable(value) {
		return ""
	}
	return value[1 : len(value)-1]
}

// ProcessVariable processes a variable and returns its content
func ProcessVariable(value string, path string, context map[string]any) ([]byte, error) {
	if !IsVariable(value) {
		return nil, fmt.Errorf("not a variable: %s", value)
	}
	
	varName := ExtractVariableName(value)
	handler, ok := GetHandler(varName)
	if !ok {
		return nil, fmt.Errorf("unknown variable: %s", varName)
	}
	
	return handler(path, context)
}

// ListVariables returns a list of all available variables
func ListVariables() []string {
	vars := make([]string, 0, len(Handlers))
	for name := range Handlers {
		vars = append(vars, name)
	}
	return vars
}