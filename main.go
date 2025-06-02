// ABOUTME: gotestshow is a CLI tool that parses and formats the output of `go test -json`.
// ABOUTME: It receives JSON-formatted test results from stdin and displays them in a readable format in real-time.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// TestEvent represents a single test event from go test -json
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
}

// TestResult holds the summary of a test
type TestResult struct {
	Package    string
	Test       string
	Passed     bool
	Skipped    bool
	Failed     bool
	Elapsed    float64
	Output     []string
	Started    bool
	Location   string // File name and line number (e.g., "math_test.go:47")
	HasSubtest bool   // Whether this test has subtests
}

// PackageState tracks the state of tests in a package
type PackageState struct {
	Name                 string
	Total                int
	Passed               int
	Failed               int
	Skipped              int
	Running              int
	Elapsed              float64
	Output               []string // Store package-level output
	IndividualTestFailed int      // Number of individual test failures
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	clearLine   = "\r"
)

var (
	spinnerChars      = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	dotsChars         = []string{"   ", ".  ", ".. ", "..."}
	lastDisplayLength int       // Record the length of the previously displayed line
	startTime         time.Time // Record the execution start time
	hasTestsStarted   bool      // Track if any tests have started running
)

// exitWithCursorRestore ensures cursor is shown before program exits
func exitWithCursorRestore(code int) {
	fmt.Print("\033[?25h") // Show the cursor again
	os.Exit(code)
}

func getSpinner() string {
	// Switch animation frames at 100ms intervals
	now := time.Now().UnixNano() / int64(time.Millisecond)
	index := (now / 100) % int64(len(spinnerChars))
	return spinnerChars[index]
}

func getDots() string {
	// Switch dot animation frames at 500ms intervals
	now := time.Now().UnixNano() / int64(time.Millisecond)
	index := (now / 500) % int64(len(dotsChars))
	return dotsChars[index]
}

// smartDisplayLine compares the length of display content and updates the line appropriately
// Only clears the line when new content is shorter, otherwise simply overwrites to prevent flickering
func smartDisplayLine(content string) {
	currentLength := len(content)

	if currentLength < lastDisplayLength {
		// When new content is shorter: \r (return to beginning of line) + \033[K (clear to end of line) then display
		fmt.Print("\r\033[K")
		fmt.Print(content)
	} else {
		// When new content is same length or longer: overwrite with \r (return to beginning of line)
		fmt.Print("\r")
		fmt.Print(content)
	}

	lastDisplayLength = currentLength
}

func showHelp() {
	fmt.Println("gotestshow - A real-time formatter for `go test -json` output")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go test -json ./... | gotestshow")
	fmt.Println()
	fmt.Println("Description:")
	fmt.Println("  gotestshow reads JSON-formatted test output from stdin and displays")
	fmt.Println("  it in a human-readable format with real-time progress updates.")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  • Real-time progress with animated spinner")
	fmt.Println("  • Shows only failed test details to reduce noise")
	fmt.Println("  • Smart handling of subtests and parallel tests")
	fmt.Println("  • Package-level failure detection (build errors, syntax errors)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Test all packages")
	fmt.Println("  go test -json ./... | gotestshow")
	fmt.Println()
	fmt.Println("  # Test specific package")
	fmt.Println("  go test -json ./pkg/... | gotestshow")
	fmt.Println()
	fmt.Println("  # Run specific test")
	fmt.Println("  go test -json -run TestName ./... | gotestshow")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -help  Show this help message")
}

func main() {
	help := flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		showHelp()
		exitWithCursorRestore(0)
	}

	// Check if stdin has input
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// No input from pipe
		showHelp()
		exitWithCursorRestore(0)
	}

	results := make(map[string]*TestResult)
	packages := make(map[string]*PackageState)
	var mu sync.RWMutex

	// Record the execution start time
	startTime = time.Now()

	// Hide the cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show the cursor again when the program exits normally

	// Ensure the cursor is shown again through signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		exitWithCursorRestore(1)
	}()

	// Initial display will be handled by the periodic update goroutine

	// Goroutine for periodic screen updates
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mu.RLock()
				// Always display the progress bar (during compilation or test execution)
				displayProgress(packages)
				mu.RUnlock()
			}
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var event TestEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}

		mu.Lock()
		// Initialize the package
		if _, exists := packages[event.Package]; !exists && event.Package != "" {
			packages[event.Package] = &PackageState{
				Name: event.Package,
			}
		}

		// Track test results
		if event.Test != "" {
			key := fmt.Sprintf("%s/%s", event.Package, event.Test)
			if _, exists := results[key]; !exists {
				results[key] = &TestResult{
					Package: event.Package,
					Test:    event.Test,
					Output:  []string{},
				}
			}
			result := results[key]
			pkg := packages[event.Package]

			// In case of subtest, mark the parent test
			if strings.Contains(event.Test, "/") {
				parentTestName := strings.Split(event.Test, "/")[0]
				parentKey := fmt.Sprintf("%s/%s", event.Package, parentTestName)
				if parentResult, exists := results[parentKey]; exists {
					parentResult.HasSubtest = true
				}
			}

			switch event.Action {
			case "run":
				result.Started = true
				pkg.Running++
				hasTestsStarted = true // Mark that tests have started

				// Don't count parent tests that have subtests
				// However, since subtests don't exist yet at the time of the run event
				// We need to count everything first and adjust later
				pkg.Total++
			case "output":
				result.Output = append(result.Output, event.Output)
				// Extract filename and line number
				if result.Location == "" {
					location := extractFileLocation(event.Output)
					if location != "" {
						result.Location = location
					}
				}
			case "pass":
				result.Passed = true
				result.Elapsed = event.Elapsed
				pkg.Running--

				// Don't count parent tests that have subtests
				// For parent tests, since HasSubtest is set before subtest determination
				// Determine by checking if the test name contains subtests
				isParentWithSubtests := false
				if !strings.Contains(event.Test, "/") {
					// For parent tests, check if there are subtests in the same package
					for existingKey := range results {
						if strings.HasPrefix(existingKey, fmt.Sprintf("%s/%s/", event.Package, event.Test)) {
							isParentWithSubtests = true
							break
						}
					}
				}

				if !isParentWithSubtests {
					pkg.Passed++
				}
				displayTestResult(result, true)
			case "fail":
				result.Failed = true
				result.Elapsed = event.Elapsed
				pkg.Running--

				// Don't count parent tests that have subtests
				// For parent tests, since HasSubtest is set before subtest determination
				// Determine by checking if the test name contains subtests
				isParentWithSubtests := false
				if !strings.Contains(event.Test, "/") {
					// For parent tests, check if there are subtests in the same package
					for existingKey := range results {
						if strings.HasPrefix(existingKey, fmt.Sprintf("%s/%s/", event.Package, event.Test)) {
							isParentWithSubtests = true
							break
						}
					}
				}

				if !isParentWithSubtests {
					pkg.Failed++
					pkg.IndividualTestFailed++ // Count the number of individual test failures
				}
				displayTestResult(result, false)
			case "skip":
				result.Skipped = true
				result.Elapsed = event.Elapsed
				pkg.Running--

				// Don't count parent tests that have subtests
				// For parent tests, since HasSubtest is set before subtest determination
				// Determine by checking if the test name contains subtests
				isParentWithSubtests := false
				if !strings.Contains(event.Test, "/") {
					// For parent tests, check if there are subtests in the same package
					for existingKey := range results {
						if strings.HasPrefix(existingKey, fmt.Sprintf("%s/%s/", event.Package, event.Test)) {
							isParentWithSubtests = true
							break
						}
					}
				}

				if !isParentWithSubtests {
					pkg.Skipped++
				}
				displayTestResult(result, true)
			}
		} else if event.Package != "" {
			// Package-level events
			pkg := packages[event.Package]
			switch event.Action {
			case "output":
				// Accumulate package-level output
				pkg.Output = append(pkg.Output, event.Output)
			case "pass":
				pkg.Elapsed = event.Elapsed
			case "fail":
				pkg.Elapsed = event.Elapsed
				// Record package-level failure (using accumulated output)
				key := fmt.Sprintf("%s/[PACKAGE]", event.Package)
				results[key] = &TestResult{
					Package: event.Package,
					Test:    "[PACKAGE]",
					Failed:  true,
					Elapsed: event.Elapsed,
					Output:  pkg.Output, // Use accumulated output
				}

				// In case of Package Fail, update statistics for progress display
				if shouldDisplayPackageFailure(pkg) {
					pkg.Total++  // Include Package Fail in the test count
					pkg.Failed++ // Increase the Failed count
					displayPackageFailure(event.Package, pkg.Output)
				}
			}
		}
		mu.Unlock()
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		exitWithCursorRestore(1)
	}

	// Display final results
	fmt.Println()
	displayFinalResults(packages, results)
}

func displayProgress(packages map[string]*PackageState) {
	animation := getSpinner()
	elapsed := time.Since(startTime)

	// Show simple initialization message until first test starts
	if !hasTestsStarted {
		dots := getDots()
		content := fmt.Sprintf("%s%s Initializing%s%s", colorBlue, animation, dots, colorReset)
		smartDisplayLine(content)
		return
	}

	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	totalRunning := 0

	for _, pkg := range packages {
		totalTests += pkg.Total
		totalPassed += pkg.Passed
		totalFailed += pkg.Failed
		totalSkipped += pkg.Skipped
		totalRunning += pkg.Running
	}

	// Display detailed progress bar
	content := fmt.Sprintf("%s%s Running: %d%s | %s✓ Passed: %d%s | %s✗ Failed: %d%s | %s⚡ Skipped: %d%s | %s⏱ %.1fs%s",
		colorBlue, animation, totalRunning, colorReset,
		colorGreen, totalPassed, colorReset,
		colorRed, totalFailed, colorReset,
		colorYellow, totalSkipped, colorReset,
		colorGray, elapsed.Seconds(), colorReset)

	smartDisplayLine(content)
}

func displayTestResult(result *TestResult, success bool) {
	if success {
		// Don't display PASSED and SKIPPED tests (only update progress bar numbers)
	} else {
		// If failed, clear the current line and display on a new line
		fmt.Print("\r\033[K")
		lastDisplayLength = 0 // Reset display length to start a new line
		// Don't display if parent test with subtests fails (only display subtest failures)
		if result.HasSubtest {
			return
		}

		// Display test name and location information
		if result.Location != "" {
			fmt.Printf("%s✗ FAIL%s %s %s[%s]%s %s(%.2fs)%s\n",
				colorRed, colorReset, result.Test, colorBlue, result.Location, colorReset, colorGray, result.Elapsed, colorReset)
		} else {
			fmt.Printf("%s✗ FAIL%s %s %s(%.2fs)%s\n",
				colorRed, colorReset, result.Test, colorGray, result.Elapsed, colorReset)
		}

		// Display error output (display all related output)
		relevantOutput := extractRelevantOutput(result.Output)
		if len(relevantOutput) > 0 {
			fmt.Printf("\n")
			for _, line := range relevantOutput {
				fmt.Printf("        %s%s%s", colorRed, line, colorReset)
			}
			fmt.Printf("\n")
		}
	}
}

func extractRelevantOutput(output []string) []string {
	var relevant []string
	for _, line := range output {
		trimmed := strings.TrimSpace(line)
		// Skip empty lines but greatly relax other restrictions
		if trimmed != "" &&
			!strings.HasPrefix(trimmed, "=== RUN") &&
			!strings.HasPrefix(trimmed, "=== PAUSE") &&
			!strings.HasPrefix(trimmed, "=== CONT") {
			relevant = append(relevant, line)
		}
	}

	// Remove line limit - display all error information
	return relevant
}

// extractFileLocation extracts file:line information from test output
func extractFileLocation(output string) string {
	trimmed := strings.TrimSpace(output)
	// Look for filename:line_number: pattern
	parts := strings.SplitN(trimmed, ":", 3)
	if len(parts) >= 2 {
		// If filename ends with _test.go and next is a number
		if strings.HasSuffix(parts[0], "_test.go") {
			if _, err := fmt.Sscanf(parts[1], "%d", new(int)); err == nil {
				return parts[0] + ":" + parts[1]
			}
		}
	}
	return ""
}

// shouldDisplayPackageFailure determines if package failure should be displayed
// Returns true if the failure is due to build errors or other non-test issues
func shouldDisplayPackageFailure(pkg *PackageState) bool {
	// Check if package output contains symptoms of build errors, etc.
	for _, line := range pkg.Output {
		trimmed := strings.TrimSpace(line)
		// Check for symptoms of build errors
		if strings.Contains(trimmed, "[build failed]") ||
			strings.Contains(trimmed, "build constraints exclude all Go files") ||
			strings.Contains(trimmed, "no buildable Go source files") ||
			strings.Contains(trimmed, "syntax error") ||
			strings.Contains(trimmed, "cannot find package") ||
			strings.Contains(trimmed, "undefined:") {
			return true
		}
	}

	// If no individual tests exist (0 tests) and the package fails
	if pkg.Total == 0 && len(pkg.Output) > 0 {
		return true
	}

	// Don't display in other cases (only individual test failures)
	return false
}

func displayPackageFailure(packageName string, output []string) {
	// In case of package failure, clear the current line and display on a new line
	fmt.Print("\r\033[K")
	lastDisplayLength = 0 // Reset display length to start a new line
	fmt.Printf("%s✗ PACKAGE FAIL%s %s\n", colorRed, colorReset, packageName)

	// Display error output (display all related output)
	relevantOutput := extractRelevantOutput(output)
	if len(relevantOutput) > 0 {
		fmt.Printf("\n")
		for _, line := range relevantOutput {
			fmt.Printf("        %s%s%s", colorRed, line, colorReset)
		}
		fmt.Printf("\n")
	}
}

func displayFinalResults(packages map[string]*PackageState, results map[string]*TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 Test Results Summary")
	fmt.Println(strings.Repeat("=", 50))

	// Collect data using the same method as progress display
	totalTests := 0
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	var totalElapsed float64
	exitCode := 0

	// First collect overall statistics (same method as progress display)
	for _, pkg := range packages {
		totalTests += pkg.Total
		totalPassed += pkg.Passed
		totalFailed += pkg.Failed
		totalSkipped += pkg.Skipped
		totalElapsed += pkg.Elapsed
	}

	for pkgName, pkg := range packages {
		// Check if there is a Package Fail
		hasPackageFail := false
		if packageResult, exists := results[fmt.Sprintf("%s/[PACKAGE]", pkgName)]; exists {
			if packageResult.Failed && shouldDisplayPackageFailure(pkg) {
				hasPackageFail = true
			}
		}

		// Skip if there are 0 tests and no Package Fail
		if pkg.Total == 0 && !hasPackageFail {
			continue
		}

		// Display only if there are failures or Package Fail
		if pkg.Failed == 0 && !hasPackageFail {
			continue
		}

		// Count as failure if there is Package Fail (unused but kept for future extension)
		_ = hasPackageFail

		status := colorGreen + "✓ PASS" + colorReset
		if pkg.Failed > 0 || hasPackageFail {
			if hasPackageFail && pkg.Failed == 0 {
				// When only Package Fail
				status = colorRed + "✗ PACKAGE FAIL" + colorReset
			} else {
				// When there are individual test failures
				status = colorRed + "✗ FAIL" + colorReset
			}
			exitCode = 1
		}

		fmt.Printf("\n%s %s %s(%.2fs)%s\n", status, pkgName, colorGray, pkg.Elapsed, colorReset)

		// Don't display details when only Package Fail
		if hasPackageFail && pkg.Failed == 0 {
			// Display nothing when only Package Fail (title line only)
		} else {
			// Display details only when there are individual test failures
			// Avoid duplication if Package Fail is already counted in progress display
			totalTests := pkg.Total
			totalFailedInPkg := pkg.Failed
			fmt.Printf("  Tests: %d | Passed: %d | Failed: %d | Skipped: %d\n",
				totalTests, pkg.Passed, totalFailedInPkg, pkg.Skipped)

			// Details of failed tests
			if pkg.Failed > 0 || hasPackageFail {
				// Don't display 'Failed Tests:' when only Package Fail
				if pkg.Failed > 0 {
					fmt.Printf("\n")

					// Display individual test failures
					for _, result := range results {
						if result.Package == pkgName && result.Failed && result.Test != "[PACKAGE]" && !result.HasSubtest {
							// Don't display parent tests that have subtests
							// Display if location information is available
							if result.Location != "" {
								fmt.Printf("    %s✗ %s%s %s[%s]%s %s(%.2fs)%s\n",
									colorRed, result.Test, colorReset, colorBlue, result.Location, colorReset, colorGray, result.Elapsed, colorReset)
							} else {
								fmt.Printf("    %s✗ %s%s %s(%.2fs)%s\n",
									colorRed, result.Test, colorReset, colorGray, result.Elapsed, colorReset)
							}
						}
					}
				}
			}
		}
	}

	// Overall summary
	fmt.Println("\n" + strings.Repeat("-", 50))

	// Don't calculate duplicates since Package Fail is already counted in progress display
	// Use the time measured within the CLI for overall execution time
	actualElapsed := time.Since(startTime)
	fmt.Printf("Total: %d tests | %s✓ Passed: %d%s | %s✗ Failed: %d%s | %s⚡ Skipped: %d%s | %s⏱ %.2fs%s\n",
		totalTests,
		colorGreen, totalPassed, colorReset,
		colorRed, totalFailed, colorReset,
		colorYellow, totalSkipped, colorReset,
		colorGray, actualElapsed.Seconds(), colorReset)

	// Determine if there are failures (Package Fail already counted in progress display)
	if exitCode != 0 || totalFailed > 0 {
		fmt.Printf("\n%s❌ Tests failed!%s\n", colorRed, colorReset)
		exitWithCursorRestore(1)
	} else {
		fmt.Printf("\n%s✨ All tests passed!%s\n", colorGreen, colorReset)
		exitWithCursorRestore(0)
	}
}
