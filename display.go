// ABOUTME: display.go contains all display-related functionality for formatting and showing test results.
// ABOUTME: It provides interfaces and implementations for different display strategies.

package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// Display interface defines methods for displaying test progress and results
type Display interface {
	ShowProgress(packages map[string]*PackageState, hasTestsStarted bool, startTime time.Time)
	ShowTestResult(result *TestResult, success bool)
	ShowPackageFailure(packageName string, output []string)
	ShowFinalResults(packages map[string]*PackageState, results map[string]*TestResult, startTime time.Time) int
	ShowHelp()
	ClearLine()
}

// TerminalDisplay implements Display for terminal output
type TerminalDisplay struct {
	writer            io.Writer
	lastDisplayLength int
	colorEnabled      bool
}

// NewTerminalDisplay creates a new TerminalDisplay
func NewTerminalDisplay(writer io.Writer, colorEnabled bool) Display {
	return &TerminalDisplay{
		writer:       writer,
		colorEnabled: colorEnabled,
	}
}

// Animation provides animated characters for display
type Animation struct {
	spinnerChars []string
	dotsChars    []string
}

// NewAnimation creates a new Animation instance
func NewAnimation() *Animation {
	return &Animation{
		spinnerChars: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		dotsChars:    []string{"   ", ".  ", ".. ", "..."},
	}
}

func (a *Animation) GetSpinner() string {
	return a.getAnimation(a.spinnerChars, 100)
}

func (a *Animation) GetDots() string {
	return a.getAnimation(a.dotsChars, 500)
}

func (a *Animation) getAnimation(chars []string, intervalMs int64) string {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	index := (now / intervalMs) % int64(len(chars))
	return chars[index]
}

// ShowProgress displays the current test progress
func (d *TerminalDisplay) ShowProgress(packages map[string]*PackageState, hasTestsStarted bool, startTime time.Time) {
	animation := NewAnimation()
	spinner := animation.GetSpinner()
	elapsed := time.Since(startTime)

	// Show simple initialization message until first test starts
	if !hasTestsStarted {
		dots := animation.GetDots()
		content := fmt.Sprintf("%s%s Initializing%s%s", colorBlue, spinner, dots, colorReset)
		d.smartDisplayLine(content)
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
		colorBlue, spinner, totalRunning, colorReset,
		colorGreen, totalPassed, colorReset,
		colorRed, totalFailed, colorReset,
		colorYellow, totalSkipped, colorReset,
		colorGray, elapsed.Seconds(), colorReset)

	d.smartDisplayLine(content)
}

// ShowTestResult displays the result of a single test
func (d *TerminalDisplay) ShowTestResult(result *TestResult, success bool) {
	if success {
		// Don't display PASSED and SKIPPED tests (only update progress bar numbers)
		return
	}

	// If failed, clear the current line and display on a new line
	d.ClearLine()
	// Don't display if parent test with subtests fails (only display subtest failures)
	if result.HasSubtest {
		return
	}

	// Display test name and location information
	if result.Location != "" {
		fmt.Fprintf(d.writer, "%s✗ FAIL%s %s %s[%s]%s %s(%.2fs)%s\n",
			colorRed, colorReset, result.Test, colorBlue, result.Location, colorReset, colorGray, result.Elapsed, colorReset)
	} else {
		fmt.Fprintf(d.writer, "%s✗ FAIL%s %s %s(%.2fs)%s\n",
			colorRed, colorReset, result.Test, colorGray, result.Elapsed, colorReset)
	}

	// Display error output (display all related output)
	relevantOutput := extractRelevantOutput(result.Output)
	if len(relevantOutput) > 0 {
		fmt.Fprintf(d.writer, "\n")
		for _, line := range relevantOutput {
			fmt.Fprintf(d.writer, "        %s%s%s", colorRed, line, colorReset)
		}
		fmt.Fprintf(d.writer, "\n")
	}
}

// ShowPackageFailure displays package-level failures
func (d *TerminalDisplay) ShowPackageFailure(packageName string, output []string) {
	// In case of package failure, clear the current line and display on a new line
	d.ClearLine()
	fmt.Fprintf(d.writer, "%s✗ PACKAGE FAIL%s %s\n", colorRed, colorReset, packageName)

	// Display error output (display all related output)
	relevantOutput := extractRelevantOutput(output)
	if len(relevantOutput) > 0 {
		fmt.Fprintf(d.writer, "\n")
		for _, line := range relevantOutput {
			fmt.Fprintf(d.writer, "        %s%s%s", colorRed, line, colorReset)
		}
		fmt.Fprintf(d.writer, "\n")
	}
}

// ShowFinalResults displays the final test results summary
func (d *TerminalDisplay) ShowFinalResults(packages map[string]*PackageState, results map[string]*TestResult, startTime time.Time) int {
	stats := collectSummaryStats(packages, results)
	exitCode := 0

	// Show failure summary if there are any failures
	if stats.hasFailures {
		fmt.Fprintln(d.writer, "\n"+strings.Repeat("=", 50))
		fmt.Fprintln(d.writer, "📊 Failed Tests Summary")
		fmt.Fprintln(d.writer, strings.Repeat("=", 50))

		for pkgName, pkg := range packages {
			if code := d.displayPackageSummary(pkgName, pkg, results); code != 0 {
				exitCode = code
			}
		}

		fmt.Fprintln(d.writer, "\n"+strings.Repeat("-", 50))
	}

	// Overall summary
	actualElapsed := time.Since(startTime)
	fmt.Fprintf(d.writer, "\nTotal: %d tests | %s✓ Passed: %d%s | %s✗ Failed: %d%s | %s⚡ Skipped: %d%s | %s⏱ %.2fs%s\n",
		stats.totalTests,
		colorGreen, stats.totalPassed, colorReset,
		colorRed, stats.totalFailed, colorReset,
		colorYellow, stats.totalSkipped, colorReset,
		colorGray, actualElapsed.Seconds(), colorReset)

	// Final status message
	if exitCode != 0 || stats.totalFailed > 0 {
		fmt.Fprintf(d.writer, "\n%s❌ Tests failed!%s\n", colorRed, colorReset)
		return 1
	} else {
		fmt.Fprintf(d.writer, "\n%s✨ All tests passed!%s\n", colorGreen, colorReset)
		return 0
	}
}

// ShowHelp displays the help message
func (d *TerminalDisplay) ShowHelp() {
	fmt.Fprintln(d.writer, "gotestshow - A real-time formatter for `go test -json` output")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Usage:")
	fmt.Fprintln(d.writer, "  go test -json ./... | gotestshow")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Description:")
	fmt.Fprintln(d.writer, "  gotestshow reads JSON-formatted test output from stdin and displays")
	fmt.Fprintln(d.writer, "  it in a human-readable format with real-time progress updates.")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Features:")
	fmt.Fprintln(d.writer, "  • Real-time progress with animated spinner")
	fmt.Fprintln(d.writer, "  • Shows only failed test details to reduce noise")
	fmt.Fprintln(d.writer, "  • Smart handling of subtests and parallel tests")
	fmt.Fprintln(d.writer, "  • Package-level failure detection (build errors, syntax errors)")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Examples:")
	fmt.Fprintln(d.writer, "  # Test all packages")
	fmt.Fprintln(d.writer, "  go test -json ./... | gotestshow")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  # Test specific package")
	fmt.Fprintln(d.writer, "  go test -json ./pkg/... | gotestshow")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "  # Run specific test")
	fmt.Fprintln(d.writer, "  go test -json -run TestName ./... | gotestshow")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Options:")
	fmt.Fprintln(d.writer, "  -help  Show this help message")
}

// ClearLine clears the current line
func (d *TerminalDisplay) ClearLine() {
	fmt.Fprint(d.writer, "\r\033[K")
	d.lastDisplayLength = 0
}

// smartDisplayLine compares the length of display content and updates the line appropriately
func (d *TerminalDisplay) smartDisplayLine(content string) {
	currentLength := len(content)

	if currentLength < d.lastDisplayLength {
		// When new content is shorter: \r (return to beginning of line) + \033[K (clear to end of line) then display
		fmt.Fprint(d.writer, "\r\033[K")
		fmt.Fprint(d.writer, content)
	} else {
		// When new content is same length or longer: overwrite with \r (return to beginning of line)
		fmt.Fprint(d.writer, "\r")
		fmt.Fprint(d.writer, content)
	}

	d.lastDisplayLength = currentLength
}

func (d *TerminalDisplay) displayPackageSummary(pkgName string, pkg *PackageState, results map[string]*TestResult) int {
	// Check if there is a Package Fail
	hasPackageFail := false
	if packageResult, exists := results[fmt.Sprintf("%s/[PACKAGE]", pkgName)]; exists {
		if packageResult.Failed && shouldDisplayPackageFailure(pkg) {
			hasPackageFail = true
		}
	}

	// Skip if there are 0 tests and no Package Fail
	if pkg.Total == 0 && !hasPackageFail {
		return 0
	}

	// Display only if there are failures or Package Fail
	if pkg.Failed == 0 && !hasPackageFail {
		return 0
	}

	status := colorGreen + "✓ PASS" + colorReset
	exitCode := 0
	if pkg.Failed > 0 || hasPackageFail {
		if hasPackageFail && pkg.Failed == 0 {
			status = colorRed + "✗ PACKAGE FAIL" + colorReset
		} else {
			status = colorRed + "✗ FAIL" + colorReset
		}
		exitCode = 1
	}

	fmt.Fprintf(d.writer, "\n%s %s %s(%.2fs)%s\n", status, pkgName, colorGray, pkg.Elapsed, colorReset)

	// Don't display details when only Package Fail
	if hasPackageFail && pkg.Failed == 0 {
		return exitCode
	}

	// Display test statistics
	fmt.Fprintf(d.writer, "  Tests: %d | Passed: %d | Failed: %d | Skipped: %d\n",
		pkg.Total, pkg.Passed, pkg.Failed, pkg.Skipped)

	// Display individual test failures
	if pkg.Failed > 0 {
		fmt.Fprintf(d.writer, "\n")
		d.displayFailedTestsFor(pkgName, pkg, results)
	}

	return exitCode
}

func (d *TerminalDisplay) displayFailedTestsFor(pkgName string, pkg *PackageState, results map[string]*TestResult) {
	for _, result := range results {
		if result.Package == pkgName && result.Failed && result.Test != "[PACKAGE]" && !result.HasSubtest {
			if result.Location != "" {
				fmt.Fprintf(d.writer, "    %s✗ %s%s %s[%s]%s %s(%.2fs)%s\n",
					colorRed, result.Test, colorReset, colorBlue, result.Location, colorReset, colorGray, result.Elapsed, colorReset)
			} else {
				fmt.Fprintf(d.writer, "    %s✗ %s%s %s(%.2fs)%s\n",
					colorRed, result.Test, colorReset, colorGray, result.Elapsed, colorReset)
			}
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

func collectSummaryStats(packages map[string]*PackageState, results map[string]*TestResult) summaryStats {
	stats := summaryStats{}
	for _, pkg := range packages {
		stats.totalTests += pkg.Total
		stats.totalPassed += pkg.Passed
		stats.totalFailed += pkg.Failed
		stats.totalSkipped += pkg.Skipped

		if pkg.Failed > 0 {
			stats.hasFailures = true
		}
		// Check for package-level failures
		if packageResult, exists := results[fmt.Sprintf("%s/[PACKAGE]", pkg.Name)]; exists {
			if packageResult.Failed && shouldDisplayPackageFailure(pkg) {
				stats.hasFailures = true
			}
		}
	}
	return stats
}

// ProgressRunner manages the progress display goroutine
type ProgressRunner struct {
	display        Display
	processor      EventProcessor
	updateInterval time.Duration
}

// NewProgressRunner creates a new ProgressRunner
func NewProgressRunner(display Display, processor EventProcessor, updateInterval time.Duration) *ProgressRunner {
	return &ProgressRunner{
		display:        display,
		processor:      processor,
		updateInterval: updateInterval,
	}
}

// Run starts the progress display loop
func (pr *ProgressRunner) Run(ctx context.Context, startTime time.Time) {
	ticker := time.NewTicker(pr.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			packages := pr.processor.GetPackages()
			hasStarted := pr.processor.HasTestsStarted()
			pr.display.ShowProgress(packages, hasStarted, startTime)
		}
	}
}
