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
	SetConfig(config *Config)
}

// TerminalDisplay implements Display for terminal output
type TerminalDisplay struct {
	writer            io.Writer
	lastDisplayLength int
	colorEnabled      bool
	config            *Config
	packages          map[string]*PackageState
}

// NewTerminalDisplay creates a new TerminalDisplay
func NewTerminalDisplay(writer io.Writer, colorEnabled bool) Display {
	return &TerminalDisplay{
		writer:       writer,
		colorEnabled: colorEnabled,
		packages:     make(map[string]*PackageState),
	}
}

// SetConfig sets the configuration for the display
func (d *TerminalDisplay) SetConfig(config *Config) {
	d.config = config
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
	// Update packages for use in ShowTestResult
	d.packages = packages

	// In CI mode, don't show progress updates
	if d.config != nil && d.config.CIMode {
		return
	}

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
	// In CI mode, only show failed tests with simple format (no escape sequences)
	if d.config != nil && d.config.CIMode {
		// Only show failures
		if success {
			return
		}

		// Don't display if parent test with subtests fails (only display subtest failures)
		if result.HasSubtest {
			return
		}

		// Handle build errors differently in CI mode
		if result.Test == "[BUILD]" {
			shortPkg := getShortPackageName(result.Package)
			if result.Location != "" {
				fmt.Fprintf(d.writer, "BUILD FAIL %s [%s]\n", shortPkg, result.Location)
			} else {
				fmt.Fprintf(d.writer, "BUILD FAIL %s\n", shortPkg)
			}
		} else {
			// Simple format without colors or escape sequences
			if result.Location != "" {
				fmt.Fprintf(d.writer, "FAIL %s [%s] (%.2fs)\n", result.Test, result.Location, result.Elapsed)
			} else {
				fmt.Fprintf(d.writer, "FAIL %s (%.2fs)\n", result.Test, result.Elapsed)
			}
		}

		// Display error output
		relevantOutput := extractRelevantOutput(result.Output)
		if len(relevantOutput) > 0 {
			fmt.Fprintf(d.writer, "\n")
			for _, line := range relevantOutput {
				fmt.Fprintf(d.writer, "        %s", line)
			}
			fmt.Fprintf(d.writer, "\n")
		}
		return
	}

	// In timing mode, show all test results with execution time
	if d.config != nil && d.config.TimingMode {
		// Don't display if parent test with subtests fails (only display subtest failures)
		if result.HasSubtest {
			return
		}

		// Format execution time
		elapsed := formatDuration(result.Elapsed)

		// Check if test is slow
		isSlow := false
		if d.config.Threshold > 0 && time.Duration(result.Elapsed*float64(time.Second)) > d.config.Threshold {
			isSlow = true
		}

		// In timing mode, only show slow tests or failed tests
		if !isSlow && !result.Failed {
			return
		}

		// Display test result with timing
		var icon, color string
		switch {
		case result.Failed:
			icon = "✗"
			color = colorRed
		case result.Skipped:
			icon = "⚡"
			color = colorYellow
		case result.Passed:
			icon = "✓"
			color = colorGreen
		}

		// Display test result with timing and slow indicator if applicable
		slowIndicator := ""
		if isSlow {
			slowIndicator = fmt.Sprintf(" %s[SLOW]%s", colorRed, colorReset)
		}

		// Check if we should show package name (only when multiple packages)
		packageInfo := ""
		if shouldShowPackageName(d.packages) {
			shortPkg := getShortPackageName(result.Package)
			packageInfo = fmt.Sprintf(" %s%s%s", colorGray, shortPkg, colorReset)
		}

		if result.Location != "" {
			fmt.Fprintf(d.writer, "\r\033[K%s%s%s %s %s[%s]%s %s(%s)%s%s%s\n",
				color, icon, colorReset, result.Test, colorBlue, result.Location, colorReset,
				colorGray, elapsed, colorReset, slowIndicator, packageInfo)
		} else {
			fmt.Fprintf(d.writer, "\r\033[K%s%s%s %s %s(%s)%s%s%s\n",
				color, icon, colorReset, result.Test, colorGray, elapsed, colorReset, slowIndicator, packageInfo)
		}

		// Show error output for failed tests
		if result.Failed {
			relevantOutput := extractRelevantOutput(result.Output)
			if len(relevantOutput) > 0 {
				fmt.Fprintf(d.writer, "\n")
				for _, line := range relevantOutput {
					fmt.Fprintf(d.writer, "        %s%s%s", colorRed, line, colorReset)
				}
				fmt.Fprintf(d.writer, "\n")
			}
		}
		return
	}

	// Normal mode - only show failures
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

	// Handle build errors differently
	if result.Test == "[BUILD]" {
		shortPkg := getShortPackageName(result.Package)
		if result.Location != "" {
			fmt.Fprintf(d.writer, "%s✗ BUILD FAIL%s %s %s[%s]%s\n",
				colorRed, colorReset, shortPkg, colorBlue, result.Location, colorReset)
		} else {
			fmt.Fprintf(d.writer, "%s✗ BUILD FAIL%s %s\n",
				colorRed, colorReset, shortPkg)
		}
	} else {
		// Display test name and location information
		if result.Location != "" {
			fmt.Fprintf(d.writer, "%s✗ FAIL%s %s %s[%s]%s %s(%.2fs)%s\n",
				colorRed, colorReset, result.Test, colorBlue, result.Location, colorReset, colorGray, result.Elapsed, colorReset)
		} else {
			fmt.Fprintf(d.writer, "%s✗ FAIL%s %s %s(%.2fs)%s\n",
				colorRed, colorReset, result.Test, colorGray, result.Elapsed, colorReset)
		}
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
	// In CI mode, simple format without colors or escape sequences
	if d.config != nil && d.config.CIMode {
		fmt.Fprintf(d.writer, "PACKAGE FAIL %s\n", packageName)

		// Display error output
		relevantOutput := extractRelevantOutput(output)
		if len(relevantOutput) > 0 {
			fmt.Fprintf(d.writer, "\n")
			for _, line := range relevantOutput {
				fmt.Fprintf(d.writer, "        %s", line)
			}
			fmt.Fprintf(d.writer, "\n")
		}
		return
	}

	// In case of package failure, clear the current line and display on a new line
	d.ClearLine()
	shortPkg := getShortPackageName(packageName)
	fmt.Fprintf(d.writer, "%s✗ PACKAGE FAIL%s %s\n", colorRed, colorReset, shortPkg)

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

	// In CI mode, show simple summary without colors or decorations
	if d.config != nil && d.config.CIMode {
		actualElapsed := time.Since(startTime)

		// Show failure summary if there are any failures (without colors/decorations)
		if stats.hasFailures {
			fmt.Fprintln(d.writer, "\n"+strings.Repeat("=", 50))
			fmt.Fprintln(d.writer, "Failed Tests Summary")
			fmt.Fprintln(d.writer, strings.Repeat("=", 50))

			for pkgName, pkg := range packages {
				if code := d.displayPackageSummaryCI(pkgName, pkg, results); code != 0 {
					exitCode = code
				}
			}

			fmt.Fprintln(d.writer, "\n"+strings.Repeat("-", 50))
		}

		// Simple summary
		fmt.Fprintf(d.writer, "\nTotal: %d tests | Passed: %d | Failed: %d | Skipped: %d | Time: %.2fs\n",
			stats.totalTests, stats.totalPassed, stats.totalFailed, stats.totalSkipped, actualElapsed.Seconds())

		// Final status message
		if stats.totalFailed > 0 {
			fmt.Fprintf(d.writer, "\nTests failed!\n")
			return 1
		} else {
			fmt.Fprintf(d.writer, "\nAll tests passed!\n")
			return 0
		}
	}

	// In timing mode, show slow tests summary
	if d.config != nil && d.config.TimingMode {
		d.showSlowTestsSummary(results)
	}

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
	fmt.Fprintln(d.writer, "  go test -json ./... | gotestshow [flags]")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Flags:")
	fmt.Fprintln(d.writer, "  -timing         Enable timing mode to show only slow tests and failures")
	fmt.Fprintln(d.writer, "  -threshold      Threshold for slow tests (default: 500ms)")
	fmt.Fprintln(d.writer, "                  Examples: 1s, 500ms, 1.5s")
	fmt.Fprintln(d.writer, "  -ci             Enable CI mode - no escape sequences, only show failures and summary")
	fmt.Fprintln(d.writer, "  -help           Show this help message")
	fmt.Fprintln(d.writer)
	fmt.Fprintln(d.writer, "Description:")
	fmt.Fprintln(d.writer, "  gotestshow reads JSON-formatted test output from stdin and displays")
	fmt.Fprintln(d.writer, "  it in a human-readable format with real-time progress updates.")
	fmt.Fprintln(d.writer, "  Real-time progress display shows only failed test details.")
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
	fmt.Fprintln(d.writer, "  # Enable timing mode with custom threshold")
	fmt.Fprintln(d.writer, "  go test -json ./... | gotestshow -timing -threshold=1s")
}

// ClearLine clears the current line
func (d *TerminalDisplay) ClearLine() {
	// In CI mode, don't use escape sequences
	if d.config != nil && d.config.CIMode {
		return
	}
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

	shortPkg := getShortPackageName(pkgName)
	fmt.Fprintf(d.writer, "\n%s %s %s(%.2fs)%s\n", status, shortPkg, colorGray, pkg.Elapsed, colorReset)

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
			if result.Test == "[BUILD]" {
				if result.Location != "" {
					fmt.Fprintf(d.writer, "    %s✗ BUILD FAIL%s %s[%s]%s\n",
						colorRed, colorReset, colorBlue, result.Location, colorReset)
				} else {
					fmt.Fprintf(d.writer, "    %s✗ BUILD FAIL%s\n",
						colorRed, colorReset)
				}
			} else {
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
}

func (d *TerminalDisplay) displayPackageSummaryCI(pkgName string, pkg *PackageState, results map[string]*TestResult) int {
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

	status := "PASS"
	exitCode := 0
	if pkg.Failed > 0 || hasPackageFail {
		if hasPackageFail && pkg.Failed == 0 {
			status = "PACKAGE FAIL"
		} else {
			status = "FAIL"
		}
		exitCode = 1
	}

	shortPkg := getShortPackageName(pkgName)
	fmt.Fprintf(d.writer, "\n%s %s (%.2fs)\n", status, shortPkg, pkg.Elapsed)

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
		d.displayFailedTestsForCI(pkgName, pkg, results)
	}

	return exitCode
}

func (d *TerminalDisplay) displayFailedTestsForCI(pkgName string, pkg *PackageState, results map[string]*TestResult) {
	for _, result := range results {
		if result.Package == pkgName && result.Failed && result.Test != "[PACKAGE]" && !result.HasSubtest {
			if result.Test == "[BUILD]" {
				if result.Location != "" {
					fmt.Fprintf(d.writer, "    BUILD FAIL [%s]\n", result.Location)
				} else {
					fmt.Fprintf(d.writer, "    BUILD FAIL\n")
				}
			} else {
				if result.Location != "" {
					fmt.Fprintf(d.writer, "    FAIL %s [%s] (%.2fs)\n",
						result.Test, result.Location, result.Elapsed)
				} else {
					fmt.Fprintf(d.writer, "    FAIL %s (%.2fs)\n",
						result.Test, result.Elapsed)
				}
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

// formatDuration formats a duration in seconds to a human-readable string
func formatDuration(seconds float64) string {
	if seconds < 1.0 {
		return fmt.Sprintf("%.0fms", seconds*1000)
	}
	return fmt.Sprintf("%.3fs", seconds)
}

// getShortPackageName extracts the last part of a package path
func getShortPackageName(fullPackage string) string {
	parts := strings.Split(fullPackage, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullPackage
}

// shouldShowPackageName determines if package name should be shown in real-time display
func shouldShowPackageName(packages map[string]*PackageState) bool {
	// Count packages that have actual tests (not just package failures)
	packageCount := 0
	for _, pkg := range packages {
		if pkg.Total > 0 {
			packageCount++
		}
	}
	return packageCount > 1
}

// showSlowTestsSummary displays a summary of slow tests
func (d *TerminalDisplay) showSlowTestsSummary(results map[string]*TestResult) {
	// Collect slow tests grouped by package
	type slowTest struct {
		name     string
		elapsed  float64
		location string
	}

	slowTestsByPackage := make(map[string][]slowTest)
	for _, result := range results {
		if result.HasSubtest || result.Test == "[PACKAGE]" {
			continue
		}

		testDuration := time.Duration(result.Elapsed * float64(time.Second))
		if testDuration > d.config.Threshold {
			test := slowTest{
				name:     result.Test,
				elapsed:  result.Elapsed,
				location: result.Location,
			}
			slowTestsByPackage[result.Package] = append(slowTestsByPackage[result.Package], test)
		}
	}

	if len(slowTestsByPackage) == 0 {
		return
	}

	// Display slow tests summary grouped by package
	fmt.Fprintln(d.writer, "\n"+strings.Repeat("=", 50))
	fmt.Fprintf(d.writer, "🐢 Slow Tests (>%s)\n", d.config.Threshold)
	fmt.Fprintln(d.writer, strings.Repeat("=", 50))

	for pkgName, tests := range slowTestsByPackage {
		// Sort tests by elapsed time (slowest first)
		for i := 0; i < len(tests)-1; i++ {
			for j := i + 1; j < len(tests); j++ {
				if tests[j].elapsed > tests[i].elapsed {
					tests[i], tests[j] = tests[j], tests[i]
				}
			}
		}

		shortPkg := getShortPackageName(pkgName)
		fmt.Fprintf(d.writer, "\n=== %s%s%s ===\n", colorBlue, shortPkg, colorReset)

		for _, test := range tests {
			elapsed := formatDuration(test.elapsed)
			if test.location != "" {
				fmt.Fprintf(d.writer, "  %s %s[%s]%s %s(%s)%s\n",
					test.name, colorBlue, test.location, colorReset, colorRed, elapsed, colorReset)
			} else {
				fmt.Fprintf(d.writer, "  %s %s(%s)%s\n",
					test.name, colorRed, elapsed, colorReset)
			}
		}
	}
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
