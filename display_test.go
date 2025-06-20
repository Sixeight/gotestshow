// ABOUTME: display_test.go contains unit tests for the Display functionality.
// ABOUTME: It tests display output formatting, progress reporting, and result presentation.

package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTerminalDisplay_ShowHelp(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	display.ShowHelp()

	output := buf.String()
	if !strings.Contains(output, "gotestshow - A real-time formatter") {
		t.Error("Help output should contain application description")
	}

	if !strings.Contains(output, "go test -json ./... | gotestshow") {
		t.Error("Help output should contain usage example")
	}

	if !strings.Contains(output, "Real-time progress") {
		t.Error("Help output should contain feature description")
	}
}

func TestTerminalDisplay_ShowProgress(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	packages := map[string]*PackageState{
		"example": {
			Name:    "example",
			Total:   3,
			Passed:  2,
			Failed:  1,
			Running: 0,
		},
	}

	startTime := time.Now().Add(-2 * time.Second)
	display.ShowProgress(packages, true, startTime)

	output := buf.String()
	if !strings.Contains(output, "Running: 0") {
		t.Error("Progress should show running count")
	}

	if !strings.Contains(output, "Passed: 2") {
		t.Error("Progress should show passed count")
	}

	if !strings.Contains(output, "Failed: 1") {
		t.Error("Progress should show failed count")
	}
}

func TestTerminalDisplay_ShowProgress_Initializing(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	packages := map[string]*PackageState{}
	startTime := time.Now()

	display.ShowProgress(packages, false, startTime)

	output := buf.String()
	if !strings.Contains(output, "Initializing") {
		t.Error("Should show initializing message when no tests started")
	}
}

func TestTerminalDisplay_ShowTestResult_Success(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	result := &TestResult{
		Package: "example",
		Test:    "TestExample",
		Passed:  true,
		Elapsed: 0.1,
	}

	display.ShowTestResult(result, true)

	// Success tests should not produce output
	output := buf.String()
	if output != "" {
		t.Error("Successful tests should not produce output")
	}
}

func TestTerminalDisplay_ShowTestResult_Failure(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	result := &TestResult{
		Package:  "example",
		Test:     "TestExample",
		Failed:   true,
		Elapsed:  0.2,
		Location: "example_test.go:15",
		Output:   []string{"    Error: assertion failed\n"},
	}

	display.ShowTestResult(result, false)

	output := buf.String()
	if !strings.Contains(output, "FAIL") {
		t.Error("Failed test should show FAIL")
	}

	if !strings.Contains(output, "TestExample") {
		t.Error("Failed test should show test name")
	}

	if !strings.Contains(output, "example_test.go:15") {
		t.Error("Failed test should show location")
	}

	if !strings.Contains(output, "0.2") {
		t.Error("Failed test should show elapsed time")
	}

	if !strings.Contains(output, "Error: assertion failed") {
		t.Error("Failed test should show error output")
	}
}

func TestTerminalDisplay_ShowTestResult_Failure_NoLocation(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	result := &TestResult{
		Package: "example",
		Test:    "TestExample",
		Failed:  true,
		Elapsed: 0.3,
		Output:  []string{"    Simple error\n"},
	}

	display.ShowTestResult(result, false)

	output := buf.String()
	if !strings.Contains(output, "FAIL") {
		t.Error("Failed test should show FAIL")
	}

	if !strings.Contains(output, "TestExample") {
		t.Error("Failed test should show test name")
	}

	if !strings.Contains(output, "0.3") {
		t.Error("Failed test should show elapsed time")
	}

	// Should not contain location brackets when no location
	if strings.Contains(output, "[") && strings.Contains(output, "]") {
		t.Error("Should not show location brackets when no location")
	}
}

func TestTerminalDisplay_ShowTestResult_Subtest(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	result := &TestResult{
		Package:    "example",
		Test:       "TestParent",
		Failed:     true,
		HasSubtest: true,
		Elapsed:    0.2,
	}

	display.ShowTestResult(result, false)

	// Parent test with subtests should not produce output
	output := buf.String()
	if output != "\r\033[K" && output != "" {
		t.Error("Parent test with subtests should not produce detailed output")
	}
}

func TestTerminalDisplay_ShowPackageFailure(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	output := []string{
		"# example [build failed]\n",
		"syntax error: unexpected token\n",
	}

	display.ShowPackageFailure("example", output)

	result := buf.String()
	if !strings.Contains(result, "PACKAGE FAIL") {
		t.Error("Package failure should show PACKAGE FAIL")
	}

	if !strings.Contains(result, "example") {
		t.Error("Package failure should show package name")
	}

	if !strings.Contains(result, "syntax error") {
		t.Error("Package failure should show error output")
	}
}

func TestTerminalDisplay_ShowFinalResults_Success(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	packages := map[string]*PackageState{
		"example": {
			Name:   "example",
			Total:  3,
			Passed: 3,
			Failed: 0,
		},
	}

	results := map[string]*TestResult{}
	startTime := time.Now().Add(-2 * time.Second)

	exitCode := display.ShowFinalResults(packages, results, startTime)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for successful tests, got %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "All tests passed") {
		t.Error("Should show success message")
	}

	if !strings.Contains(output, "Total: 3 tests") {
		t.Error("Should show total test count")
	}

	if !strings.Contains(output, "Passed: 3") {
		t.Error("Should show passed count")
	}
}

func TestTerminalDisplay_ShowFinalResults_Failure(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)

	packages := map[string]*PackageState{
		"example": {
			Name:   "example",
			Total:  3,
			Passed: 2,
			Failed: 1,
		},
	}

	results := map[string]*TestResult{
		"example/TestExample": {
			Package: "example",
			Test:    "TestExample",
			Failed:  true,
			Elapsed: 0.2,
		},
	}
	startTime := time.Now().Add(-2 * time.Second)

	exitCode := display.ShowFinalResults(packages, results, startTime)

	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for failed tests, got %d", exitCode)
	}

	output := buf.String()
	if !strings.Contains(output, "Tests failed") {
		t.Error("Should show failure message")
	}

	if !strings.Contains(output, "Failed Tests Summary") {
		t.Error("Should show failure summary")
	}

	if !strings.Contains(output, "Failed: 1") {
		t.Error("Should show failed count")
	}
}

func TestAnimation_GetSpinner(t *testing.T) {
	animation := NewAnimation()

	spinner1 := animation.GetSpinner()
	if spinner1 == "" {
		t.Error("Spinner should not be empty")
	}

	// Wait a bit and get another spinner character
	time.Sleep(150 * time.Millisecond)
	spinner2 := animation.GetSpinner()

	// They might be different due to timing, but both should be valid
	validChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	found1, found2 := false, false

	for _, char := range validChars {
		if spinner1 == char {
			found1 = true
		}
		if spinner2 == char {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("Invalid spinner character: %s", spinner1)
	}
	if !found2 {
		t.Errorf("Invalid spinner character: %s", spinner2)
	}
}

func TestAnimation_GetDots(t *testing.T) {
	animation := NewAnimation()

	dots := animation.GetDots()
	validDots := []string{"   ", ".  ", ".. ", "..."}

	found := false
	for _, dot := range validDots {
		if dots == dot {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Invalid dots pattern: '%s'", dots)
	}
}

func TestExtractRelevantOutput(t *testing.T) {
	output := []string{
		"=== RUN   TestExample\n",
		"    example_test.go:10: Error message\n",
		"    Detailed error information\n",
		"=== PAUSE TestExample\n",
		"=== CONT  TestExample\n",
		"--- FAIL: TestExample (0.00s)\n",
		"\n",
	}

	relevant := extractRelevantOutput(output)

	expected := []string{
		"    example_test.go:10: Error message\n",
		"    Detailed error information\n",
		"--- FAIL: TestExample (0.00s)\n",
	}

	if len(relevant) != len(expected) {
		t.Errorf("Expected %d relevant lines, got %d", len(expected), len(relevant))
	}

	for i, line := range expected {
		if i < len(relevant) && relevant[i] != line {
			t.Errorf("Expected line %d to be %q, got %q", i, line, relevant[i])
		}
	}
}

func TestCollectSummaryStats(t *testing.T) {
	packages := map[string]*PackageState{
		"example1": {
			Name:   "example1",
			Total:  3,
			Passed: 2,
			Failed: 1,
		},
		"example2": {
			Name:   "example2",
			Total:  2,
			Passed: 2,
			Failed: 0,
		},
	}

	results := map[string]*TestResult{}

	stats := collectSummaryStats(packages, results)

	if stats.totalTests != 5 {
		t.Errorf("Expected total tests 5, got %d", stats.totalTests)
	}

	if stats.totalPassed != 4 {
		t.Errorf("Expected total passed 4, got %d", stats.totalPassed)
	}

	if stats.totalFailed != 1 {
		t.Errorf("Expected total failed 1, got %d", stats.totalFailed)
	}

	if !stats.hasFailures {
		t.Error("Expected hasFailures to be true")
	}
}

func TestTerminalDisplay_CIMode_ShowProgress(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)
	config := &Config{CIMode: true}
	display.SetConfig(config)

	packages := map[string]*PackageState{
		"example": {
			Name:    "example",
			Total:   3,
			Passed:  2,
			Failed:  1,
			Running: 0,
		},
	}

	startTime := time.Now().Add(-2 * time.Second)
	display.ShowProgress(packages, true, startTime)

	// In CI mode, ShowProgress should produce no output
	output := buf.String()
	if output != "" {
		t.Error("CI mode should not show progress updates")
	}
}

func TestTerminalDisplay_CIMode_ShowTestResult_Success(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)
	config := &Config{CIMode: true}
	display.SetConfig(config)

	result := &TestResult{
		Package: "example",
		Test:    "TestExample",
		Passed:  true,
		Elapsed: 0.1,
	}

	display.ShowTestResult(result, true)

	// In CI mode, success tests should not produce output
	output := buf.String()
	if output != "" {
		t.Error("CI mode should not show successful tests")
	}
}

func TestTerminalDisplay_CIMode_ShowTestResult_Failure(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)
	config := &Config{CIMode: true}
	display.SetConfig(config)

	result := &TestResult{
		Package:  "example",
		Test:     "TestExample",
		Failed:   true,
		Elapsed:  0.2,
		Location: "example_test.go:15",
		Output:   []string{"    Error: assertion failed\n"},
	}

	display.ShowTestResult(result, false)

	output := buf.String()

	// Should contain FAIL but without color codes
	if !strings.Contains(output, "FAIL TestExample") {
		t.Error("CI mode should show FAIL for failed tests")
	}

	// Should not contain ANSI color codes
	if strings.Contains(output, "\033[") {
		t.Error("CI mode should not contain ANSI escape sequences")
	}

	if !strings.Contains(output, "example_test.go:15") {
		t.Error("CI mode should show location")
	}

	if !strings.Contains(output, "0.2") {
		t.Error("CI mode should show elapsed time")
	}

	if !strings.Contains(output, "Error: assertion failed") {
		t.Error("CI mode should show error output")
	}
}

func TestTerminalDisplay_CIMode_ShowPackageFailure(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)
	config := &Config{CIMode: true}
	display.SetConfig(config)

	output := []string{
		"# example [build failed]\n",
		"syntax error: unexpected token\n",
	}

	display.ShowPackageFailure("example", output)

	result := buf.String()

	// Should contain PACKAGE FAIL but without color codes
	if !strings.Contains(result, "PACKAGE FAIL example") {
		t.Error("CI mode should show PACKAGE FAIL")
	}

	// Should not contain ANSI color codes
	if strings.Contains(result, "\033[") {
		t.Error("CI mode should not contain ANSI escape sequences")
	}

	if !strings.Contains(result, "syntax error") {
		t.Error("CI mode should show error output")
	}
}

func TestTerminalDisplay_CIMode_ShowFinalResults(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, true)
	config := &Config{CIMode: true}
	display.SetConfig(config)

	packages := map[string]*PackageState{
		"example": {
			Name:   "example",
			Total:  3,
			Passed: 2,
			Failed: 1,
		},
	}

	results := map[string]*TestResult{
		"example/TestExample": {
			Package: "example",
			Test:    "TestExample",
			Failed:  true,
			Elapsed: 0.2,
		},
	}
	startTime := time.Now().Add(-2 * time.Second)

	exitCode := display.ShowFinalResults(packages, results, startTime)

	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for failed tests, got %d", exitCode)
	}

	output := buf.String()

	// Should show simple summary without colors
	if !strings.Contains(output, "Total: 3 tests") {
		t.Error("CI mode should show total test count")
	}

	if !strings.Contains(output, "Passed: 2") {
		t.Error("CI mode should show passed count")
	}

	if !strings.Contains(output, "Failed: 1") {
		t.Error("CI mode should show failed count")
	}

	if !strings.Contains(output, "Tests failed!") {
		t.Error("CI mode should show failure message")
	}

	// Should not contain ANSI color codes
	if strings.Contains(output, "\033[") {
		t.Error("CI mode should not contain ANSI escape sequences")
	}

	// Should contain failure summary in CI mode
	if !strings.Contains(output, "Failed Tests Summary") {
		t.Error("CI mode should show failure summary")
	}

	// Should not contain emojis or decorative elements
	if strings.Contains(output, "✓") || strings.Contains(output, "✗") || strings.Contains(output, "❌") {
		t.Error("CI mode should not contain decorative characters")
	}
}
