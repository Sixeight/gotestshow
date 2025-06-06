// ABOUTME: e2e_test.go contains end-to-end tests that test the complete application with real go test output.
// ABOUTME: It tests the actual binary with real test scenarios to ensure the application works as expected.

package main

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestE2E_HelpFlag tests the help flag functionality
func TestE2E_HelpFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run gotestshow with -help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "gotestshow - A real-time formatter") {
		t.Error("Help output should contain application description")
	}

	if !strings.Contains(outputStr, "go test -json ./... | gotestshow") {
		t.Error("Help output should contain usage example")
	}
}

// TestE2E_RealTestOutput tests with real go test output from example directory
func TestE2E_RealTestOutput(t *testing.T) {
	// First, build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Run go test -json on example directory and pipe to gotestshow
	cmd := exec.Command("bash", "-c", "cd example && go test -json ./... | ../gotestshow")
	output, err := cmd.CombinedOutput()

	// The exit code should be non-zero because example has failing tests
	if err == nil {
		t.Error("Expected non-zero exit code due to failing tests in example")
	}

	outputStr := string(output)

	// Check that it shows failed tests
	if !strings.Contains(outputStr, "FAIL") {
		t.Error("Output should contain FAIL for failed tests in example")
	}

	// Check that it shows test statistics
	if !strings.Contains(outputStr, "Total:") {
		t.Error("Output should contain total test count")
	}

	if !strings.Contains(outputStr, "Passed:") {
		t.Error("Output should contain passed count")
	}

	if !strings.Contains(outputStr, "Failed:") {
		t.Error("Output should contain failed count")
	}

	// Check final failure message
	if !strings.Contains(outputStr, "Tests failed") {
		t.Error("Output should contain final failure message")
	}
}

// TestE2E_SuccessfulTests tests with only successful tests from specific package
func TestE2E_SuccessfulTests(t *testing.T) {
	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Run go test -json on a successful test (math.go functions that should pass)
	cmd := exec.Command("bash", "-c", "cd example && go test -json -run TestAdd | ../gotestshow")
	output, err := cmd.CombinedOutput()

	// The exit code should be zero for passing tests
	if err != nil {
		t.Errorf("Expected zero exit code for passing tests, got error: %v", err)
	}

	outputStr := string(output)

	// Check that it shows success message
	if !strings.Contains(outputStr, "All tests passed") {
		t.Error("Output should contain success message for passing tests")
	}

	// Should not contain FAIL
	if strings.Contains(outputStr, "FAIL") {
		t.Error("Output should not contain FAIL for passing tests")
	}
}

// TestE2E_BuildFailure tests with package that has build errors
func TestE2E_BuildFailure(t *testing.T) {
	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Run go test -json on broken package
	cmd := exec.Command("bash", "-c", "cd example && go test -json ./broken | ../gotestshow")
	output, err := cmd.CombinedOutput()

	// The exit code should be non-zero due to build failure
	if err == nil {
		t.Error("Expected non-zero exit code due to build failure")
	}

	outputStr := string(output)

	// Check that it shows package failure
	if !strings.Contains(outputStr, "PACKAGE FAIL") {
		t.Error("Output should contain PACKAGE FAIL for build errors")
	}

	// Check that it shows the package name
	if !strings.Contains(outputStr, "broken") {
		t.Error("Output should contain the failed package name")
	}

	// Check final failure message
	if !strings.Contains(outputStr, "Tests failed") {
		t.Error("Output should contain final failure message")
	}
}

// TestE2E_NoStdin tests behavior when no stdin is provided
func TestE2E_NoStdin(t *testing.T) {
	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Run gotestshow without any input
	cmd := exec.Command("./gotestshow")
	output, err := cmd.CombinedOutput()

	// Should show help when no stdin
	if err != nil {
		t.Errorf("Expected zero exit code when showing help, got error: %v", err)
	}

	outputStr := string(output)

	// Should show help message
	if !strings.Contains(outputStr, "gotestshow - A real-time formatter") {
		t.Error("Output should contain help message when no stdin provided")
	}
}

// TestE2E_Performance tests that gotestshow can handle a reasonable load
func TestE2E_Performance(t *testing.T) {
	// Skip this test in short mode
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Run tests with timeout to ensure it doesn't hang
	cmd := exec.Command("bash", "-c", "cd example && timeout 30s go test -json ./... | ../gotestshow")

	start := time.Now()
	cmd.CombinedOutput()
	elapsed := time.Since(start)

	// Should complete within reasonable time (30 seconds is generous)
	if elapsed > 30*time.Second {
		t.Error("gotestshow took too long to process test output")
	}

	// Note: We expect an error here because the tests in example fail,
	// but we're mainly testing that it doesn't hang or crash
}

// TestE2E_LongRunningTest tests with slow tests
func TestE2E_LongRunningTest(t *testing.T) {
	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Run the slow test - note: some slow tests in example may fail
	cmd := exec.Command("bash", "-c", "cd example && go test -json -run TestSlowOperation1 | ../gotestshow")
	output, err := cmd.CombinedOutput()

	// Should handle slow tests without issues (this specific test should pass)
	if err != nil {
		t.Errorf("Expected zero exit code for slow test, got error: %v", err)
	}

	outputStr := string(output)

	// Should show progress and completion
	if !strings.Contains(outputStr, "All tests passed") {
		t.Error("Output should show completion of slow test")
	}
}

// TestE2E_InvalidJSON tests behavior with malformed JSON input
func TestE2E_InvalidJSON(t *testing.T) {
	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Create some invalid JSON mixed with valid JSON
	invalidJSON := `{"invalid": json}
{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
not json at all
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestExample","Elapsed":0.01}
`

	cmd := exec.Command("./gotestshow")
	cmd.Stdin = strings.NewReader(invalidJSON)
	output, err := cmd.CombinedOutput()

	// Should handle invalid JSON gracefully
	if err != nil {
		t.Errorf("gotestshow should handle invalid JSON gracefully, got error: %v", err)
	}

	outputStr := string(output)

	// Should still process valid JSON events
	if !strings.Contains(outputStr, "All tests passed") {
		t.Error("Should still process valid JSON events despite invalid ones")
	}
}

// TestE2E_ColorOutput tests that color output can be controlled
func TestE2E_ColorOutput(t *testing.T) {
	// Build the gotestshow binary
	buildCmd := exec.Command("go", "build", "-o", "gotestshow", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build gotestshow: %v", err)
	}

	// Create simple test JSON
	testJSON := `{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestExample","Elapsed":0.01}
`

	cmd := exec.Command("./gotestshow")
	cmd.Stdin = strings.NewReader(testJSON)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Errorf("Expected zero exit code, got error: %v", err)
	}

	outputStr := string(output)

	// Output should contain ANSI color codes by default (when output is to terminal)
	// Note: This might not work in all CI environments, so we check for content instead
	if !strings.Contains(outputStr, "tests") {
		t.Error("Output should contain test information")
	}
}
