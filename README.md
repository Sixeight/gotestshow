# gotestshow

A CLI tool that displays `go test -json` output in a real-time, human-friendly format

![Go Version](https://img.shields.io/badge/Go-1.24.3%2B-blue.svg)

## Features

### 🚀 Real-time Progress Display
- Animated spinner to visualize execution state
- Real-time updates of running, passed, failed, and skipped test counts
- Dynamic elapsed time display

### 🎯 Smart Result Display
- **Passed tests**: Update progress counter only (reduces noise)
- **Failed tests**: Show detailed information immediately
  - Test name and file location (e.g., `math_test.go:47`)
  - Execution time and error messages
- **Skipped tests**: Update counter only

### 📦 Advanced Features
- Proper handling of subtests (`t.Run()`)
- Support for parallel tests (`t.Parallel()`)
- Package-level failure detection (build errors, syntax errors, etc.)
- Optimized screen updates to prevent flickering

## Installation

```bash
go install github.com/Sixeight/gotestshow@latest
```

## Usage

### Basic Usage

```bash
go test -json ./... | gotestshow
```

### Test Specific Packages

```bash
go test -json ./pkg/... | gotestshow
```

### Run with Verbose Mode

```bash
go test -v -json ./... | gotestshow
```

## Example Output

```
⠸ Running: 3 | ✓ Passed: 12 | ✗ Failed: 2 | ⚡ Skipped: 1 | ⏱ 3.2s

✗ FAIL TestMultiply/negative_numbers [math_test.go:47] (0.00s)

        math_test.go:48: Multiply(-2, 5) = -10, want -11
        

==================================================
📊 Test Results Summary
==================================================

✗ FAIL github.com/example/math (1.23s)
  Tests: 15 | Passed: 12 | Failed: 2 | Skipped: 1

    ✗ TestMultiply/negative_numbers [math_test.go:47] (0.00s)
    ✗ TestDivide/divide_by_zero [math_test.go:72] (0.00s)

--------------------------------------------------
Total: 15 tests | ✓ Passed: 12 | ✗ Failed: 2 | ⚡ Skipped: 1 | ⏱ 3.24s

❌ Tests failed!
```

## Why Use gotestshow?

Standard `go test` output can be difficult to read, especially with large test suites or in CI environments. gotestshow:

1. **Reduces noise**: Hides details of passing tests to focus on failures
2. **Immediate feedback**: Failed tests are displayed instantly
3. **Visual progress**: Current state is clear at a glance, even with large test suites
4. **CI/CD friendly**: Reads from stdin, making it easy to integrate into existing workflows

## Related Projects

- [gotestfmt](https://github.com/GoTestTools/gotestfmt) - Format `go test` output for improved readability.
- [gotestsum](https://github.com/gotestyourself/gotestsum) - Summarize `go test` results.
- [richgo](https://github.com/kyoh86/richgo) - Colored test output enhancement.


## Development

### Build

```bash
go build -o gotestshow main.go
```

### Example Test Run

```bash
# Run tests in the example directory
go test -json ./example/... | ./gotestshow
```

## License

MIT License

## Author

Sixeight

---

*Made to improve the Go testing experience* ✨