# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Development
```bash
make build      # Build the binary
make test       # Run tests with race detection
make lint       # Format and vet the code
make fmt        # Format code only  
make all        # Run all checks and build
```

### Testing Examples
```bash
# Test with the tool itself (using example tests)
go test -json ./example | ./gotestshow

# CI mode testing
go test -json ./example | ./gotestshow -ci

# Timing mode testing  
go test -json ./example | ./gotestshow -timing -threshold=500ms
```

## Architecture Overview

gotestshow is a CLI tool that transforms `go test -json` output into human-readable real-time display.

### Core Components

- **main.go**: Entry point and CLI flag parsing
- **runner.go**: Application lifecycle management with signal handling and concurrent progress updates
- **processor.go**: Event processing and state management (implements EventProcessor interface)
- **display.go**: Output formatting and display modes (implements Display interface)
- **types.go**: Core data structures (TestEvent, TestResult, PackageState)

### Key Design Patterns

1. **Interface-driven design**: EventProcessor and Display interfaces enable testing and extensibility
2. **Concurrent processing**: Progress updates run in separate goroutine with 100ms intervals
3. **State management**: Thread-safe state tracking using sync.RWMutex
4. **Multiple display modes**: Normal (animated), Timing (slow tests only), CI (no ANSI)

### Package Path Processing

The tool extracts relative file paths from full package paths using heuristics to handle common patterns like `github.com/user/repo/path` → `path/file.go:line`.

### Test Structure

- Unit tests for each component (`*_test.go`)
- Integration tests (`integration_test.go`) 
- E2E tests (`e2e_test.go`)
- Example test cases in `example/` directory with intentional failures and varying execution times

## Development Notes

- Zero external dependencies (stdlib only)
- Requires Go 1.24.3+
- Uses `go test -race` for concurrency testing
- CI runs on GitHub Actions with `make test`
- The tool processes JSON events line-by-line from stdin
- Graceful shutdown with context cancellation and signal handling