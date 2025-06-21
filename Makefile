# Go Test and Coverage Makefile

.PHONY: test test-coverage test-coverage-html test-race test-bench profile-cpu profile-mem profile-mutex clean help

# Variables
COVERAGE_DIR=coverage
PROFILE_DIR=profiles
TEST_TIMEOUT=30s

# Create directories
$(COVERAGE_DIR):
	mkdir -p $(COVERAGE_DIR)

$(PROFILE_DIR):
	mkdir -p $(PROFILE_DIR)

# Run all tests
test:
	@echo "Running all tests..."
	go test -v ./...

# Run tests with coverage
test-coverage: $(COVERAGE_DIR)
	@echo "Running tests with coverage..."
	go test -v -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "Coverage report saved to $(COVERAGE_DIR)/coverage.out"

# Generate HTML coverage report
test-coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "HTML coverage report saved to $(COVERAGE_DIR)/coverage.html"
	@echo "Open the file in your browser to view the coverage report"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Run benchmark tests
test-bench: $(PROFILE_DIR)
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem -v ./...

# Run tests with CPU profiling
profile-cpu: $(PROFILE_DIR)
	@echo "Running tests with CPU profiling..."
	go test -cpuprofile=$(PROFILE_DIR)/cpu.prof -bench=. ./...
	@echo "CPU profile saved to $(PROFILE_DIR)/cpu.prof"
	@echo "To analyze: go tool pprof $(PROFILE_DIR)/cpu.prof"

# Run tests with memory profiling
profile-mem: $(PROFILE_DIR)
	@echo "Running tests with memory profiling..."
	go test -memprofile=$(PROFILE_DIR)/mem.prof -bench=. ./...
	@echo "Memory profile saved to $(PROFILE_DIR)/mem.prof"
	@echo "To analyze: go tool pprof $(PROFILE_DIR)/mem.prof"

# Run tests with mutex profiling
profile-mutex: $(PROFILE_DIR)
	@echo "Running tests with mutex profiling..."
	go test -mutexprofile=$(PROFILE_DIR)/mutex.prof -bench=. ./...
	@echo "Mutex profile saved to $(PROFILE_DIR)/mutex.prof"
	@echo "To analyze: go tool pprof $(PROFILE_DIR)/mutex.prof"

# Run all profiling together
profile-all: profile-cpu profile-mem profile-mutex
	@echo "All profiles generated in $(PROFILE_DIR)/"

# Run tests for a specific package
test-package:
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-package PKG=./path/to/package"; \
		exit 1; \
	fi
	go test -v $(PKG)

# Run tests with coverage for a specific package
test-package-coverage: $(COVERAGE_DIR)
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make test-package-coverage PKG=./path/to/package"; \
		exit 1; \
	fi
	go test -v -coverprofile=$(COVERAGE_DIR)/coverage-$(shell basename $(PKG)).out $(PKG)
	go tool cover -func=$(COVERAGE_DIR)/coverage-$(shell basename $(PKG)).out

# Run only unit tests (excluding integration tests)
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

# Run only integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -run Integration ./...

# Generate test coverage and upload to codecov (if token is set)
test-coverage-ci: test-coverage
	@if [ -n "$(CODECOV_TOKEN)" ]; then \
		echo "Uploading coverage to codecov..."; \
		bash <(curl -s https://codecov.io/bash) -f $(COVERAGE_DIR)/coverage.out; \
	else \
		echo "CODECOV_TOKEN not set, skipping upload"; \
	fi

# Run tests with timeout
test-timeout:
	@echo "Running tests with timeout $(TEST_TIMEOUT)..."
	go test -timeout $(TEST_TIMEOUT) -v ./...

# Analyze CPU profile (requires profile-cpu to be run first)
analyze-cpu-profile:
	@if [ ! -f "$(PROFILE_DIR)/cpu.prof" ]; then \
		echo "CPU profile not found. Run 'make profile-cpu' first."; \
		exit 1; \
	fi
	go tool pprof $(PROFILE_DIR)/cpu.prof

# Analyze memory profile (requires profile-mem to be run first)
analyze-mem-profile:
	@if [ ! -f "$(PROFILE_DIR)/mem.prof" ]; then \
		echo "Memory profile not found. Run 'make profile-mem' first."; \
		exit 1; \
	fi
	go tool pprof $(PROFILE_DIR)/mem.prof

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf $(COVERAGE_DIR) $(PROFILE_DIR)
	go clean -testcache

# Show help
help:
	@echo "Go Test and Coverage Commands:"
	@echo ""
	@echo "Basic Testing:"
	@echo "  test                    - Run all tests"
	@echo "  test-unit              - Run only unit tests"
	@echo "  test-integration       - Run only integration tests"
	@echo "  test-race              - Run tests with race detection"
	@echo "  test-timeout           - Run tests with timeout"
	@echo ""
	@echo "Coverage:"
	@echo "  test-coverage          - Run tests with coverage report"
	@echo "  test-coverage-html     - Generate HTML coverage report"
	@echo "  test-coverage-ci       - Coverage for CI with codecov upload"
	@echo ""
	@echo "Benchmarking:"
	@echo "  test-bench             - Run benchmark tests"
	@echo ""
	@echo "Profiling:"
	@echo "  profile-cpu            - Generate CPU profile"
	@echo "  profile-mem            - Generate memory profile"
	@echo "  profile-mutex          - Generate mutex profile"
	@echo "  profile-all            - Generate all profiles"
	@echo "  analyze-cpu-profile    - Analyze CPU profile interactively"
	@echo "  analyze-mem-profile    - Analyze memory profile interactively"
	@echo ""
	@echo "Package-specific:"
	@echo "  test-package PKG=path  - Test specific package"
	@echo "  test-package-coverage PKG=path - Coverage for specific package"
	@echo ""
	@echo "Utilities:"
	@echo "  clean                  - Clean generated files"
	@echo "  help                   - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make test-coverage-html"
	@echo "  make profile-cpu"
	@echo "  make test-package PKG=./application/services"

# Default target
.DEFAULT_GOAL := help
