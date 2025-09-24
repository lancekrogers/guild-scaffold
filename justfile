# Guild Scaffold - Development Commands
# https://github.com/casey/just

# Import modular recipes
import '.just/testing.just'
import '.just/docker.just'
import '.just/development.just'

# Default recipe - show available commands
default:
    @just --list --unsorted

# Build the scaffold binary
build:
    @echo "Building scaffold..."
    @mkdir -p bin
    go build -v -o bin/scaffold ./cmd/scaffold

# Run all tests
test: unit-test integration-test-local

# Run unit tests
unit-test:
    @echo "Running unit tests..."
    go test -v -race -cover ./pkg/...

# Run integration tests with containers (requires Docker)
integration-test:
    @echo "Running container-based integration tests..."
    go test -v -tags=integration ./tests/integration/... -timeout 10m

# Run integration tests with memory filesystem (no Docker required)
integration-test-local:
    @echo "Running local integration tests..."
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
    @echo "Running quick tests..."
    go test ./pkg/...

# Run local scaffold for testing
run-local TEMPLATE OUTPUT *ARGS:
    @echo "Running scaffold locally..."
    go run ./cmd/scaffold init test-project \
        --template {{TEMPLATE}} \
        --output {{OUTPUT}} \
        {{ARGS}}

# Test with minimal template
test-minimal:
    @echo "Testing minimal template..."
    go run ./cmd/scaffold init test-minimal \
        --template examples/minimal.yaml \
        --output /tmp/scaffold-test-minimal \
        --var project_name=TestProject \
        --var module_name=github.com/test/project \
        --force

# Test with library template
test-library:
    @echo "Testing library template..."
    go run ./cmd/scaffold init test-library \
        --template examples/library.yaml \
        --output /tmp/scaffold-test-library \
        --var package_name=mylib \
        --var module_name=github.com/test/mylib \
        --force

# Build and run in Docker for isolated testing
docker-test:
    @echo "Building Docker image for testing..."
    docker build -t scaffold-test .
    docker run --rm -v $(pwd)/examples:/examples scaffold-test \
        init test-project \
        --template /examples/minimal.yaml \
        --output /output

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