package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// MockEventProcessor is a mock implementation of EventProcessor for testing
type MockEventProcessor struct {
	events     []TestEvent
	results    map[string]*TestResult
	packages   map[string]*PackageState
	hasStarted bool
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

	if event.Test != "" {
		m.processTestEvent(event)
	}

	if event.Package != "" {
		m.processPackageEvent(event)
	}
}

func (m *MockEventProcessor) processTestEvent(event TestEvent) {
	key := event.Package + "/" + event.Test

	if event.Action == "run" {
		m.hasStarted = true
		m.results[key] = &TestResult{
			Package: event.Package,
			Test:    event.Test,
			Started: true,
		}
		return
	}

	if result, exists := m.results[key]; exists {
		result.Passed = event.Action == "pass"
		result.Failed = event.Action == "fail"
		result.Skipped = event.Action == "skip"
	}
}

func (m *MockEventProcessor) processPackageEvent(event TestEvent) {
	pkg := m.ensurePackage(event.Package)

	if event.Action == "fail" {
		pkg.Failed = 1
	}
}

func (m *MockEventProcessor) ensurePackage(name string) *PackageState {
	if _, exists := m.packages[name]; !exists {
		m.packages[name] = &PackageState{Name: name}
	}
	return m.packages[name]
}

func (m *MockEventProcessor) GetResults() map[string]*TestResult {
	return m.results
}

func (m *MockEventProcessor) GetPackages() map[string]*PackageState {
	return m.packages
}

func (m *MockEventProcessor) HasTestsStarted() bool {
	return m.hasStarted
}

// MockDisplay is a mock implementation of Display for testing
type MockDisplay struct {
	output          bytes.Buffer
	progressCalls   int
	testResults     []string
	packageFailures []string
	helpShown       bool
	lineCleared     bool
}

func NewMockDisplay() *MockDisplay {
	return &MockDisplay{}
}

func (m *MockDisplay) ShowProgress(packages map[string]*PackageState, hasTestsStarted bool, startTime time.Time) {
	m.progressCalls++
}

func (m *MockDisplay) ShowTestResult(result *TestResult, success bool) {
	m.testResults = append(m.testResults, result.Test)
}

func (m *MockDisplay) ShowPackageFailure(packageName string, output []string) {
	m.packageFailures = append(m.packageFailures, packageName)
}

func (m *MockDisplay) ShowFinalResults(packages map[string]*PackageState, results map[string]*TestResult, startTime time.Time) int {
	m.output.WriteString("Final results shown\n")
	return 0
}

func (m *MockDisplay) ShowHelp() {
	m.helpShown = true
	m.output.WriteString("Help shown\n")
}

func (m *MockDisplay) ClearLine() {
	m.lineCleared = true
}

func (m *MockDisplay) SetConfig(config *Config) {
	// Mock implementation - no operation needed
}

func TestRunner_Run_Success(t *testing.T) {
	t.Parallel()
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestExample","Elapsed":1.0}`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	if len(processor.events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(processor.events))
	}

	if !display.lineCleared {
		t.Error("Expected line to be cleared")
	}
}

func TestRunner_Run_Failure(t *testing.T) {
	t.Parallel()
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"fail","Package":"example","Test":"TestExample","Elapsed":1.0}`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	if len(display.testResults) != 1 {
		t.Errorf("Expected 1 test result shown, got %d", len(display.testResults))
	}
}

func TestRunner_Run_PackageFailure(t *testing.T) {
	t.Parallel()
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"output","Package":"example","Output":"build failed\n"}
{"Time":"2023-01-01T00:00:01Z","Action":"fail","Package":"example","Elapsed":1.0}`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)

	// Set up package state to trigger package failure display
	processor.packages["example"] = &PackageState{
		Name:   "example",
		Output: []string{"build failed\n"},
		Failed: 1,
	}

	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	if len(display.packageFailures) != 1 {
		t.Errorf("Expected 1 package failure shown, got %d", len(display.packageFailures))
	}
}

func TestRunner_Run_InvalidJSON(t *testing.T) {
	t.Parallel()
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
invalid json line
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestExample","Elapsed":1.0}`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	// Should continue processing despite invalid JSON
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Should have processed 2 valid events
	if len(processor.events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(processor.events))
	}
}

func TestRunner_Run_NonJSONInput(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("This is not JSON")

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	// Should exit with error code
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Should show help
	if !display.helpShown {
		t.Error("Expected help to be shown for non-JSON input")
	}

	// Should contain error message
	outputStr := output.String()
	if !strings.Contains(outputStr, "Input is not in JSON format") {
		t.Error("Expected error message about non-JSON input")
	}

	if !strings.Contains(outputStr, "go test -json") {
		t.Error("Expected guidance about using 'go test -json'")
	}
}

func TestRunner_Run_SkipEvent(t *testing.T) {
	t.Parallel()
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestExample"}
{"Time":"2023-01-01T00:00:01Z","Action":"skip","Package":"example","Test":"TestExample","Elapsed":0.1}`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Skip events should be shown
	if len(display.testResults) != 1 {
		t.Errorf("Expected 1 test result shown, got %d", len(display.testResults))
	}
}

func TestRunner_Run_EmptyInput(t *testing.T) {
	t.Parallel()
	input := strings.NewReader("")

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	// Should complete successfully even with empty input
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// No events should be processed
	if len(processor.events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(processor.events))
	}
}

func TestRunner_Run_MultipleTests(t *testing.T) {
	t.Parallel()
	input := strings.NewReader(`{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestA"}
{"Time":"2023-01-01T00:00:00Z","Action":"run","Package":"example","Test":"TestB"}
{"Time":"2023-01-01T00:00:01Z","Action":"pass","Package":"example","Test":"TestA","Elapsed":1.0}
{"Time":"2023-01-01T00:00:02Z","Action":"fail","Package":"example","Test":"TestB","Elapsed":2.0}`)

	var output bytes.Buffer
	processor := NewMockEventProcessor()
	display := NewMockDisplay()

	runner := NewRunner(processor, display, input, &output)
	exitCode := runner.Run()

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}

	// Should have processed 4 events
	if len(processor.events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(processor.events))
	}

	// Should show 2 test results (1 pass, 1 fail)
	if len(display.testResults) != 2 {
		t.Errorf("Expected 2 test results shown, got %d", len(display.testResults))
	}
}
