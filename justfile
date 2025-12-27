#!/usr/bin/env just --justfile
# Guild Scaffold - Development Commands
# https://github.com/casey/just

set dotenv-load := true

# Configuration
binary_name := "scaffold"
bin_dir := "bin"
gobin := env_var_or_default("GOBIN", `go env GOPATH` + "/bin")

# Modules
[doc('Testing (unit, integration, coverage)')]
mod test '.justfiles/testing.just'

[doc('Docker and container operations')]
mod docker '.justfiles/docker.just'

[doc('Development utilities')]
mod dev '.justfiles/development.just'

[private]
default:
    #!/usr/bin/env bash
    echo "Guild Scaffold - Project Scaffolding Tool"
    echo ""
    just --list --unsorted

# Build scaffold binary
build:
    @echo "Building scaffold..."
    @mkdir -p {{bin_dir}}
    go build -v -o {{bin_dir}}/{{binary_name}} ./cmd/scaffold

# Build scaffold binary (fast, no vet)
build-only:
    @echo "Building scaffold (fast)..."
    @mkdir -p {{bin_dir}}
    go build -o {{bin_dir}}/{{binary_name}} ./cmd/scaffold

# Build Linux binary for containers
build-linux:
    @echo "Building Linux binary..."
    @mkdir -p {{bin_dir}}/linux
    GOOS=linux GOARCH=amd64 go build -o {{bin_dir}}/linux/{{binary_name}} ./cmd/scaffold

# Format Go code
fmt:
    go fmt ./...

# Run go vet
vet:
    go vet ./...

# Run formatting and vetting
lint: fmt vet
    @echo "✅ Linting complete"

# Clean build artifacts
clean:
    @echo "Cleaning..."
    rm -rf {{bin_dir}}
    rm -rf tests/integration/tmp
    go clean -testcache
    @echo "✅ Clean complete"

# Update and tidy dependencies
deps:
    go get -u ./...
    go mod tidy

# Install scaffold to $GOBIN
install: build
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Installing scaffold..."
    mkdir -p {{gobin}}
    cp {{bin_dir}}/{{binary_name}} {{gobin}}/{{binary_name}}
    if [[ "$(uname)" == "Darwin" ]]; then
        echo "Signing binary for macOS..."
        codesign --force --sign - {{gobin}}/{{binary_name}} 2>/dev/null || \
        echo "Warning: Could not sign binary (non-fatal)"
    fi
    echo "✅ scaffold installed to {{gobin}}/{{binary_name}}"

# Uninstall scaffold from $GOBIN
uninstall:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Uninstalling scaffold..."
    if [ -f {{gobin}}/{{binary_name}} ]; then
        rm {{gobin}}/{{binary_name}}
        echo "scaffold uninstalled from {{gobin}}"
    else
        echo "scaffold not found in {{gobin}}"
    fi

# Quick check (build + vet)
check: vet build
    @echo "✅ All checks passed"
