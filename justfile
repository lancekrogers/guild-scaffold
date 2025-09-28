# Guild Scaffold - Development Commands
# https://github.com/casey/just
#
# ⚠️  SAFETY NOTICE:
# This justfile prioritizes containerized testing to protect your filesystem.
# Commands marked with "unsafe" can modify your local filesystem.
# Default commands (like 'just test') only run in containers.
#
# Safe commands: test, integration-test, test-minimal-container, run-container
# Unsafe commands: *-unsafe commands require confirmation

# Import modular recipes
import '.just/testing.just'
import '.just/docker.just'
import '.just/development.just'

# Default recipe - show available commands
default:
    @echo "Guild Scaffold - Safe Testing Commands"
    @echo "======================================"
    @echo ""
    @echo "Safe Container Commands:"
    @echo "  test                - Run all tests in containers"
    @echo "  integration-test    - Run integration tests in containers"
    @echo "  test-minimal-container - Test minimal template in container"
    @echo "  run-container       - Run scaffold in container"
    @echo ""
    @echo "Other Commands:"
    @echo "  build              - Build the scaffold binary"
    @echo "  clean              - Clean build artifacts"
    @echo "  coverage           - Generate coverage report"
    @echo ""
    @echo "Use 'just --list' to see all commands (including unsafe ones)"

# Build the scaffold binary
build:
    @echo "Building scaffold..."
    @mkdir -p bin
    go build -v -o bin/scaffold ./cmd/scaffold

# Run all tests (container-safe)
test: unit-test integration-test

# Run unit tests
unit-test:
    @echo "Running unit tests..."
    go test -v -race -cover ./pkg/...

# Run integration tests with containers (requires Docker)
integration-test:
    @echo "Running container-based integration tests..."
    @echo "ℹ️  These tests run in containers and won't affect your filesystem"
    go test -v -tags=integration ./tests/integration/... -timeout 10m

# Run integration tests locally (WARNING: may affect filesystem)
integration-test-local-unsafe:
    @echo "⚠️  WARNING: This command may affect your filesystem!"
    @echo "Use 'just integration-test' for safe container-based testing"
    @echo "Press Ctrl+C to cancel, Enter to continue..."
    @read
    go test -v ./pkg/scaffold/... -run ".*Integration.*"

# Run integration tests in CI environment
integration-test-ci:
    @echo "Running CI integration tests..."
    go test -v -tags=integration ./tests/integration/... -timeout 15m -short=false

# Run a specific integration test
integration-test-single TEST:
    @echo "Running single integration test: {{TEST}}"
    go test -v -tags=integration ./tests/integration/... -run "{{TEST}}" -timeout 5m

# Clean build artifacts
clean:
    @echo "Cleaning..."
    rm -rf bin
    rm -rf tests/integration/tmp
    go clean -testcache

# Install the binary
install: build
    @echo "Installing scaffold..."
    go install ./cmd/scaffold

# Run tests with coverage
coverage:
    @echo "Running tests with coverage..."
    go test -v -race -coverprofile=coverage.out ./pkg/...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...
    gofmt -s -w .

# Run linters
lint:
    @echo "Running linters..."
    go vet ./...

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download
    go mod tidy

# Quick test (unit tests only, no race detector)
quick:
    @echo "Running quick tests (safe - no filesystem operations)..."
    go test ./pkg/...

# Run scaffold in container (safe)
run-container TEMPLATE *ARGS:
    @echo "Running scaffold in container (safe)..."
    @echo "Building Linux binary..."
    @GOOS=linux GOARCH=amd64 go build -o bin/scaffold-linux ./cmd/scaffold
    docker run --rm \
        -v $(pwd)/bin/scaffold-linux:/scaffold \
        -v $(pwd)/examples:/examples:ro \
        alpine:latest \
        /scaffold init test-project \
        --template /examples/{{TEMPLATE}} \
        --output /output \
        {{ARGS}}

# UNSAFE: Run scaffold locally (WARNING: affects filesystem)
run-local-unsafe TEMPLATE OUTPUT *ARGS:
    @echo "⚠️  WARNING: This will affect your filesystem at {{OUTPUT}}!"
    @echo "Use 'just run-container' for safe testing"
    @echo "Press Ctrl+C to cancel, Enter to continue..."
    @read
    go run ./cmd/scaffold init test-project \
        --template {{TEMPLATE}} \
        --output {{OUTPUT}} \
        {{ARGS}}

# Test with minimal template (runs in container)
test-minimal-container:
    @echo "Testing minimal template in container..."
    @echo "Building Linux binary for container..."
    @GOOS=linux GOARCH=amd64 go build -o bin/scaffold-linux ./cmd/scaffold
    @echo "Running in container..."
    docker run --rm \
        -v $(pwd)/bin/scaffold-linux:/scaffold \
        -v $(pwd)/examples:/examples:ro \
        alpine:latest \
        /scaffold init test-minimal \
        --template /examples/minimal.yaml \
        --output /output \
        --var project_name=TestProject \
        --var module_name=github.com/test/project

# Test with library template (runs in container)
test-library-container:
    @echo "Testing library template in container..."
    @echo "Building Linux binary for container..."
    @GOOS=linux GOARCH=amd64 go build -o bin/scaffold-linux ./cmd/scaffold
    @echo "Running in container..."
    docker run --rm \
        -v $(pwd)/bin/scaffold-linux:/scaffold \
        -v $(pwd)/examples:/examples:ro \
        alpine:latest \
        /scaffold init test-library \
        --template /examples/library.yaml \
        --output /output \
        --var package_name=mylib \
        --var module_name=github.com/test/mylib

# UNSAFE: Test locally (WARNING: affects filesystem)
test-minimal-unsafe:
    @echo "⚠️  WARNING: This will write to /tmp on your filesystem!"
    @echo "Use 'just test-minimal-container' for safe testing"
    @echo "Press Ctrl+C to cancel, Enter to continue..."
    @read
    go run ./cmd/scaffold init test-minimal \
        --template examples/minimal.yaml \
        --output /tmp/scaffold-test-minimal \
        --var project_name=TestProject \
        --var module_name=github.com/test/project \
        --force

# Build and run in Docker for isolated testing
docker-test:
    @echo "Building Docker image for testing..."
    docker build -t scaffold-test .
    @echo "Running in isolated container (safe)..."
    docker run --rm \
        -v $(pwd)/examples:/examples:ro \
        scaffold-test \
        init test-project \
        --template /examples/minimal.yaml \
        --output /output
    @echo "✅ Test completed in container (no filesystem changes)"

# Watch for changes and run tests
watch:
    @echo "Watching for changes..."
    fswatch -o . | xargs -n1 -I{} just quick

# Check for common issues
check:
    @echo "Running checks..."
    go vet ./...
    go mod verify
    @echo "✅ All checks passed"

# Generate test fixtures
generate-fixtures:
    @echo "Generating test fixtures..."
    @mkdir -p tests/integration/fixtures/templates
    @mkdir -p tests/integration/fixtures/expected
    @echo "✅ Fixture directories created"

# Benchmark the scaffold engine
bench:
    @echo "Running benchmarks..."
    go test -bench=. -benchmem ./pkg/scaffold/...

# Run security scan
security:
    @echo "Running security scan..."
    go list -json -deps ./... | nancy sleuth

# Show test coverage in terminal
cover-report:
    @echo "Generating coverage report..."
    go test -coverprofile=coverage.out ./pkg/...
    go tool cover -func=coverage.out

# Initialize development environment
init-dev:
    @echo "Initializing development environment..."
    just deps
    just generate-fixtures
    @echo "✅ Development environment ready"
    @echo ""
    @echo "🛡️  Remember: Use container-based commands for testing:"
    @echo "  - 'just test' for all tests"
    @echo "  - 'just integration-test' for integration tests"
    @echo "  - Avoid '*-unsafe' commands unless necessary"