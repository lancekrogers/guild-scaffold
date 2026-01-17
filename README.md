# Guild Scaffold

A generic, template-agnostic project scaffolding tool. Create and manage reusable project templates with variable substitution and directory structure generation.

## Installation

```bash
# From source
just install

# Or manually
go build -o bin/scaffold ./cmd/scaffold
cp bin/scaffold ~/go/bin/
```

## Quick Start

```bash
# List available scaffolds
scaffold list

# Initialize a new project from a scaffold
scaffold init my-project --template guild-campaign

# Validate a scaffold definition
scaffold validate --template guild-campaign
```

## Registry System

Guild-scaffold uses a registry system to discover scaffolds from multiple sources:

### Source Priority (highest to lowest)

1. **Project Registry** (`.campaign/scaffold.yaml`) - Project-specific scaffolds
2. **Global Registry** (`~/.guild/scaffold.yaml`) - User-defined scaffolds
3. **Builtin Scaffolds** - Embedded default scaffolds (e.g., `guild-campaign`)

When the same scaffold name exists in multiple sources, the higher-priority source wins.

### Registry Configuration

```yaml
# ~/.guild/scaffold.yaml or .campaign/scaffold.yaml
scaffolds:
  - name: my-scaffold
    path: ~/scaffolds/my-scaffold/
    description: "My custom project scaffold"

  - name: go-cli
    path: /path/to/scaffolds/go-cli/
    description: "Go CLI application template"
```

## Creating Custom Scaffolds

Each scaffold is a directory containing:

```
my-scaffold/
├── scaffold.yaml      # Scaffold definition
└── templates/         # Template files
    ├── README.md.tmpl
    └── config.yaml.tmpl
```

### Scaffold Definition (`scaffold.yaml`)

```yaml
name: my-scaffold
version: "1.0"
description: "Description of this scaffold"

# Variable definitions
variables:
  project_name:
    type: string
    required: true
    description: "Name of the project"

  port:
    type: integer
    default: 8080
    description: "Server port"

# Directory structure (tree format)
tree:
  README.md: README.md.tmpl
  config/:
    app.yaml: config.yaml.tmpl
  src/:
    _empty: true  # Creates .gitkeep in empty directory
  logs/:
    _empty: true
  data/:
    cache/:
      _empty: true
```

### Empty Directories

Git doesn't track empty directories. To ensure empty directories are created and persist in version control, use `_empty: true`:

```yaml
myproject/:
  logs/:
    _empty: true
  data/:
    cache/:
      _empty: true
    temp/:
      _empty: true
```

This creates a `.gitkeep` file in each empty directory, allowing Git to track them. When you initialize a project from this scaffold:
- `logs/.gitkeep` is created
- `data/cache/.gitkeep` is created
- `data/temp/.gitkeep` is created

The `.gitkeep` file itself is empty and serves as a placeholder. If you later add actual files to these directories, you can safely delete the `.gitkeep` files.

### Template Syntax

Templates use Go's `text/template` syntax with additional functions:

```
# {{ .vars.project_name }}

Port: {{ .vars.port | default 8080 }}
Name: {{ .vars.project_name | lower }}
```

#### Available Template Functions

**String manipulation:**
- `lower`, `upper`, `title` - Case conversion
- `trim`, `trimPrefix`, `trimSuffix` - Whitespace handling
- `replace old new` - String replacement
- `split delimiter` - Split into array
- `join delimiter` - Join array into string

**Formatting:**
- `quote` - Wrap in quotes
- `indent n` - Indent text by n spaces
- `toYAML`, `toJSON` - Format as YAML/JSON

**Path operations:**
- `base`, `dir`, `ext` - Path components
- `clean` - Clean path

**Conditionals:**
- `default value` - Provide default if empty
- `empty` - Check if value is empty
- `coalesce` - Return first non-empty value

## Commands

### `scaffold list`

List all available scaffolds from the merged registry.

```bash
scaffold list           # Brief list
scaffold list -v        # Verbose with variables
```

### `scaffold init`

Initialize a new project from a scaffold.

```bash
scaffold init my-project --template go-cli
scaffold init my-project --template go-cli --dry-run  # Preview only
scaffold init my-project --template go-cli --force    # Overwrite existing
```

### `scaffold validate`

Validate a scaffold definition and templates.

```bash
scaffold validate --template my-scaffold
```

## Development

```bash
# Run tests
just test unit          # Unit tests
just test integration   # Container-based integration tests

# Build
just build              # Build binary
just check              # Vet + build

# Development
just dev watch          # Watch mode
just dev pre-commit     # Pre-commit checks
```

## Project Structure

```
guild-scaffold/
├── cmd/scaffold/           # CLI entry point
├── pkg/scaffold/           # Core library
│   ├── registry.go         # Registry types and operations
│   ├── loader.go           # Registry loading from sources
│   ├── parser.go           # YAML parsing
│   ├── renderer.go         # Template rendering
│   ├── validator.go        # Scaffold validation
│   ├── filesystem.go       # Safe file operations
│   ├── cli/                # CLI command implementations
│   └── builtin/            # Embedded scaffolds
├── tests/integration/      # Container-based tests
└── .justfiles/             # Modular just recipes
```

## License

See LICENSE file.
