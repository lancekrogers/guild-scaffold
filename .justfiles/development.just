# Development workflow recipes for guild-scaffold

# Show available dev commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Start development with file watching
[no-cd]
watch:
    @echo "Starting development mode..."
    watchexec -e go -r -- go run ./cmd/scaffold

# Run continuous testing
[no-cd]
watch-test:
    @echo "Starting continuous testing..."
    watchexec -e go -- just test unit-fast

# Validate all example templates
[no-cd]
validate-examples:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Validating example templates..."
    for f in examples/*.yaml; do
        if [[ -f "$f" ]]; then
            echo "Validating $f..."
            go run ./cmd/scaffold validate --template "$f"
        fi
    done
    echo "✅ All examples valid"

# Check for TODO/FIXME comments
[no-cd]
check-todos:
    @echo "Checking for TODO/FIXME comments..."
    @grep -r "TODO\|FIXME" --include="*.go" . | head -20 || echo "No TODOs found"

# Generate mocks for testing
[no-cd]
generate-mocks:
    @echo "Generating mocks..."
    go generate ./...

# Run pre-commit checks
[no-cd]
pre-commit:
    @echo "Running pre-commit checks..."
    just fmt
    just vet
    just test unit-fast
    @echo "✅ Pre-commit checks passed"

# Show code statistics
[no-cd]
stats:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Code statistics:"
    echo ""
    echo "Lines of Go code:"
    find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
    echo ""
    echo "Number of Go files:"
    find . -name "*.go" -not -path "./vendor/*" | wc -l
    echo ""
    echo "Package breakdown:"
    scc --by-file --no-cocomo --no-complexity --include-ext go . 2>/dev/null | head -30 || \
        echo "(install scc for detailed breakdown)"

# Debug a specific test with delve
[no-cd]
debug-test TEST:
    @echo "Debugging test: {{TEST}}"
    dlv test ./pkg/... -- -test.run "{{TEST}}"

# Profile memory usage
[no-cd]
profile-mem:
    @echo "Profiling memory usage..."
    go test -memprofile=mem.prof -bench=. ./pkg/scaffold/...
    go tool pprof -top mem.prof

# Profile CPU usage
[no-cd]
profile-cpu:
    @echo "Profiling CPU usage..."
    go test -cpuprofile=cpu.prof -bench=. ./pkg/scaffold/...
    go tool pprof -top cpu.prof

# Initialize git hooks
[no-cd]
init-hooks:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Setting up git hooks..."
    mkdir -p ".git/hooks"
    printf '#!/bin/sh\njust dev pre-commit\n' > ".git/hooks/pre-commit"
    chmod +x ".git/hooks/pre-commit"
    echo "✅ Git hooks initialized"

# Show registry configuration
[no-cd]
show-registry:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Global registry (~/.guild/scaffold.yaml):"
    if [[ -f ~/.guild/scaffold.yaml ]]; then
        cat ~/.guild/scaffold.yaml
    else
        echo "  (not found)"
    fi
    echo ""
    echo "Project registry (.campaign/scaffold.yaml):"
    if [[ -f .campaign/scaffold.yaml ]]; then
        cat .campaign/scaffold.yaml
    else
        echo "  (not found)"
    fi
