# Guild Scaffold Examples

This directory contains example scaffold templates demonstrating different project structures that can be generated using guild-scaffold.

## Structure

```
examples/
├── README.md                 # This file
├── *.yaml                    # Scaffold definitions using tree-like YAML format
└── templates/                # Template files referenced by the YAML definitions
    ├── minimal/             # Templates for minimal project
    ├── go/                  # Templates for standard Go project
    ├── cli/                 # Templates for CLI tool
    ├── library/             # Templates for Go library
    ├── microservice/        # Templates for microservice
    ├── webapp/              # Templates for web application
    ├── monorepo/            # Templates for monorepo
    ├── campaign/            # Templates for Guild campaign workspace
    └── dynamic/             # Templates demonstrating dynamic variables
```

## Example Definitions

Each `.yaml` file defines a project structure using a tree-like format:

- **minimal.yaml** - Bare minimum project structure
- **simple_go_project.yaml** - Standard Go application layout
- **cli_tool.yaml** - Command-line tool with subcommands
- **library.yaml** - Reusable Go package/library
- **microservice.yaml** - Production-ready microservice
- **web_app.yaml** - Full-stack web application
- **monorepo.yaml** - Multi-project repository
- **campaign.yaml** - Guild campaign workspace for AI development
- **dynamic.yaml** - Demonstrates variable usage and special markers

## YAML Format

The examples use a tree-like YAML structure:

```yaml
_scaffold_version: "1.0.0"
_templates_dir: "minimal"
_vars:
  project_name: "MyProject"

project/:
  src/:
    _files:
      main.go: main.go.tmpl
  _files:
    README.md: readme.md.tmpl
    .gitignore: gitignore.tmpl
```

### Special Keys

- `_scaffold_version` - Version of scaffold format
- `_templates_dir` - Base directory for templates
- `_vars` - Global variables available to all templates
- `_files` - Marks file entries in a directory
- `_empty: true` - Creates an empty directory with .gitkeep

### Special Template Values

- `"~"` - Creates an empty file
- `"{ENV_FILE}"` - Uses special variable for environment file
- `"{UUID_FILE}"` - Generates UUID file
- `"{DATABASE_FILE}"` - Creates database initialization file
- `"{SOCKET_FILE}"` - Creates socket file

## Usage

To use these examples with guild-scaffold:

```bash
# Initialize a project using an example
scaffold init myproject --template examples/minimal.yaml

# Or specify custom variables
scaffold init myapp --template examples/cli_tool.yaml \
  --var ProjectName=MyCliTool \
  --var ModuleName=github.com/myorg/mycli
```

## Creating Your Own Templates

1. Create a YAML file defining your project structure
2. Create corresponding template files in `templates/your-template/`
3. Use Go template syntax in `.tmpl` files
4. Reference variables with `{{ .VarName }}`

## Template Development

When developing templates:

1. Keep templates simple and focused
2. Use meaningful variable names
3. Provide sensible defaults with `{{ .Var | default "value" }}`
4. Document required and optional variables
5. Test templates with different variable combinations

## Contributing

To add new example templates:

1. Create a descriptive YAML definition file
2. Add corresponding templates in `templates/` directory
3. Update this README with the new example
4. Include sample usage in your template's documentation
