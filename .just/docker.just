# Docker/Container recipes for guild-scaffold

# Build a minimal test container
docker-build-test:
    @echo "Building test container..."
    docker build -f Dockerfile.test -t scaffold-test:latest .

# Run scaffold in an isolated Alpine container
docker-run-alpine TEMPLATE OUTPUT='output':
    @echo "Running in Alpine container..."
    docker run --rm -it \
        -v $(pwd):/workspace \
        -w /workspace \
        alpine:latest \
        sh -c "apk add --no-cache go && go run ./cmd/scaffold init test --template {{TEMPLATE}} --output /{{OUTPUT}}"

# Test in Ubuntu container
docker-run-ubuntu TEMPLATE OUTPUT='output':
    @echo "Running in Ubuntu container..."
    docker run --rm -it \
        -v $(pwd):/workspace \
        -w /workspace \
        ubuntu:latest \
        sh -c "apt-get update && apt-get install -y golang && go run ./cmd/scaffold init test --template {{TEMPLATE}} --output /{{OUTPUT}}"

# Run integration tests in container
docker-test-integration:
    @echo "Running integration tests in container..."
    docker run --rm \
        -v $(pwd):/workspace \
        -v /var/run/docker.sock:/var/run/docker.sock \
        -w /workspace \
        golang:1.21-alpine \
        sh -c "apk add --no-cache docker && go test -v -tags=integration ./tests/integration/..."

# Clean up test containers and images
docker-clean:
    @echo "Cleaning up test containers..."
    docker ps -a | grep scaffold-test | awk '{print $1}' | xargs -r docker rm -f
    docker images | grep scaffold-test | awk '{print $3}' | xargs -r docker rmi -f

# Run scaffold in different Go versions
docker-test-go-versions:
    @echo "Testing with multiple Go versions..."
    @for version in 1.20 1.21 1.22; do \
        echo "Testing with Go $$version"; \
        docker run --rm \
            -v $(pwd):/workspace \
            -w /workspace \
            golang:$$version-alpine \
            go test -v ./pkg/...; \
    done

# Create a test environment container
docker-create-env:
    @echo "Creating test environment container..."
    docker run -d \
        --name scaffold-test-env \
        -v $(pwd):/workspace \
        -w /workspace \
        golang:1.21-alpine \
        sh -c "while true; do sleep 3600; done"
    @echo "Container 'scaffold-test-env' created. Use 'docker exec -it scaffold-test-env sh' to enter."

# Stop and remove test environment
docker-remove-env:
    @echo "Removing test environment..."
    docker stop scaffold-test-env || true
    docker rm scaffold-test-env || true

# Run tests with Docker Compose
docker-compose-test:
    @echo "Running tests with Docker Compose..."
    docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
    docker-compose -f docker-compose.test.yml down