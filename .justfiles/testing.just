# Test recipes for guild-scaffold

# Show available test commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Run all tests (unit + integration)
[no-cd]
all: unit integration
    @echo "✅ All tests passed"

# Run unit tests
[no-cd]
unit:
    @echo "Running unit tests..."
    go test -v -race -cover ./pkg/...

# Run unit tests (fast, no race detector)
[no-cd]
unit-fast:
    @echo "Running quick unit tests..."
    go test ./pkg/...

# Run unit tests with verbose output
[no-cd]
unit-verbose:
    @echo "Running unit tests (verbose)..."
    go test -v -race ./pkg/... -count=1

# Run integration tests (requires Docker/Colima)
[no-cd]
integration:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building Linux binary for containers..."
    just build-linux
    echo "Running container-based integration tests..."
    DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock" \
    TESTCONTAINERS_RYUK_DISABLED=true \
    go test -v -tags=integration ./tests/integration/... -timeout 10m

# Run a specific integration test
[no-cd]
integration-single TEST:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building Linux binary..."
    just build-linux
    echo "Running test: {{TEST}}"
    DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock" \
    TESTCONTAINERS_RYUK_DISABLED=true \
    go test -v -tags=integration ./tests/integration/... -run "{{TEST}}" -timeout 5m

# Run integration tests (verbose)
[no-cd]
integration-verbose:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building Linux binary..."
    just build-linux
    echo "Running integration tests (verbose)..."
    DOCKER_HOST="unix://${HOME}/.colima/default/docker.sock" \
    TESTCONTAINERS_RYUK_DISABLED=true \
    go test -v -tags=integration ./tests/integration/... -timeout 10m -count=1

# Run tests with coverage
[no-cd]
coverage:
    @echo "Running tests with coverage..."
    go test -coverprofile=coverage.out ./pkg/...
    go tool cover -func=coverage.out

# Generate HTML coverage report
[no-cd]
coverage-html:
    @echo "Generating HTML coverage report..."
    go test -coverprofile=coverage.out ./pkg/...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run benchmarks
[no-cd]
bench:
    @echo "Running benchmarks..."
    go test -bench=. -benchmem ./pkg/scaffold/...

# Run tests matching a pattern
[no-cd]
pattern PATTERN:
    @echo "Running tests matching: {{PATTERN}}"
    go test -v ./pkg/... -run "{{PATTERN}}"

# Run tests on a specific package
[no-cd]
pkg PKG:
    @echo "Testing package: {{PKG}}"
    go test -v -race -cover ./pkg/{{PKG}}/...

# Fuzz testing
[no-cd]
fuzz TARGET='FuzzParser' TIME='10s':
    @echo "Running fuzz test: {{TARGET}} for {{TIME}}"
    go test -fuzz={{TARGET}} -fuzztime={{TIME}} ./pkg/scaffold/...
