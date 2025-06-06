// ABOUTME: processor.go contains the core logic for processing test events and managing test state.
// ABOUTME: It provides interfaces and implementations for handling JSON test events from go test.

package main

import (
	"fmt"
	"strings"
	"sync"
)

// EventProcessor processes test events and maintains state
type EventProcessor interface {
	ProcessEvent(event TestEvent)
	GetResults() map[string]*TestResult
	GetPackages() map[string]*PackageState
	HasTestsStarted() bool
}

// DefaultEventProcessor is the default implementation of EventProcessor
type DefaultEventProcessor struct {
	results         map[string]*TestResult
	packages        map[string]*PackageState
	mu              sync.RWMutex
	hasTestsStarted bool
}

// NewEventProcessor creates a new EventProcessor
func NewEventProcessor() EventProcessor {
	return &DefaultEventProcessor{
		results:  make(map[string]*TestResult),
		packages: make(map[string]*PackageState),
	}
}

// ProcessEvent processes a single test event
func (p *DefaultEventProcessor) ProcessEvent(event TestEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Initialize package if needed
	if _, exists := p.packages[event.Package]; !exists && event.Package != "" {
		p.packages[event.Package] = &PackageState{Name: event.Package}
	}

	// Process events
	if event.Test != "" {
		p.processTestEvent(event)
	} else if event.Package != "" {
		p.processPackageEvent(event)
	}
}

// GetResults returns the test results
func (p *DefaultEventProcessor) GetResults() map[string]*TestResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	results := make(map[string]*TestResult)
	for k, v := range p.results {
		results[k] = v
	}
	return results
}

// GetPackages returns the package states
func (p *DefaultEventProcessor) GetPackages() map[string]*PackageState {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	packages := make(map[string]*PackageState)
	for k, v := range p.packages {
		packages[k] = v
	}
	return packages
}

// HasTestsStarted returns whether any tests have started
func (p *DefaultEventProcessor) HasTestsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.hasTestsStarted
}

func (p *DefaultEventProcessor) processTestEvent(event TestEvent) {
	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	if _, exists := p.results[key]; !exists {
		p.results[key] = &TestResult{
			Package: event.Package,
			Test:    event.Test,
			Output:  []string{},
		}
	}
	result := p.results[key]
	pkg := p.packages[event.Package]

	// Mark parent test for subtests
	if strings.Contains(event.Test, "/") {
		parentTestName := strings.Split(event.Test, "/")[0]
		parentKey := fmt.Sprintf("%s/%s", event.Package, parentTestName)
		if parentResult, exists := p.results[parentKey]; exists {
			parentResult.HasSubtest = true
		}
	}

	switch event.Action {
	case "run":
		result.Started = true
		pkg.Running++
		p.hasTestsStarted = true
		pkg.Total++
	case "output":
		result.Output = append(result.Output, event.Output)
		if result.Location == "" {
			if location := extractFileLocation(event.Output); location != "" {
				result.Location = location
			}
		}
	case "pass":
		result.Passed = true
		result.Elapsed = event.Elapsed
		pkg.Running--
		if !p.isParentWithSubtests(event.Test, event.Package) {
			pkg.Passed++
		}
	case "fail":
		result.Failed = true
		result.Elapsed = event.Elapsed
		pkg.Running--
		if !p.isParentWithSubtests(event.Test, event.Package) {
			pkg.Failed++
			pkg.IndividualTestFailed++
		}
	case "skip":
		result.Skipped = true
		result.Elapsed = event.Elapsed
		pkg.Running--
		if !p.isParentWithSubtests(event.Test, event.Package) {
			pkg.Skipped++
		}
	}
}

func (p *DefaultEventProcessor) processPackageEvent(event TestEvent) {
	pkg := p.packages[event.Package]
	switch event.Action {
	case "output":
		pkg.Output = append(pkg.Output, event.Output)
	case "pass":
		pkg.Elapsed = event.Elapsed
	case "fail":
		pkg.Elapsed = event.Elapsed
		key := fmt.Sprintf("%s/[PACKAGE]", event.Package)
		p.results[key] = &TestResult{
			Package: event.Package,
			Test:    "[PACKAGE]",
			Failed:  true,
			Elapsed: event.Elapsed,
			Output:  pkg.Output,
		}
		if shouldDisplayPackageFailure(pkg) {
			pkg.Total++
			pkg.Failed++
		}
	}
}

func (p *DefaultEventProcessor) isParentWithSubtests(testName, packageName string) bool {
	if strings.Contains(testName, "/") {
		return false
	}
	for existingKey := range p.results {
		if strings.HasPrefix(existingKey, fmt.Sprintf("%s/%s/", packageName, testName)) {
			return true
		}
	}
	return false
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
