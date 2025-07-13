// ABOUTME: display.go contains all display-related functionality for formatting and showing test results.
// ABOUTME: It provides interfaces and implementations for different display strategies.

package main

import (
	"context"
	"fmt"
	"io"
	"sort"
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
	// Skip parent tests with subtests in all modes
	if result.HasSubtest {
		return
	}

	switch {
	case d.config != nil && d.config.CIMode:
		d.showTestResultCI(result, success)
	case d.config != nil && d.config.TimingMode:
		d.showTestResultTiming(result, success)
	default:
		d.showTestResultNormal(result, success)
	}
}

func (d *TerminalDisplay) showTestResultCI(result *TestResult, success bool) {
	// CI mode: Only show failures, no colors
	if success {
		return
	}

	d.printTestFailureCI(result)
	d.printTestOutput(result.Output, false)
}

func (d *TerminalDisplay) showTestResultTiming(result *TestResult, success bool) {
	// Timing mode: Show slow tests and failures
	elapsed := formatDuration(result.Elapsed)
	isSlow := d.isSlowTest(result.Elapsed)

	if !isSlow && !result.Failed {
		return
	}

	icon, color := d.getTestIcon(result)
	slowIndicator := ""
	if isSlow {
		slowIndicator = fmt.Sprintf(" %s[SLOW]%s", colorRed, colorReset)
	}

	d.printTestResult(icon, color, result, elapsed, slowIndicator)

	if result.Failed {
		d.printTestOutput(result.Output, true)
	}
}

func (d *TerminalDisplay) showTestResultNormal(result *TestResult, success bool) {
	// Normal mode: Only show failures
	if success {
		return
	}

	d.ClearLine()
	d.printTestFailure(result)
	d.printTestOutput(result.Output, true)
}

func (d *TerminalDisplay) isSlowTest(elapsed float64) bool {
	return d.config.Threshold > 0 &&
		time.Duration(elapsed*float64(time.Second)) > d.config.Threshold
}

func (d *TerminalDisplay) getTestIcon(result *TestResult) (string, string) {
	switch {
	case result.Failed:
		return "✗", colorRed
	case result.Skipped:
		return "⚡", colorYellow
	case result.Passed:
		return "✓", colorGreen
	default:
		return "?", colorGray
	}
}

func (d *TerminalDisplay) printTestFailureCI(result *TestResult) {
	if result.Test == "[BUILD]" {
		shortPkg := getShortPackageName(result.Package)
		if result.Location != "" {
			fmt.Fprintf(d.writer, "BUILD FAIL %s [%s]\n", shortPkg, result.Location)
		} else {
			fmt.Fprintf(d.writer, "BUILD FAIL %s\n", shortPkg)
		}
	} else {
		packageInfo := ""
		if shouldShowPackageName(d.packages) {
			packageInfo = fmt.Sprintf(" in %s", result.Package)
		}

		if result.Location != "" {
			fmt.Fprintf(d.writer, "FAIL %s [%s] (%.2fs)%s\n", result.Test, result.Location, result.Elapsed, packageInfo)
		} else {
			fmt.Fprintf(d.writer, "FAIL %s (%.2fs)%s\n", result.Test, result.Elapsed, packageInfo)
		}
	}
}

func (d *TerminalDisplay) printTestFailure(result *TestResult) {
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
		packageInfo := ""
		if shouldShowPackageName(d.packages) {
			packageInfo = fmt.Sprintf(" in %s", result.Package)
		}

		if result.Location != "" {
			fmt.Fprintf(d.writer, "%s✗ FAIL%s %s %s[%s]%s %s(%.2fs)%s%s\n",
				colorRed, colorReset, result.Test, colorBlue, result.Location, colorReset, colorGray, result.Elapsed, colorReset, packageInfo)
		} else {
			fmt.Fprintf(d.writer, "%s✗ FAIL%s %s %s(%.2fs)%s%s\n",
				colorRed, colorReset, result.Test, colorGray, result.Elapsed, colorReset, packageInfo)
		}
	}
}

func (d *TerminalDisplay) printTestResult(icon, color string, result *TestResult, elapsed, slowIndicator string) {
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
}

func (d *TerminalDisplay) printTestOutput(output []string, withColor bool) {
	relevantOutput := extractRelevantOutput(output)
	if len(relevantOutput) == 0 {
		return
	}

	fmt.Fprintf(d.writer, "\n")
	for _, line := range relevantOutput {
		if withColor {
			fmt.Fprintf(d.writer, "        %s%s%s", colorRed, line, colorReset)
		} else {
			fmt.Fprintf(d.writer, "        %s", line)
		}
	}
	fmt.Fprintf(d.writer, "\n")
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
	// Return full package name instead of just the last part
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

type slowTest struct {
	name     string
	elapsed  float64
	location string
}

// showSlowTestsSummary displays a summary of slow tests
func (d *TerminalDisplay) showSlowTestsSummary(results map[string]*TestResult) {
	slowTestsByPackage := make(map[string][]slowTest)

	// Collect slow tests
	for _, result := range results {
		if result.HasSubtest || result.Test == "[PACKAGE]" {
			continue
		}

		if d.isSlowTest(result.Elapsed) {
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

	// Display header
	fmt.Fprintln(d.writer, "\n"+strings.Repeat("=", 50))
	fmt.Fprintf(d.writer, "🐢 Slow Tests (>%s)\n", d.config.Threshold)
	fmt.Fprintln(d.writer, strings.Repeat("=", 50))

	// Display tests by package
	for pkgName, tests := range slowTestsByPackage {
		// Sort by elapsed time (slowest first)
		sort.Slice(tests, func(i, j int) bool {
			return tests[i].elapsed > tests[j].elapsed
		})

		d.displaySlowTestsForPackage(pkgName, tests)
	}
}

func (d *TerminalDisplay) displaySlowTestsForPackage(pkgName string, tests []slowTest) {
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
