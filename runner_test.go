// ABOUTME: runner_test.go contains unit tests for the Runner functionality.
// ABOUTME: It tests the main application flow, JSON event processing, and coordination between components.

package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// MockEventProcessor implements EventProcessor for testing
type MockEventProcessor struct {
	events          []TestEvent
	results         map[string]*TestResult
	packages        map[string]*PackageState
	hasTestsStarted bool
}

func NewMockEventProcessor() *MockEventProcessor {
	return &MockEventProcessor{
		events:   []TestEvent{},
		results:  make(map[string]*TestResult),
		packages: make(map[string]*PackageState),
	}
}

func (m *MockEventProcessor) ProcessEvent(event TestEvent) {
	m.events = append(m.events, event)
	// Simple mock implementation
	if event.Action == "run" {
		m.hasTestsStarted = true
		if m.packages[event.Package] == nil {
			m.packages[event.Package] = &PackageState{Name: event.Package}
		}
		key := event.Package + "/" + event.Test
		m.results[key] = &TestResult{
			Package: event.Package,
			Test:    event.Test,
			Started: true,
		}
	}
}

func (m *MockEventProcessor) GetResults() map[string]*TestResult {
	return m.results
}

func (m *MockEventProcessor) GetPackages() map[string]*PackageState {
	return m.packages
}

func (m *MockEventProcessor) HasTestsStarted() bool {
	return m.hasTestsStarted
}

// MockDisplay implements Display for testing
type MockDisplay struct {
	helpShown         bool
	progressCalls     int
	testResults       []string
	packageFailures   []string
	finalResultsCalls int
	finalExitCode     int
}

func NewMockDisplay() *MockDisplay {
	return &MockDisplay{
		testResults:     []string{},
		packageFailures: []string{},
		finalExitCode:   0,
	}
}

func (m *MockDisplay) ShowHelp() {
	m.helpShown = true
}

func (m *MockDisplay) ShowProgress(packages map[string]*PackageState, hasTestsStarted bool, startTime time.Time) {
	m.progressCalls++
}

func (m *MockDisplay) ShowTestResult(result *TestResult, success bool) {
	status := "PASS"
	if !success {
		status = "FAIL"
	}
	m.testResults = append(m.testResults, status+":"+result.Test)
}

func (m *MockDisplay) ShowPackageFailure(packageName string, output []string) {
	m.packageFailures = append(m.packageFailures, packageName)
}

func (m *MockDisplay) ShowFinalResults(packages map[string]*PackageState, results map[string]*TestResult, startTime time.Time) int {
	m.finalResultsCalls++
	return m.finalExitCode
}

func (m *MockDisplay) ClearLine() {
	// No-op for mock
}

func (m *MockDisplay) SetConfig(config *Config) {
	// No-op for mock
}

func TestRunner_Run_Success(t *testing.T) {
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestExample","Elapsed":0.1}
`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Check that events were processed
	if len(processor.events) != 2 {
		t.Errorf("Expected 2 events processed, got %d", len(processor.events))
	}

	// Check first event
	if processor.events[0].Action != "run" {
		t.Errorf("Expected first event action 'run', got '%s'", processor.events[0].Action)
	}

	// Check second event
	if processor.events[1].Action != "pass" {
		t.Errorf("Expected second event action 'pass', got '%s'", processor.events[1].Action)
	}

	// Check that test result was shown
	if len(display.testResults) != 1 {
		t.Errorf("Expected 1 test result shown, got %d", len(display.testResults))
	}

	if display.testResults[0] != "PASS:TestExample" {
		t.Errorf("Expected 'PASS:TestExample', got '%s'", display.testResults[0])
	}

	// Check that final results were called
	if display.finalResultsCalls != 1 {
		t.Errorf("Expected final results to be called once, got %d", display.finalResultsCalls)
	}
}

func TestRunner_Run_Failure(t *testing.T) {
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"fail","Package":"example","Test":"TestExample","Elapsed":0.2}
`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()
	display.finalExitCode = 1

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Check that test result was shown as failure
	if len(display.testResults) != 1 {
		t.Errorf("Expected 1 test result shown, got %d", len(display.testResults))
	}

	if display.testResults[0] != "FAIL:TestExample" {
		t.Errorf("Expected 'FAIL:TestExample', got '%s'", display.testResults[0])
	}
}

func TestRunner_Run_PackageFailure(t *testing.T) {
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"output","Package":"example","Output":"# example [build failed]\n"}
{"Time":"2023-01-01T00:00:01Z","Action":"fail","Package":"example","Elapsed":0.1}
`)

	var output bytes.Buffer
	processor := NewEventProcessor() // Use real processor for package failure logic
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	runner.Run()

	// Check that package failure was shown
	if len(display.packageFailures) != 1 {
		t.Errorf("Expected 1 package failure shown, got %d", len(display.packageFailures))
	}

	if display.packageFailures[0] != "example" {
		t.Errorf("Expected package failure for 'example', got '%s'", display.packageFailures[0])
	}
}

func TestRunner_Run_InvalidJSON(t *testing.T) {
	input := strings.NewReader(`invalid json
{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	// Should not fail due to invalid JSON - just skip it
	if exitCode != 0 {
		t.Errorf("Expected exit code 0 even with invalid JSON, got %d", exitCode)
	}

	// Should only process the valid event
	if len(processor.events) != 1 {
		t.Errorf("Expected 1 valid event processed, got %d", len(processor.events))
	}

	if processor.events[0].Action != "run" {
		t.Errorf("Expected valid event action 'run', got '%s'", processor.events[0].Action)
	}
}

func TestRunner_Run_SkipEvent(t *testing.T) {
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"skip","Package":"example","Test":"TestExample","Elapsed":0.05}
`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	runner.Run()

	// Check that test result was shown as success (skip is treated as success)
	if len(display.testResults) != 1 {
		t.Errorf("Expected 1 test result shown, got %d", len(display.testResults))
	}

	if display.testResults[0] != "PASS:TestExample" {
		t.Errorf("Expected 'PASS:TestExample' for skipped test, got '%s'", display.testResults[0])
	}
}

func TestRunner_Run_EmptyInput(t *testing.T) {
	input := strings.NewReader("")

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for empty input, got %d", exitCode)
	}

	// Should still call final results
	if display.finalResultsCalls != 1 {
		t.Errorf("Expected final results to be called once, got %d", display.finalResultsCalls)
	}

	// No events should be processed
	if len(processor.events) != 0 {
		t.Errorf("Expected 0 events processed for empty input, got %d", len(processor.events))
	}
}

func TestRunner_Run_MultipleTests(t *testing.T) {
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestA"}
{"Time":"2023-01-01T00:00:01Z","Action":"run","Package":"example","Test":"TestB"}
{"Time":"2023-01-01T00:00:02Z","Action":"pass","Package":"example","Test":"TestA","Elapsed":0.1}
{"Time":"2023-01-01T00:00:03Z","Action":"fail","Package":"example","Test":"TestB","Elapsed":0.2}
`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	runner.Run()

	// Check that both test results were shown
	if len(display.testResults) != 2 {
		t.Errorf("Expected 2 test results shown, got %d", len(display.testResults))
	}

	// Results might be in any order, so check both are present
	resultMap := make(map[string]bool)
	for _, result := range display.testResults {
		resultMap[result] = true
	}

	if !resultMap["PASS:TestA"] {
		t.Error("Expected 'PASS:TestA' to be shown")
	}

	if !resultMap["FAIL:TestB"] {
		t.Error("Expected 'FAIL:TestB' to be shown")
	}
}
