// ABOUTME: runner.go contains the main application runner that coordinates event processing and display.
// ABOUTME: It handles the application lifecycle, signal handling, and orchestrates all components.

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Runner is the main application runner
type Runner struct {
	processor EventProcessor
	display   Display
	input     io.Reader
	output    io.Writer
	config    *Config
}

// NewRunner creates a new Runner instance
func NewRunner(processor EventProcessor, display Display, input io.Reader, output io.Writer) *Runner {
	return &Runner{
		processor: processor,
		display:   display,
		input:     input,
		output:    output,
	}
}

// SetConfig sets the configuration for the runner
func (r *Runner) SetConfig(config *Config) {
	r.config = config
}

// Run executes the main application logic
func (r *Runner) Run() int {
	startTime := time.Now()

	// Hide cursor
	fmt.Fprint(r.output, "\033[?25l")
	// Ensure cursor is restored when function exits
	defer fmt.Fprint(r.output, "\033[?25h")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprint(r.output, "\033[?25h") // Show cursor
		os.Exit(1)
	}()

	// Start progress display goroutine
	ctx, cancel := context.WithCancel(context.Background())

	progressRunner := NewProgressRunner(r.display, r.processor, 100*time.Millisecond)
	go progressRunner.Run(ctx, startTime)

	// Process JSON events from input
	scanner := bufio.NewScanner(r.input)
	jsonErrorCount := 0
	totalLines := 0

	for scanner.Scan() {
		totalLines++
		line := scanner.Bytes()

		var event TestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			jsonErrorCount++

			// If first line is not JSON, it's likely not JSON input at all
			if totalLines == 1 && len(line) > 0 {
				// Stop progress display
				cancel()
				time.Sleep(50 * time.Millisecond)
				r.display.ClearLine()

				// Show help message
				fmt.Fprintln(r.output, "\nError: Input is not in JSON format.")
				fmt.Fprintln(r.output, "gotestshow expects JSON output from 'go test -json'.")
				fmt.Fprintln(r.output)
				r.display.ShowHelp()
				return 1
			}
			continue
		}

		r.processor.ProcessEvent(event)

		// Display test results immediately
		if event.Test != "" && (event.Action == "pass" || event.Action == "fail" || event.Action == "skip") {
			results := r.processor.GetResults()
			key := fmt.Sprintf("%s/%s", event.Package, event.Test)
			if result, exists := results[key]; exists {
				r.display.ShowTestResult(result, event.Action != "fail")
			}
		} else if event.Package != "" && event.Action == "fail" {
			// Display package failures immediately
			packages := r.processor.GetPackages()
			if pkg, exists := packages[event.Package]; exists && shouldDisplayPackageFailure(pkg) {
				r.display.ShowPackageFailure(event.Package, pkg.Output)
			}
		}
	}

	// Stop progress display immediately after scanner loop ends
	cancel()

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		return 1
	}

	// Wait a bit for progress goroutine to stop
	time.Sleep(50 * time.Millisecond)

	// Clear progress and show final results
	r.display.ClearLine()
	packages := r.processor.GetPackages()
	results := r.processor.GetResults()
	return r.display.ShowFinalResults(packages, results, startTime)
}
