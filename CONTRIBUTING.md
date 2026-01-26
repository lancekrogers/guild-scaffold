# Contributing to Guild Scaffold

Thank you for your interest in contributing to Guild Scaffold!

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/guild-scaffold.git
   cd guild-scaffold
   ```
3. Install dependencies:
   ```bash
   just install
   ```

## Development Workflow

### Building

```bash
just build              # Build the binary
just check              # Run vet and build
```

### Testing

```bash
just test unit          # Run unit tests
just test integration   # Run container-based integration tests
```

### Linting

```bash
just lint               # Run golangci-lint
just fmt                # Format code
```

### Pre-commit Checks

Before submitting a PR, run:
```bash
just dev pre-commit
```

## Project Structure

- `cmd/scaffold/` - CLI entry point and commands
- `pkg/scaffold/` - Core library code
  - `config/` - Path and configuration management
  - `cli/` - CLI-specific implementations
- `examples/` - Example scaffold definitions
- `tests/integration/` - Container-based integration tests

## Code Style

### Go Standards

- Follow standard Go conventions and idioms
- Use `context.Context` for I/O operations
- Wrap errors with contextual information
- Keep functions under 50 lines, files under 500 lines

### Error Handling

All errors should include context:
```go
if err != nil {
    return fmt.Errorf("failed to load scaffold (name=%s): %w", name, err)
}
```

### Testing

- Write table-driven tests for multiple scenarios
- Test error cases explicitly
- Use meaningful test names that describe the scenario

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear, atomic commits
3. Ensure all tests pass: `just test unit`
4. Run pre-commit checks: `just dev pre-commit`
5. Update documentation if needed
6. Submit a pull request with a clear description

### Commit Messages

Use clear, descriptive commit messages:
- Start with a verb (Add, Fix, Update, Remove)
- Keep the first line under 72 characters
- Reference issues when applicable

Good:
```
Add hash() template function for generating short hashes

The hash() function generates an 8-character SHA256 hash from any
string input. This is useful for generating unique identifiers in
scaffolded projects.
```

## Adding New Features

### New Template Functions

1. Add the function to `getTemplateFuncMap()` in `pkg/scaffold/renderer.go`
2. Add tests in `renderer_test.go`
3. Document in README.md under "Available Template Functions"

### New Scaffold Examples

1. Create a directory in `examples/` with:
   - `scaffold.yaml` - Scaffold definition
   - `templates/` - Template files
2. Update `examples/README.md` if it exists
3. Add to the list in the main README.md

## Questions?

Open an issue for questions or discussion about proposed changes.
