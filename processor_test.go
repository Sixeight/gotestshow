// ABOUTME: processor_test.go contains unit tests for the EventProcessor functionality.
// ABOUTME: It tests event processing, state management, and result tracking.

package main

import (
	"testing"
	"time"
)

func TestEventProcessor_ProcessEvent(t *testing.T) {
	processor := NewEventProcessor()

	// Test "run" action
	event := TestEvent{
		Time:    time.Now(),
		Action:  "run",
		Package: "example",
		Test:    "TestExample",
		Elapsed: 0,
		Output:  "",
	}

	processor.ProcessEvent(event)

	packages := processor.GetPackages()
	results := processor.GetResults()

	if len(packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(packages))
	}

	pkg := packages["example"]
	if pkg.Running != 1 {
		t.Errorf("Expected 1 running test, got %d", pkg.Running)
	}

	if pkg.Total != 1 {
		t.Errorf("Expected 1 total test, got %d", pkg.Total)
	}

	if !processor.HasTestsStarted() {
		t.Error("Expected tests to have started")
	}

	key := "example/TestExample"
	result := results[key]
	if result == nil {
		t.Fatal("Test result not found")
	}

	if !result.Started {
		t.Error("Expected test to be marked as started")
	}
}

func TestEventProcessor_ProcessEvent_Pass(t *testing.T) {
	processor := NewEventProcessor()

	// First start the test
	processor.ProcessEvent(TestEvent{
		Action:  "run",
		Package: "example",
		Test:    "TestExample",
	})

	// Then pass the test
	processor.ProcessEvent(TestEvent{
		Action:  "pass",
		Package: "example",
		Test:    "TestExample",
		Elapsed: 0.1,
	})

	packages := processor.GetPackages()
	results := processor.GetResults()

	pkg := packages["example"]
	if pkg.Running != 0 {
		t.Errorf("Expected 0 running tests, got %d", pkg.Running)
	}

	if pkg.Passed != 1 {
		t.Errorf("Expected 1 passed test, got %d", pkg.Passed)
	}

	key := "example/TestExample"
	result := results[key]
	if !result.Passed {
		t.Error("Expected test to be marked as passed")
	}

	if result.Elapsed != 0.1 {
		t.Errorf("Expected elapsed time 0.1, got %f", result.Elapsed)
	}
}

func TestEventProcessor_ProcessEvent_Fail(t *testing.T) {
	processor := NewEventProcessor()

	// First start the test
	processor.ProcessEvent(TestEvent{
		Action:  "run",
		Package: "example",
		Test:    "TestExample",
	})

	// Then fail the test
	processor.ProcessEvent(TestEvent{
		Action:  "fail",
		Package: "example",
		Test:    "TestExample",
		Elapsed: 0.2,
	})

	packages := processor.GetPackages()
	results := processor.GetResults()

	pkg := packages["example"]
	if pkg.Running != 0 {
		t.Errorf("Expected 0 running tests, got %d", pkg.Running)
	}

	if pkg.Failed != 1 {
		t.Errorf("Expected 1 failed test, got %d", pkg.Failed)
	}

	if pkg.IndividualTestFailed != 1 {
		t.Errorf("Expected 1 individual test failed, got %d", pkg.IndividualTestFailed)
	}

	key := "example/TestExample"
	result := results[key]
	if !result.Failed {
		t.Error("Expected test to be marked as failed")
	}

	if result.Elapsed != 0.2 {
		t.Errorf("Expected elapsed time 0.2, got %f", result.Elapsed)
	}
}

func TestEventProcessor_ProcessEvent_Skip(t *testing.T) {
	processor := NewEventProcessor()

	// First start the test
	processor.ProcessEvent(TestEvent{
		Action:  "run",
		Package: "example",
		Test:    "TestExample",
	})

	// Then skip the test
	processor.ProcessEvent(TestEvent{
		Action:  "skip",
		Package: "example",
		Test:    "TestExample",
		Elapsed: 0.05,
	})

	packages := processor.GetPackages()
	results := processor.GetResults()

	pkg := packages["example"]
	if pkg.Running != 0 {
		t.Errorf("Expected 0 running tests, got %d", pkg.Running)
	}

	if pkg.Skipped != 1 {
		t.Errorf("Expected 1 skipped test, got %d", pkg.Skipped)
	}

	key := "example/TestExample"
	result := results[key]
	if !result.Skipped {
		t.Error("Expected test to be marked as skipped")
	}

	if result.Elapsed != 0.05 {
		t.Errorf("Expected elapsed time 0.05, got %f", result.Elapsed)
	}
}

func TestEventProcessor_ProcessEvent_Output(t *testing.T) {
	processor := NewEventProcessor()

	// First start the test
	processor.ProcessEvent(TestEvent{
		Action:  "run",
		Package: "example",
		Test:    "TestExample",
	})

	// Add output
	processor.ProcessEvent(TestEvent{
		Action:  "output",
		Package: "example",
		Test:    "TestExample",
		Output:  "    example_test.go:10: Error message\n",
	})

	results := processor.GetResults()
	key := "example/TestExample"
	result := results[key]

	if len(result.Output) != 1 {
		t.Errorf("Expected 1 output line, got %d", len(result.Output))
	}

	if result.Output[0] != "    example_test.go:10: Error message\n" {
		t.Errorf("Unexpected output: %s", result.Output[0])
	}

	if result.Location != "example_test.go:10" {
		t.Errorf("Expected location 'example_test.go:10', got '%s'", result.Location)
	}
}

func TestEventProcessor_ProcessEvent_Subtests(t *testing.T) {
	processor := NewEventProcessor()

	// Start parent test
	processor.ProcessEvent(TestEvent{
		Action:  "run",
		Package: "example",
		Test:    "TestParent",
	})

	// Start subtest
	processor.ProcessEvent(TestEvent{
		Action:  "run",
		Package: "example",
		Test:    "TestParent/SubTest1",
	})

	// Fail subtest
	processor.ProcessEvent(TestEvent{
		Action:  "fail",
		Package: "example",
		Test:    "TestParent/SubTest1",
		Elapsed: 0.1,
	})

	// Fail parent test (should not count as individual test failure)
	processor.ProcessEvent(TestEvent{
		Action:  "fail",
		Package: "example",
		Test:    "TestParent",
		Elapsed: 0.2,
	})

	packages := processor.GetPackages()
	results := processor.GetResults()

	pkg := packages["example"]
	if pkg.Failed != 1 {
		t.Errorf("Expected 1 failed test (only subtest), got %d", pkg.Failed)
	}

	parentKey := "example/TestParent"
	parentResult := results[parentKey]
	if !parentResult.HasSubtest {
		t.Error("Expected parent test to be marked as having subtests")
	}

	subtestKey := "example/TestParent/SubTest1"
	subtestResult := results[subtestKey]
	if !subtestResult.Failed {
		t.Error("Expected subtest to be marked as failed")
	}
}

func TestEventProcessor_ProcessPackageEvent(t *testing.T) {
	processor := NewEventProcessor()

	// Package output
	processor.ProcessEvent(TestEvent{
		Action:  "output",
		Package: "example",
		Test:    "",
		Output:  "# example [example.test]\n",
	})

	// Package pass
	processor.ProcessEvent(TestEvent{
		Action:  "pass",
		Package: "example",
		Test:    "",
		Elapsed: 1.5,
	})

	packages := processor.GetPackages()
	pkg := packages["example"]

	if len(pkg.Output) != 1 {
		t.Errorf("Expected 1 package output line, got %d", len(pkg.Output))
	}

	if pkg.Elapsed != 1.5 {
		t.Errorf("Expected package elapsed time 1.5, got %f", pkg.Elapsed)
	}
}

func TestEventProcessor_ProcessPackageEvent_Fail(t *testing.T) {
	processor := NewEventProcessor()

	// Package output with build error
	processor.ProcessEvent(TestEvent{
		Action:  "output",
		Package: "example",
		Test:    "",
		Output:  "# example [build failed]\n",
	})

	// Package fail
	processor.ProcessEvent(TestEvent{
		Action:  "fail",
		Package: "example",
		Test:    "",
		Elapsed: 0.5,
	})

	packages := processor.GetPackages()
	results := processor.GetResults()

	pkg := packages["example"]
	if pkg.Total != 1 {
		t.Errorf("Expected 1 total (package failure), got %d", pkg.Total)
	}

	if pkg.Failed != 1 {
		t.Errorf("Expected 1 failed (package failure), got %d", pkg.Failed)
	}

	packageKey := "example/[PACKAGE]"
	packageResult := results[packageKey]
	if packageResult == nil {
		t.Fatal("Package result not found")
	}

	if !packageResult.Failed {
		t.Error("Expected package result to be marked as failed")
	}
}

func TestExtractFileLocation(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"    math_test.go:15: Error occurred", "math_test.go:15"},
		{"        example_test.go:42: assertion failed", "example_test.go:42"},
		{"not a file location", ""},
		{"regular.go:10: not a test file", "regular.go:10"},
		{"math_test.go:abc: invalid line number", ""},
		{"math_test.go: missing line number", ""},
	}

	for _, test := range tests {
		result := extractFileLocation(test.input)
		if result != test.expected {
			t.Errorf("extractFileLocation(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestExtractFileLocationWithPackage(t *testing.T) {
	tests := []struct {
		input       string
		packageName string
		expected    string
	}{
		{"    math_test.go:15: Error occurred", "github.com/Sixeight/gotestshow/example", "example/math_test.go:15"},
		{"        example_test.go:42: assertion failed", "github.com/Sixeight/gotestshow/example/broken", "example/broken/example_test.go:42"},
		{"example/broken/broken.go:9:9: undefined: undefinedVariable", "github.com/Sixeight/gotestshow/example/broken", "example/broken/broken.go:9"},
		{"not a file location", "github.com/Sixeight/gotestshow/example", ""},
		{"math_test.go:abc: invalid line number", "github.com/Sixeight/gotestshow/example", ""},
		{"math_test.go: missing line number", "github.com/Sixeight/gotestshow/example", ""},
		// Already relative path should be preserved
		{"pkg/service/handler_test.go:23: error", "github.com/company/project/pkg/service", "pkg/service/handler_test.go:23"},
		// Without package context
		{"handler_test.go:23: error", "", "handler_test.go:23"},
	}

	for _, test := range tests {
		result := extractFileLocationWithPackage(test.input, test.packageName)
		if result != test.expected {
			t.Errorf("extractFileLocationWithPackage(%q, %q) = %q, expected %q", test.input, test.packageName, result, test.expected)
		}
	}
}

func TestGetRelativePackagePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"github.com/Sixeight/gotestshow/example", "example"},
		{"github.com/Sixeight/gotestshow/example/broken", "example/broken"},
		{"github.com/company/project/pkg/service", "pkg/service"},
		{"gitlab.com/user/repo/internal/api", "internal/api"},
		{"simple/package", "package"},
		{"standalone", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := getRelativePackagePath(test.input)
		if result != test.expected {
			t.Errorf("getRelativePackagePath(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestShouldDisplayPackageFailure(t *testing.T) {
	tests := []struct {
		name     string
		pkg      *PackageState
		expected bool
	}{
		{
			name: "build failed",
			pkg: &PackageState{
				Total:  0,
				Output: []string{"# example [build failed]"},
			},
			expected: true,
		},
		{
			name: "syntax error",
			pkg: &PackageState{
				Total:  0,
				Output: []string{"syntax error: unexpected token"},
			},
			expected: true,
		},
		{
			name: "no buildable files",
			pkg: &PackageState{
				Total:  0,
				Output: []string{"no buildable Go source files in example"},
			},
			expected: true,
		},
		{
			name: "individual test failure",
			pkg: &PackageState{
				Total:                1,
				Failed:               1,
				IndividualTestFailed: 1,
				Output:               []string{"=== RUN TestExample", "--- FAIL: TestExample"},
			},
			expected: false,
		},
		{
			name: "successful package",
			pkg: &PackageState{
				Total:  5,
				Passed: 5,
				Output: []string{"PASS"},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shouldDisplayPackageFailure(test.pkg)
			if result != test.expected {
				t.Errorf("shouldDisplayPackageFailure() = %v, expected %v", result, test.expected)
			}
		})
	}
}
