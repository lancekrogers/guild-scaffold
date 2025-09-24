# Development workflow recipes for guild-scaffold

# Start development with file watching
dev:
    @echo "Starting development mode..."
    watchexec -e go -r -- go run ./cmd/scaffold

# Run continuous testing
dev-test:
    @echo "Starting continuous testing..."
    watchexec -e go -- just quick

# Generate a new template scaffold
new-template NAME:
    @echo "Creating new template: {{NAME}}"
    @mkdir -p examples/{{NAME}}/templates
    @echo "# {{NAME}} Template\n_scaffold_version: \"1.0.0\"\n_templates_dir: \"templates\"" > examples/{{NAME}}.yaml
    @echo "Template scaffold created at examples/{{NAME}}"

# Validate all example templates
validate-examples:
    @echo "Validating example templates..."
    @for f in examples/*.yaml; do \
        echo "Validating $$f..."; \
        go run ./cmd/scaffold init test --template $$f --output /tmp/validate --dry-run || exit 1; \
    done
    @echo "✅ All examples valid"

# Check for TODO/FIXME comments
check-todos:
    @echo "Checking for TODO/FIXME comments..."
    @grep -r "TODO\|FIXME" --include="*.go" . | head -20 || echo "No TODOs found"

# Update dependencies
update-deps:
    @echo "Updating dependencies..."
    go get -u ./...
    go mod tidy
    go mod verify

# Generate mocks for testing
generate-mocks:
    @echo "Generating mocks..."
    go generate ./...

# Run pre-commit checks
pre-commit:
    @echo "Running pre-commit checks..."
    just fmt
    just lint
    just test-fast
    @echo "✅ Pre-commit checks passed"

# Show code statistics
stats:
    @echo "Code statistics:"
    @echo "Lines of Go code:"
    @find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
    @echo "\nNumber of Go files:"
    @find . -name "*.go" -not -path "./vendor/*" | wc -l
    @echo "\nTest coverage:"
    @go test -cover ./pkg/... | grep -o '[0-9]*\.[0-9]*%' | tail -1

# Debug a specific test
debug-test TEST:
    @echo "Debugging test: {{TEST}}"
    dlv test ./pkg/... -- -test.run "{{TEST}}"

# Profile memory usage
profile-mem:
    @echo "Profiling memory usage..."
    go test -memprofile=mem.prof -bench=. ./pkg/scaffold/...
    go tool pprof -top mem.prof

# Profile CPU usage
profile-cpu:
    @echo "Profiling CPU usage..."
    go test -cpuprofile=cpu.prof -bench=. ./pkg/scaffold/...
    go tool pprof -top cpu.prof

# Check for dependency vulnerabilities
audit:
    @echo "Auditing dependencies..."
    go list -json -deps ./... | nancy sleuth

# Initialize git hooks
init-hooks:
    @echo "Setting up git hooks..."
    @mkdir -p .git/hooks
    @echo '#!/bin/sh\njust pre-commit' > .git/hooks/pre-commit
    @chmod +x .git/hooks/pre-commit
    @echo "✅ Git hooks initialized"