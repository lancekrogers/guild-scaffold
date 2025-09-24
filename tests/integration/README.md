# Container-Based Integration Testing

This directory contains container-based integration tests for the guild-scaffold tool. These tests ensure that the scaffold tool correctly generates file structures without affecting the host filesystem.

## Architecture

### Why Container-Based Testing?

1. **Filesystem Isolation**: Tests run in isolated containers, preventing any changes to the developer's filesystem
2. **Real OS Behavior**: Tests actual OS filesystem operations (permissions, symlinks, etc.)
3. **Cross-Platform**: Consistent testing environment regardless of host OS
4. **CI/CD Ready**: Works seamlessly in CI pipelines
5. **Parallel Execution**: Each test gets its own container, enabling parallel test runs

### Test Structure

```
tests/integration/
├── README.md           # This file
├── helpers.go          # Container management utilities
├── scaffold_test.go    # Main integration tests
├── simple_test.go      # Simple smoke tests
└── fixtures/           # Test data
    ├── *.yaml         # Test scaffold definitions
    ├── templates/     # Template files for testing
    └── expected/      # Expected output structures
```

## Running Tests

### Prerequisites

- Docker installed and running
- Go 1.21 or later
- `just` command installed (`brew install just` or `cargo install just`)

### Test Commands

```bash
# Run all integration tests with containers
just integration-test

# Run a specific integration test
just integration-test-single TestMinimalScaffold

# Run tests without containers (memory filesystem only)
just integration-test-local

# Run in CI mode (extended timeouts)
just integration-test-ci
```

### Docker-Specific Commands

```bash
# Run tests in Alpine container
just docker-run-alpine examples/minimal.yaml

# Run tests in Ubuntu container
just docker-run-ubuntu examples/minimal.yaml

# Test with multiple Go versions
just docker-test-go-versions

# Clean up test containers
just docker-clean
```

## Test Scenarios

### 1. Basic Functionality (`TestMinimalScaffoldInContainer`)
- Creates a simple project structure
- Verifies template variable substitution
- Checks file creation and content

### 2. Complex Structures (`TestScaffoldComplexStructure`)
- Tests deep directory hierarchies
- Multiple template variables
- Conditional file generation

### 3. Empty Directories (`TestScaffoldEmptyDirectories`)
- Ensures empty directories are created correctly
- Tests the `_empty` directive

### 4. Error Handling
- **Existing Files**: Tests behavior with existing files (with/without --force)
- **Path Traversal**: Ensures malicious paths are blocked
- **Invalid Templates**: Verifies proper error reporting

### 5. Dry Run Mode (`TestScaffoldDryRun`)
- Verifies no files are created
- Checks preview output accuracy

## Writing New Tests

### Basic Test Template

```go
func TestYourFeature(t *testing.T) {
    // Skip if Docker not available
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Create test container
    container, err := NewTestContainer(t)
    require.NoError(t, err)
    defer container.Cleanup()

    // Copy fixtures to container
    err = container.CopyToContainer("fixtures/your-template.yaml", "/test/template.yaml")
    require.NoError(t, err)

    // Run scaffold
    output, err := container.RunScaffold(
        "init", "test-project",
        "--template", "/test/template.yaml",
        "--output", "/output",
    )
    require.NoError(t, err)

    // Verify results
    exists, err := container.CheckFileExists("/output/expected-file.txt")
    require.NoError(t, err)
    require.True(t, exists)
}
```

### Helper Functions

- `NewTestContainer(t)`: Creates a new Alpine container with scaffold binary
- `CopyToContainer(src, dst)`: Copies files into the container
- `CopyDirToContainer(srcDir, dstDir)`: Copies directories recursively
- `RunScaffold(args...)`: Executes scaffold command in container
- `CheckFileExists(path)`: Checks if a file exists in container
- `CheckDirExists(path)`: Checks if a directory exists
- `ReadFile(path)`: Reads file content from container
- `ListDirectory(path)`: Lists files in a directory
- `CaptureSnapshot(path)`: Captures filesystem state for comparison
- `ValidateSnapshot(actual, expected)`: Compares filesystem snapshots

## Container Details

### Base Image
- Default: `alpine:latest` (5MB base image)
- Minimal overhead, fast startup
- Can be customized via `IMAGE` environment variable

### Container Lifecycle
1. Container created with scaffold binary mounted
2. Test fixtures copied into container
3. Scaffold command executed
4. Output validated
5. Container automatically cleaned up

### Resource Management
- Containers are ephemeral (removed after test)
- Automatic cleanup on test failure
- Parallel test execution supported
- Resource limits can be configured

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - name: Run Integration Tests
        run: just integration-test-ci
```

### Local Development

For rapid iteration during development:

```bash
# Run tests with file watching
just dev-test

# Run specific test with verbose output
just test-pattern TestMinimal

# Debug a failing test
just debug-test TestScaffoldComplexStructure
```

## Troubleshooting

### Common Issues

1. **Docker not running**
   ```
   Error: Cannot connect to Docker daemon
   Solution: Start Docker Desktop or dockerd service
   ```

2. **Permission denied**
   ```
   Error: Permission denied while trying to connect to Docker
   Solution: Add user to docker group or use sudo
   ```

3. **Container startup timeout**
   ```
   Error: Container failed to start within timeout
   Solution: Increase timeout in test or check Docker resources
   ```

### Debugging Tips

1. **Keep container running for inspection**:
   ```go
   // Comment out defer container.Cleanup() to keep container alive
   // Then inspect with: docker exec -it <container-id> sh
   ```

2. **Enable verbose output**:
   ```bash
   just test-verbose
   ```

3. **Check container logs**:
   ```bash
   docker logs <container-id>
   ```

## Performance Considerations

- Container startup: ~100-200ms per test
- File operations: Native speed within container
- Test parallelization: Limited by Docker daemon resources
- Recommended: 4-8 parallel containers maximum

## Future Enhancements

- [ ] Test matrix for multiple OS versions
- [ ] Performance benchmarking suite
- [ ] Snapshot testing for generated code
- [ ] Integration with test coverage tools
- [ ] Automated fixture generation from examples