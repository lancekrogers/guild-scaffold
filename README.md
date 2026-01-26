# Guild Scaffold

A template-agnostic project scaffolding tool. Create and manage reusable project templates with variable substitution and directory structure generation.

## Features

- **Template-based scaffolding** - Define project structures with Go templates
- **Variable substitution** - Customizable variables with types, defaults, and validation
- **Registry system** - Discover scaffolds from multiple sources with precedence rules
- **Dry-run mode** - Preview what will be created before executing
- **Empty directory tracking** - Automatic `.gitkeep` generation for empty directories
- **Configurable paths** - Override default directories via environment variables

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
scaffold init my-project --template minimal

# Validate a scaffold definition
scaffold validate --template my-scaffold

# Preview without creating files
scaffold init my-project --template my-scaffold --dry-run
```

## Configuration

### Environment Variables

Guild Scaffold supports environment variable overrides for customizing paths:

| Variable | Default | Description |
|----------|---------|-------------|
| `GUILD_SCAFFOLD_GLOBAL_DIR` | `~/.config/guild/` | Global configuration and templates directory |
| `GUILD_SCAFFOLD_WORKSPACE_DIR` | `.campaign` | Workspace directory name (relative to project root) |
| `XDG_CONFIG_HOME` | `~/.config` | Standard XDG base directory for config |

Example:
```bash
# Use a custom global templates directory
export GUILD_SCAFFOLD_GLOBAL_DIR=~/.my-scaffolds/

# Use a custom workspace directory
export GUILD_SCAFFOLD_WORKSPACE_DIR=.scaffold
```

## Registry System

Guild-scaffold uses a registry system to discover scaffolds from multiple sources:

### Source Priority (highest to lowest)

1. **Workspace Templates** (`.campaign/templates/`) - Project-specific scaffolds
2. **Global Templates** (`~/.config/guild/templates/`) - User-installed scaffolds
3. **Legacy Registries** - For backward compatibility with older configurations

When the same scaffold name exists in multiple sources, the higher-priority source wins.

### Installing Templates

To install a scaffold from the examples directory:

```bash
# Copy an example scaffold to your global templates directory
cp -r examples/go-cli ~/.config/guild/templates/

# Or create a new scaffold
mkdir -p ~/.config/guild/templates/my-scaffold
# Then create scaffold.yaml and templates/ in that directory
```

### Legacy Registry Configuration

For backward compatibility, registry YAML files are also supported:

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
└── templates/         # Template files (mirrors output structure)
    ├── README.md.tmpl
    ├── go.mod.tmpl
    └── cmd/
        └── app/
            └── main.go.tmpl
```

### Scan-Based Scaffolds (Recommended)

The simplest approach - templates mirror your desired output structure:

```yaml
# scaffold.yaml
scaffold_version: "1.0.0"
templates_dir: templates
mirror_structure: true
scan_templates: true

vars:
  project_name: ""
  module_path: ""
  description: "A project"
```

With `scan_templates: true`, the directory structure under `templates/` becomes your output. Just create your templates where you want them to appear.

### Explicit Tree Scaffolds

For more control, define the structure explicitly:

```yaml
name: my-scaffold
version: "1.0"
description: "Description of this scaffold"

variables:
  project_name:
    type: string
    required: true

tree:
  README.md: README.md.tmpl
  config/:
    app.yaml: config.yaml.tmpl
  src/:
    _empty: true  # Creates .gitkeep
```

### Variable Types

| Type | Description | Constraints |
|------|-------------|-------------|
| `string` | Text value | `pattern`, `enum` |
| `int`/`integer` | Whole number | `min`, `max` |
| `bool` | Boolean | - |
| `array` | List of values | - |
| `object` | Key-value map | - |

### Empty Directories

Git doesn't track empty directories. Use `_empty: true` to create directories with `.gitkeep`:

```yaml
tree:
  logs/:
    _empty: true
  data/:
    cache/:
      _empty: true
```

## Template Syntax

Templates use Go's `text/template` syntax with additional functions:

```
# {{ .vars.project_name }}

Port: {{ .vars.port | default 8080 }}
Name: {{ .vars.project_name | lower }}
Hash: {{ hash .vars.project_name }}
```

### Available Template Functions

**String manipulation:**
- `lower`, `upper`, `title` - Case conversion
- `trimSpace` - Remove leading/trailing whitespace
- `replace old new` - String replacement
- `split delimiter` - Split into array
- `join delimiter array` - Join array into string
- `contains`, `hasPrefix`, `hasSuffix` - String checks

**Formatting:**
- `quote` - Wrap in double quotes
- `indent n text` - Indent text by n spaces
- `toYAML`, `toJSON` - Format as YAML/JSON

**Path operations:**
- `pathBase`, `pathDir`, `pathExt` - Path components
- `pathJoin`, `pathClean` - Path manipulation

**Utilities:**
- `default value fallback` - Provide default if empty
- `empty value` - Check if value is empty
- `not bool` - Negate boolean
- `hash input` - Generate short SHA256 hash (8 chars)

**Date/Time:**
- `now` - Current time
- `date format` - Format current date
- `dateISO` - ISO 8601 formatted date

**Type checking:**
- `isString`, `isMap`, `isList` - Type assertions

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

### `scaffold sync`

Sync scaffolds from a source to your global templates directory.

```bash
scaffold sync --from /path/to/scaffolds
```

## Examples

The `examples/` directory contains sample scaffolds:

- `go-project/` - **Recommended** - Uses scan-based approach with mirrored structure
- `minimal/` - Bare-bones scaffold with explicit tree
- `go-cli/` - Go CLI with explicit tree and justfile
- Additional YAML examples for reference

To use an example:

```bash
# Copy to global templates
cp -r examples/go-project ~/.config/guild/templates/

# Then use it
scaffold init my-app --template go-project \
  --var project_name=my-app \
  --var module_path=github.com/user/my-app
```

## Development

```bash
# Run tests
just test unit          # Unit tests
just test integration   # Container-based integration tests

# Build
just build              # Build binary
just check              # Vet + build

# Lint
just lint               # Run linter

# Development
just dev watch          # Watch mode
just dev pre-commit     # Pre-commit checks
```

## Project Structure

```
guild-scaffold/
├── cmd/scaffold/           # CLI entry point
├── pkg/scaffold/           # Core library
│   ├── config/             # Path configuration
│   ├── cli/                # CLI command implementations
│   ├── registry.go         # Registry types and operations
│   ├── loader.go           # Registry loading from sources
│   ├── parser.go           # YAML parsing
│   ├── renderer.go         # Template rendering
│   ├── validator.go        # Scaffold validation
│   └── filesystem.go       # Safe file operations
├── examples/               # Example scaffolds
├── tests/integration/      # Container-based tests
└── .justfiles/             # Modular just recipes
```

## Library Usage

Guild Scaffold can be used as a Go library:

```go
import "github.com/lancekrogers/guild-scaffold/pkg/scaffold"

// Create a registry loader
loader, err := scaffold.NewRegistryLoader()
if err != nil {
    log.Fatal(err)
}

// Load all registries
registry, err := loader.Load(ctx)
if err != nil {
    log.Fatal(err)
}

// Find and resolve a scaffold
entry, err := loader.FindScaffold(ctx, "my-scaffold")
if err != nil {
    log.Fatal(err)
}

def, fsys, err := scaffold.ResolveScaffold(ctx, entry)
if err != nil {
    log.Fatal(err)
}
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
