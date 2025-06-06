// ABOUTME: integration_test.go contains integration tests that test the complete application flow.
// ABOUTME: It tests the interaction between EventProcessor, Display, and Runner components working together.

package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestIntegration_CompleteWorkflow tests the complete workflow from JSON input to final output
func TestIntegration_CompleteWorkflow(t *testing.T) {
	// Sample JSON input that represents a real go test -json output
	jsonInput := `{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestMath"}
{"Time":"2023-01-01T00:00:01Z","Action":"output","Package":"example","Test":"TestMath","Output":"=== RUN   TestMath\n"}
{"Time":"2023-01-01T00:00:02Z","Action":"output","Package":"example","Test":"TestMath","Output":"    math_test.go:10: 2 + 2 should equal 4\n"}
{"Time":"2023-01-01T00:00:03Z","Action":"pass","Package":"example","Test":"TestMath","Elapsed":0.01}
{"Time":"2023-01-01T00:00:04Z","Action":"run","Package":"example","Test":"TestFail"}
{"Time":"2023-01-01T00:00:05Z","Action":"output","Package":"example","Test":"TestFail","Output":"=== RUN   TestFail\n"}
{"Time":"2023-01-01T00:00:06Z","Action":"output","Package":"example","Test":"TestFail","Output":"    math_test.go:20: assertion failed\n"}
{"Time":"2023-01-01T00:00:07Z","Action":"output","Package":"example","Test":"TestFail","Output":"        expected: 5\n"}
{"Time":"2023-01-01T00:00:08Z","Action":"output","Package":"example","Test":"TestFail","Output":"        actual: 4\n"}
{"Time":"2023-01-01T00:00:09Z","Action":"fail","Package":"example","Test":"TestFail","Elapsed":0.02}
{"Time":"2023-01-01T00:00:10Z","Action":"output","Package":"example","Output":"FAIL\n"}
{"Time":"2023-01-01T00:00:11Z","Action":"fail","Package":"example","Elapsed":0.03}
`

	input := strings.NewReader(jsonInput)
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false) // Disable colors for easier testing
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 1 due to test failure
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for failed tests, got %d", exitCode)
	}

	outputStr := output.String()

	// Check that failed test is displayed
	if !strings.Contains(outputStr, "FAIL") {
		t.Error("Output should contain FAIL for failed test")
	}

	if !strings.Contains(outputStr, "TestFail") {
		t.Error("Output should contain the failed test name")
	}

	if !strings.Contains(outputStr, "math_test.go:20") {
		t.Error("Output should contain the test file location")
	}

	if !strings.Contains(outputStr, "assertion failed") {
		t.Error("Output should contain the failure message")
	}

	// Check final summary
	if !strings.Contains(outputStr, "Tests failed") {
		t.Error("Output should contain final failure message")
	}

	if !strings.Contains(outputStr, "Total: 2 tests") {
		t.Error("Output should contain total test count")
	}

	if !strings.Contains(outputStr, "Passed: 1") {
		t.Error("Output should contain passed count")
	}

	if !strings.Contains(outputStr, "Failed: 1") {
		t.Error("Output should contain failed count")
	}
}

// TestIntegration_SuccessfulTests tests the workflow with all passing tests
func TestIntegration_SuccessfulTests(t *testing.T) {
	jsonInput := `{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestA"}
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestA","Elapsed":0.01}
{"Time":"2023-01-01T00:00:02Z","Action":"run","Package":"example","Test":"TestB"}
{"Time":"2023-01-01T00:00:03Z","Action":"pass","Package":"example","Test":"TestB","Elapsed":0.02}
{"Time":"2023-01-01T00:00:04Z","Action":"output","Package":"example","Output":"PASS\n"}
{"Time":"2023-01-01T00:00:05Z","Action":"pass","Package":"example","Elapsed":0.05}
`

	input := strings.NewReader(jsonInput)
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 0 for successful tests
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for successful tests, got %d", exitCode)
	}

	outputStr := output.String()

	// Check final success message
	if !strings.Contains(outputStr, "All tests passed") {
		t.Error("Output should contain success message")
	}

	if !strings.Contains(outputStr, "Total: 2 tests") {
		t.Error("Output should contain total test count")
	}

	if !strings.Contains(outputStr, "Passed: 2") {
		t.Error("Output should contain passed count")
	}

	if !strings.Contains(outputStr, "Failed: 0") {
		t.Error("Output should contain failed count of 0")
	}
}

// TestIntegration_SkippedTests tests the workflow with skipped tests
func TestIntegration_SkippedTests(t *testing.T) {
	jsonInput := `{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestSkipped"}
{"Time":"2023-01-01T00:00:01Z","Action":"output","Package":"example","Test":"TestSkipped","Output":"=== RUN   TestSkipped\n"}
{"Time":"2023-01-01T00:00:02Z","Action":"output","Package":"example","Test":"TestSkipped","Output":"--- SKIP: TestSkipped (0.00s)\n"}
{"Time":"2023-01-01T00:00:03Z","Action":"skip","Package":"example","Test":"TestSkipped","Elapsed":0.00}
{"Time":"2023-01-01T00:00:04Z","Action":"output","Package":"example","Output":"PASS\n"}
{"Time":"2023-01-01T00:00:05Z","Action":"pass","Package":"example","Elapsed":0.01}
`

	input := strings.NewReader(jsonInput)
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 0 for skipped tests (not failures)
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for skipped tests, got %d", exitCode)
	}

	outputStr := output.String()

	if !strings.Contains(outputStr, "All tests passed") {
		t.Error("Output should contain success message for skipped tests")
	}

	if !strings.Contains(outputStr, "Skipped: 1") {
		t.Error("Output should contain skipped count")
	}
}

// TestIntegration_PackageBuildFailure tests the workflow with package build failures
func TestIntegration_PackageBuildFailure(t *testing.T) {
	jsonInput := `{"Time":"2023-01-01T00:00:00Z","Action":"output","Package":"broken","Output":"# broken [build failed]\n"}
{"Time":"2023-01-01T00:00:01Z","Action":"output","Package":"broken","Output":"syntax error: unexpected '}' at end of statement\n"}
{"Time":"2023-01-01T00:00:02Z","Action":"fail","Package":"broken","Elapsed":0.1}
`

	input := strings.NewReader(jsonInput)
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 1 for build failures
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for build failures, got %d", exitCode)
	}

	outputStr := output.String()

	// Check that package failure is displayed
	if !strings.Contains(outputStr, "PACKAGE FAIL") {
		t.Error("Output should contain PACKAGE FAIL for build failures")
	}

	if !strings.Contains(outputStr, "broken") {
		t.Error("Output should contain the failed package name")
	}

	if !strings.Contains(outputStr, "syntax error") {
		t.Error("Output should contain the build error message")
	}
}

// TestIntegration_Subtests tests the workflow with subtests
func TestIntegration_Subtests(t *testing.T) {
	jsonInput := `{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestParent"}
{"Time":"2023-01-01T00:00:01Z","Action":"run","Package":"example","Test":"TestParent/SubA"}
{"Time":"2023-01-01T00:00:02Z","Action":"pass","Package":"example","Test":"TestParent/SubA","Elapsed":0.01}
{"Time":"2023-01-01T00:00:03Z","Action":"run","Package":"example","Test":"TestParent/SubB"}
{"Time":"2023-01-01T00:00:04Z","Action":"output","Package":"example","Test":"TestParent/SubB","Output":"    test_test.go:25: subtest failed\n"}
{"Time":"2023-01-01T00:00:05Z","Action":"fail","Package":"example","Test":"TestParent/SubB","Elapsed":0.02}
{"Time":"2023-01-01T00:00:06Z","Action":"fail","Package":"example","Test":"TestParent","Elapsed":0.03}
{"Time":"2023-01-01T00:00:07Z","Action":"output","Package":"example","Output":"FAIL\n"}
{"Time":"2023-01-01T00:00:08Z","Action":"fail","Package":"example","Elapsed":0.05}
`

	input := strings.NewReader(jsonInput)
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 1 due to subtest failure
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for subtest failures, got %d", exitCode)
	}

	outputStr := output.String()

	// Check that failed subtest is displayed but not parent test
	if !strings.Contains(outputStr, "TestParent/SubB") {
		t.Error("Output should contain the failed subtest name")
	}

	// Parent test should not be shown in detail (only subtest failures)
	failLines := strings.Split(outputStr, "\n")
	parentFailCount := 0
	for _, line := range failLines {
		if strings.Contains(line, "FAIL") && strings.Contains(line, "TestParent") && !strings.Contains(line, "/") {
			parentFailCount++
		}
	}

	// Parent test failure should not be displayed as individual failure
	if parentFailCount > 1 {
		t.Error("Parent test with subtests should not show detailed failure output")
	}
}

// TestIntegration_MultiplePackages tests the workflow with multiple packages
func TestIntegration_MultiplePackages(t *testing.T) {
	jsonInput := `{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"pkg1","Test":"TestPkg1"}
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"pkg1","Test":"TestPkg1","Elapsed":0.01}
{"Time":"2023-01-01T00:00:02Z","Action":"pass","Package":"pkg1","Elapsed":0.02}
{"Time":"2023-01-01T00:00:03Z","Action":"run","Package":"pkg2","Test":"TestPkg2"}
{"Time":"2023-01-01T00:00:04Z","Action":"output","Package":"pkg2","Test":"TestPkg2","Output":"    pkg2_test.go:15: test failed\n"}
{"Time":"2023-01-01T00:00:05Z","Action":"fail","Package":"pkg2","Test":"TestPkg2","Elapsed":0.02}
{"Time":"2023-01-01T00:00:06Z","Action":"fail","Package":"pkg2","Elapsed":0.03}
`

	input := strings.NewReader(jsonInput)
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 1 due to pkg2 failure
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 for package failures, got %d", exitCode)
	}

	outputStr := output.String()

	// Check that failed test from pkg2 is displayed
	if !strings.Contains(outputStr, "TestPkg2") {
		t.Error("Output should contain the failed test from pkg2")
	}

	if !strings.Contains(outputStr, "pkg2_test.go:15") {
		t.Error("Output should contain the failure location")
	}

	// Check final summary includes both packages
	if !strings.Contains(outputStr, "Total: 2 tests") {
		t.Error("Output should contain total test count from both packages")
	}

	if !strings.Contains(outputStr, "Passed: 1") {
		t.Error("Output should contain passed count")
	}

	if !strings.Contains(outputStr, "Failed: 1") {
		t.Error("Output should contain failed count")
	}
}

// TestIntegration_EmptyOutput tests the workflow with no test events
func TestIntegration_EmptyOutput(t *testing.T) {
	input := strings.NewReader("")
	var output bytes.Buffer

	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)
	runner := NewRunner(processor, display, input, &output)

	exitCode := runner.Run()

	// Should exit with code 0 for no tests
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for no tests, got %d", exitCode)
	}

	outputStr := output.String()

	// Should still show final summary even with no tests
	if !strings.Contains(outputStr, "Total: 0 tests") {
		t.Error("Output should contain total test count of 0")
	}

	if !strings.Contains(outputStr, "All tests passed") {
		t.Error("Output should contain success message when no tests to fail")
	}
}
