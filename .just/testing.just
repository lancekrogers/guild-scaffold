# Testing recipes for guild-scaffold

# Run unit tests with specific package
test-pkg PKG:
    @echo "Testing package: {{PKG}}"
    go test -v -race -cover ./pkg/{{PKG}}/...

# Run unit tests with verbose output
test-verbose:
    @echo "Running tests with verbose output..."
    go test -v -race ./pkg/... -count=1

# Run tests matching a pattern
test-pattern PATTERN:
    @echo "Running tests matching: {{PATTERN}}"
    go test -v ./pkg/... -run "{{PATTERN}}"

# Integration test with specific fixture
test-fixture FIXTURE:
    @echo "Testing with fixture: {{FIXTURE}}"
    go test -v -tags=integration ./tests/integration/... \
        -run "Test.*{{FIXTURE}}.*" \
        -timeout 5m

# Run container tests with custom image
test-container-custom IMAGE='alpine:latest':
    @echo "Running tests with image: {{IMAGE}}"
    IMAGE={{IMAGE}} go test -v -tags=integration ./tests/integration/... -timeout 10m

# Test with race detection disabled (faster)
test-fast:
    @echo "Running fast tests (no race detection)..."
    go test ./pkg/...

# Test with specific timeout
test-timeout TIMEOUT='30s':
    @echo "Running tests with timeout: {{TIMEOUT}}"
    go test -v -timeout {{TIMEOUT}} ./pkg/...

# Run tests and generate JSON output
test-json:
    @echo "Running tests with JSON output..."
    go test -json ./pkg/... | tee test-results.json

# Run only the last failed test
test-failed:
    @echo "Re-running failed tests..."
    go test -v ./pkg/... -run $(go test -list ./pkg/... | grep FAIL | head -1)

# Fuzz testing
fuzz TARGET='FuzzParser' TIME='10s':
    @echo "Running fuzz test: {{TARGET}} for {{TIME}}"
    go test -fuzz={{TARGET}} -fuzztime={{TIME}} ./pkg/scaffold/...

# Memory profiling during tests
test-mem-profile:
    @echo "Running tests with memory profiling..."
    go test -memprofile=mem.prof -bench=. ./pkg/scaffold/...
    go tool pprof -http=:8080 mem.prof

# CPU profiling during tests
test-cpu-profile:
    @echo "Running tests with CPU profiling..."
    go test -cpuprofile=cpu.prof -bench=. ./pkg/scaffold/...
    go tool pprof -http=:8080 cpu.prof