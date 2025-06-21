#!/bin/bash

# Go Test and Profiling Script
# This script provides convenient commands for running tests and generating profiles

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Directories
COVERAGE_DIR="coverage"
PROFILE_DIR="profiles"

# Create directories if they don't exist
mkdir -p "$COVERAGE_DIR" "$PROFILE_DIR"

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to run tests with coverage
run_coverage_tests() {
    print_info "Running tests with coverage..."
    
    if go test -v -coverprofile="$COVERAGE_DIR/coverage.out" ./...; then
        print_success "Tests completed successfully"
        
        # Generate coverage statistics
        print_info "Coverage Statistics:"
        go tool cover -func="$COVERAGE_DIR/coverage.out"
        
        # Generate HTML report
        go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"
        print_success "HTML coverage report generated: $COVERAGE_DIR/coverage.html"
        
        # Open HTML report if on macOS
        if [[ "$OSTYPE" == "darwin"* ]]; then
            read -p "Open coverage report in browser? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                open "$COVERAGE_DIR/coverage.html"
            fi
        fi
    else
        print_error "Tests failed"
        exit 1
    fi
}

# Function to run CPU profiling
run_cpu_profile() {
    print_info "Running CPU profiling..."
    
    if go test -cpuprofile="$PROFILE_DIR/cpu.prof" -bench=. ./...; then
        print_success "CPU profile generated: $PROFILE_DIR/cpu.prof"
        print_info "To analyze: go tool pprof $PROFILE_DIR/cpu.prof"
        
        # Offer to analyze immediately
        read -p "Analyze CPU profile now? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            go tool pprof "$PROFILE_DIR/cpu.prof"
        fi
    else
        print_error "CPU profiling failed"
        exit 1
    fi
}

# Function to run memory profiling
run_memory_profile() {
    print_info "Running memory profiling..."
    
    if go test -memprofile="$PROFILE_DIR/mem.prof" -bench=. ./...; then
        print_success "Memory profile generated: $PROFILE_DIR/mem.prof"
        print_info "To analyze: go tool pprof $PROFILE_DIR/mem.prof"
        
        # Offer to analyze immediately
        read -p "Analyze memory profile now? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            go tool pprof "$PROFILE_DIR/mem.prof"
        fi
    else
        print_error "Memory profiling failed"
        exit 1
    fi
}

# Function to run all profiles
run_all_profiles() {
    print_info "Running all profiling tests..."
    
    # CPU Profile
    print_info "Generating CPU profile..."
    go test -cpuprofile="$PROFILE_DIR/cpu.prof" -bench=. ./... > /dev/null 2>&1
    
    # Memory Profile
    print_info "Generating memory profile..."
    go test -memprofile="$PROFILE_DIR/mem.prof" -bench=. ./... > /dev/null 2>&1
    
    # Mutex Profile
    print_info "Generating mutex profile..."
    go test -mutexprofile="$PROFILE_DIR/mutex.prof" -bench=. ./... > /dev/null 2>&1
    
    print_success "All profiles generated in $PROFILE_DIR/"
    ls -la "$PROFILE_DIR/"
}

# Function to run benchmarks
run_benchmarks() {
    print_info "Running benchmark tests..."
    
    if go test -bench=. -benchmem -v ./...; then
        print_success "Benchmarks completed"
    else
        print_error "Benchmarks failed"
        exit 1
    fi
}

# Function to run race detection
run_race_tests() {
    print_info "Running tests with race detection..."
    
    if go test -race -v ./...; then
        print_success "Race detection tests passed"
    else
        print_error "Race conditions detected"
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  coverage    - Run tests with coverage report and HTML output"
    echo "  cpu         - Run CPU profiling"
    echo "  memory      - Run memory profiling"
    echo "  profiles    - Run all profiling (CPU, memory, mutex)"
    echo "  bench       - Run benchmark tests"
    echo "  race        - Run tests with race detection"
    echo "  all         - Run all tests and generate all reports"
    echo "  clean       - Clean generated files"
    echo "  help        - Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 coverage"
    echo "  $0 cpu"
    echo "  $0 all"
}

# Function to clean generated files
clean_files() {
    print_info "Cleaning generated files..."
    rm -rf "$COVERAGE_DIR" "$PROFILE_DIR"
    go clean -testcache
    print_success "Cleaned successfully"
}

# Function to run all tests and profiles
run_all() {
    print_info "Running comprehensive test suite..."
    
    # Regular tests
    print_info "Step 1/5: Running basic tests..."
    go test -v ./...
    
    # Coverage tests
    print_info "Step 2/5: Running coverage tests..."
    run_coverage_tests
    
    # Race detection
    print_info "Step 3/5: Running race detection..."
    run_race_tests
    
    # Benchmarks
    print_info "Step 4/5: Running benchmarks..."
    run_benchmarks
    
    # All profiles
    print_info "Step 5/5: Generating profiles..."
    run_all_profiles
    
    print_success "Comprehensive test suite completed!"
    print_info "Coverage report: $COVERAGE_DIR/coverage.html"
    print_info "Profiles available in: $PROFILE_DIR/"
}

# Main script logic
case "${1:-help}" in
    "coverage")
        run_coverage_tests
        ;;
    "cpu")
        run_cpu_profile
        ;;
    "memory")
        run_memory_profile
        ;;
    "profiles")
        run_all_profiles
        ;;
    "bench")
        run_benchmarks
        ;;
    "race")
        run_race_tests
        ;;
    "all")
        run_all
        ;;
    "clean")
        clean_files
        ;;
    "help"|*)
        show_usage
        ;;
esac
