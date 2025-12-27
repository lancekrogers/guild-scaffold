# Docker/Container recipes for guild-scaffold

# Show available docker commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Build a minimal test container
[no-cd]
build-test:
    @echo "Building test container..."
    docker build -f Dockerfile.test -t scaffold-test:latest .

# Run scaffold in an isolated Alpine container
[no-cd]
run-alpine TEMPLATE OUTPUT='output':
    @echo "Running in Alpine container..."
    docker run --rm -it \
        -v $(pwd):/workspace \
        -w /workspace \
        alpine:latest \
        sh -c "apk add --no-cache go && go run ./cmd/scaffold init test --template {{TEMPLATE}} --output /{{OUTPUT}}"

# Test in Ubuntu container
[no-cd]
run-ubuntu TEMPLATE OUTPUT='output':
    @echo "Running in Ubuntu container..."
    docker run --rm -it \
        -v $(pwd):/workspace \
        -w /workspace \
        ubuntu:latest \
        sh -c "apt-get update && apt-get install -y golang && go run ./cmd/scaffold init test --template {{TEMPLATE}} --output /{{OUTPUT}}"

# Clean up test containers and images
[no-cd]
clean:
    @echo "Cleaning up test containers..."
    docker ps -a | grep scaffold-test | awk '{print $1}' | xargs -r docker rm -f
    docker images | grep scaffold-test | awk '{print $3}' | xargs -r docker rmi -f

# Run scaffold in different Go versions
[no-cd]
test-go-versions:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Testing with multiple Go versions..."
    for version in 1.21 1.22 1.23; do
        echo "Testing with Go $version"
        docker run --rm \
            -v $(pwd):/workspace \
            -w /workspace \
            golang:$version-alpine \
            go test -v ./pkg/...
    done

# Create a persistent test environment container
[no-cd]
create-env:
    @echo "Creating test environment container..."
    docker run -d \
        --name scaffold-test-env \
        -v $(pwd):/workspace \
        -w /workspace \
        golang:1.22-alpine \
        sh -c "while true; do sleep 3600; done"
    @echo "Container 'scaffold-test-env' created. Use 'docker exec -it scaffold-test-env sh' to enter."

# Stop and remove test environment
[no-cd]
remove-env:
    @echo "Removing test environment..."
    docker stop scaffold-test-env || true
    docker rm scaffold-test-env || true

# Check if Colima is running (required for integration tests)
[no-cd]
check-colima:
    #!/usr/bin/env bash
    set -euo pipefail
    if ! colima status &>/dev/null; then
        echo "❌ Colima is not running. Start with: colima start"
        exit 1
    fi
    echo "✅ Colima is running"
