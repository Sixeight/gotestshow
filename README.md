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
  - Test name and file location with package path (e.g., `example/math_test.go:47`)
  - Execution time and error messages
  - Clear distinction between files in different packages
- **Skipped tests**: Update counter only

### 🔧 Multiple Display Modes

- **Normal Mode**: Real-time animated progress with colors
- **Timing Mode**: Show only slow tests and failures with execution times
- **CI Mode**: Clean output without escape sequences for CI/CD pipelines

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

### Timing Mode

Show only slow tests and failures with execution times:

```bash
go test -json ./... | gotestshow -timing
```

Customize the threshold for slow tests:

```bash
go test -json ./... | gotestshow -timing -threshold=1s
```

### CI Mode

For CI/CD pipelines - clean output without escape sequences, colors, or animations:

```bash
go test -json ./... | gotestshow -ci
```

CI mode features:
- No ANSI escape sequences (safe for log files)
- No real-time progress updates
- Only shows failed tests during execution
- Includes detailed failure summary at the end
- Simple, parseable output format

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-help` | Show help message | - |
| `-timing` | Enable timing mode to show only slow tests and failures | `false` |
| `-threshold` | Threshold for slow tests (e.g., 1s, 500ms, 1.5s) | `500ms` |
| `-ci` | Enable CI mode - no escape sequences, only show failures and summary | `false` |

## Example Output

### Normal Mode

```
⠸ Running: 3 | ✓ Passed: 12 | ✗ Failed: 2 | ⚡ Skipped: 1 | ⏱ 3.2s

✗ FAIL TestMultiply/negative_numbers [example/math_test.go:47] (0.00s)

        math_test.go:48: Multiply(-2, 5) = -10, want -11
        

==================================================
📊 Failed Tests Summary
==================================================

✗ FAIL github.com/example/math (1.23s)
  Tests: 15 | Passed: 12 | Failed: 2 | Skipped: 1

    ✗ TestMultiply/negative_numbers [example/math_test.go:47] (0.00s)
    ✗ TestDivide/divide_by_zero [example/math_test.go:72] (0.00s)

--------------------------------------------------
Total: 15 tests | ✓ Passed: 12 | ✗ Failed: 2 | ⚡ Skipped: 1 | ⏱ 3.24s

❌ Tests failed!
```

### CI Mode

```
FAIL TestMultiply [example/math_test.go:47] (0.40s)

            math_test.go:47: Multiply(4, 5) = 9, want 20
        --- FAIL: TestMultiply (0.40s)

FAIL TestDivide/divide_by_zero [example/math_test.go:68] (0.30s)

            math_test.go:68: expected error for divide by zero, got nil
        --- FAIL: TestDivide/divide_by_zero (0.30s)

==================================================
Failed Tests Summary
==================================================

FAIL github.com/Sixeight/gotestshow/example (5.68s)
  Tests: 17 | Passed: 10 | Failed: 3 | Skipped: 1

    FAIL TestMultiply [example/math_test.go:47] (0.40s)
    FAIL TestDivide/divide_by_zero [example/math_test.go:68] (0.30s)

--------------------------------------------------

Total: 17 tests | Passed: 10 | Failed: 3 | Skipped: 1 | Time: 5.93s

Tests failed!
```

## Why Use gotestshow?

Standard `go test` output can be difficult to read, especially with large test suites or in CI environments. gotestshow:

1. **Reduces noise**: Hides details of passing tests to focus on failures
2. **Immediate feedback**: Failed tests are displayed instantly
3. **Visual progress**: Current state is clear at a glance, even with large test suites
4. **Package context**: File locations include package paths to distinguish between files in different packages
5. **Multiple modes**: 
   - **Normal**: Real-time progress with colors and animations
   - **Timing**: Focus on slow tests and performance analysis
   - **CI**: Clean output perfect for CI/CD pipelines
6. **CI/CD friendly**: Reads from stdin, making it easy to integrate into existing workflows

## Related Projects

- [gotestfmt](https://github.com/GoTestTools/gotestfmt) - Format `go test` output for improved readability.
- [gotestsum](https://github.com/gotestyourself/gotestsum) - Summarize `go test` results.
- [richgo](https://github.com/kyoh86/richgo) - Colored test output enhancement.


## Development

### Build

```bash
make build
# or
go build -o gotestshow .
```

### Run Tests

```bash
make test
# or
go test -v -race .
```

### Example Test Runs

```bash
# Normal mode with example tests
go test -json ./example | ./gotestshow

# CI mode
go test -json ./example | ./gotestshow -ci

# Timing mode
go test -json ./example | ./gotestshow -timing -threshold=500ms
```

## License

MIT License

## Author

Sixeight

---

*Made to improve the Go testing experience* ✨

