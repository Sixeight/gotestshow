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

	// Handle build-output events with ImportPath
	if event.Action == "build-output" || event.Action == "build-fail" {
		p.processBuildEvent(event)
		return
	}

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
	result := p.ensureTestResult(key, event)
	pkg := p.packages[event.Package]

	p.markParentTestIfSubtest(event)

	switch event.Action {
	case "run":
		p.handleTestRun(result, pkg)
	case "output":
		p.handleTestOutput(result, event)
	case "pass", "fail", "skip":
		p.handleTestCompletion(result, pkg, event)
	}
}

func (p *DefaultEventProcessor) ensureTestResult(key string, event TestEvent) *TestResult {
	if _, exists := p.results[key]; !exists {
		p.results[key] = &TestResult{
			Package: event.Package,
			Test:    event.Test,
			Output:  []string{},
		}
	}
	return p.results[key]
}

func (p *DefaultEventProcessor) markParentTestIfSubtest(event TestEvent) {
	if !strings.Contains(event.Test, "/") {
		return
	}

	parentTestName := strings.Split(event.Test, "/")[0]
	parentKey := fmt.Sprintf("%s/%s", event.Package, parentTestName)
	if parentResult, exists := p.results[parentKey]; exists {
		parentResult.HasSubtest = true
	}
}

func (p *DefaultEventProcessor) handleTestRun(result *TestResult, pkg *PackageState) {
	result.Started = true
	pkg.Running++
	pkg.Total++
	p.hasTestsStarted = true
}

func (p *DefaultEventProcessor) handleTestOutput(result *TestResult, event TestEvent) {
	result.Output = append(result.Output, event.Output)
	if result.Location == "" {
		if location := extractFileLocationWithPackage(event.Output, event.Package); location != "" {
			result.Location = location
		}
	}
}

func (p *DefaultEventProcessor) handleTestCompletion(result *TestResult, pkg *PackageState, event TestEvent) {
	result.Elapsed = event.Elapsed
	pkg.Running--

	isParentWithSubtests := p.isParentWithSubtests(event.Test, event.Package)

	switch event.Action {
	case "pass":
		result.Passed = true
		if !isParentWithSubtests {
			pkg.Passed++
		}
	case "fail":
		result.Failed = true
		if !isParentWithSubtests {
			pkg.Failed++
			pkg.IndividualTestFailed++
		}
	case "skip":
		result.Skipped = true
		if !isParentWithSubtests {
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

func (p *DefaultEventProcessor) processBuildEvent(event TestEvent) {
	// Extract package name from ImportPath
	packageName := event.ImportPath
	if packageName == "" {
		return
	}

	// Remove the test binary suffix if present
	if idx := strings.Index(packageName, " ["); idx != -1 {
		packageName = packageName[:idx]
	}

	// Initialize package if needed
	if _, exists := p.packages[packageName]; !exists {
		p.packages[packageName] = &PackageState{Name: packageName}
	}

	pkg := p.packages[packageName]

	switch event.Action {
	case "build-output":
		pkg.Output = append(pkg.Output, event.Output)

		// Create a build error result if we haven't already
		key := fmt.Sprintf("%s/[BUILD]", packageName)
		if _, exists := p.results[key]; !exists {
			p.results[key] = &TestResult{
				Package: packageName,
				Test:    "[BUILD]",
				Failed:  true,
				Output:  []string{},
			}
		}

		result := p.results[key]
		result.Output = append(result.Output, event.Output)

		// Extract file location from build error
		if result.Location == "" {
			if location := extractFileLocationWithPackage(event.Output, packageName); location != "" {
				result.Location = location
			}
		}

	case "build-fail":
		// Mark package as failed due to build issues
		pkg.Failed++
		pkg.Total++
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
		// Check for both test files and regular Go files (for build errors)
		if strings.HasSuffix(parts[0], "_test.go") || strings.HasSuffix(parts[0], ".go") {
			if _, err := fmt.Sscanf(parts[1], "%d", new(int)); err == nil {
				return parts[0] + ":" + parts[1]
			}
		}
	}
	return ""
}

// extractFileLocationWithPackage extracts file:line information with package context
func extractFileLocationWithPackage(output, packageName string) string {
	trimmed := strings.TrimSpace(output)
	// Look for filename:line_number: pattern
	parts := strings.SplitN(trimmed, ":", 3)
	if len(parts) >= 2 {
		fileName := parts[0]
		lineNum := parts[1]

		// Validate line number
		if _, err := fmt.Sscanf(lineNum, "%d", new(int)); err != nil {
			return ""
		}

		// Check for both test files and regular Go files (for build errors)
		if strings.HasSuffix(fileName, "_test.go") || strings.HasSuffix(fileName, ".go") {
			// If fileName already contains path separators, it's likely a relative path
			if strings.Contains(fileName, "/") {
				return fileName + ":" + lineNum
			}

			// If we have package information, try to create a meaningful relative path
			if packageName != "" {
				// Extract relative package path from full package name
				// e.g., "github.com/Sixeight/gotestshow/example" -> "example"
				// e.g., "github.com/Sixeight/gotestshow/example/broken" -> "example/broken"
				relativePackagePath := getRelativePackagePath(packageName)
				if relativePackagePath != "" {
					return relativePackagePath + "/" + fileName + ":" + lineNum
				}
			}

			// Fallback to just filename:line
			return fileName + ":" + lineNum
		}
	}
	return ""
}

// getRelativePackagePath extracts a relative package path from a full package name
func getRelativePackagePath(fullPackageName string) string {
	parts := strings.Split(fullPackageName, "/")
	if len(parts) <= 1 {
		return ""
	}

	// Handle VCS patterns (github.com/user/repo/...)
	for i, part := range parts {
		if isVCSDomain(part) && i+3 < len(parts) {
			return strings.Join(parts[i+3:], "/")
		}
	}

	// Handle domain patterns (example.com/...)
	for i, part := range parts {
		if isDomainLike(part) && i+1 < len(parts) {
			return strings.Join(parts[i+1:], "/")
		}
	}

	// Fallback: use last 1-2 meaningful parts
	if len(parts) == 2 {
		return parts[1]
	}
	return strings.Join(parts[len(parts)-2:], "/")
}

func isVCSDomain(s string) bool {
	return s == "github.com" || s == "gitlab.com" || s == "bitbucket.org"
}

func isDomainLike(s string) bool {
	return strings.Contains(s, ".") &&
		(strings.HasSuffix(s, ".com") ||
			strings.HasSuffix(s, ".org") ||
			strings.HasSuffix(s, ".net"))
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

	// If package failed but no individual tests failed (setup/teardown errors)
	// This happens when test initialization fails before any tests can run
	if pkg.IndividualTestFailed == 0 && pkg.Failed > 0 && pkg.Total > 0 {
		return true
	}

	// If no individual tests exist (0 tests) and the package fails
	if pkg.Total == 0 && len(pkg.Output) > 0 {
		return true
	}

	// Don't display in other cases (only individual test failures)
	return false
}
