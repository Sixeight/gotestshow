package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
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

	ctx := r.setupEnvironment()
	defer r.cleanup()

	_, cancel := r.startProgressDisplay(ctx, startTime)
	defer cancel()

	if err := r.processInput(); err != nil {
		cancel()
		time.Sleep(50 * time.Millisecond)
		r.display.ClearLine()
		r.handleInputError(err)
		return 1
	}

	return r.showResults(startTime)
}

func (r *Runner) setupEnvironment() context.Context {
	if r.config != nil && r.config.CIMode {
		return context.Background()
	}

	// Hide cursor
	fmt.Fprint(r.output, "\033[?25l")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigChan
		fmt.Fprint(r.output, "\033[?25h")
		cancel()
		os.Exit(1)
	}()

	return ctx
}

func (r *Runner) cleanup() {
	if r.config == nil || !r.config.CIMode {
		fmt.Fprint(r.output, "\033[?25h")
	}
}

func (r *Runner) startProgressDisplay(ctx context.Context, startTime time.Time) (context.Context, context.CancelFunc) {
	progressCtx, cancel := context.WithCancel(ctx)
	progressRunner := NewProgressRunner(r.display, r.processor, 100*time.Millisecond)
	go progressRunner.Run(progressCtx, startTime)
	return progressCtx, cancel
}

func (r *Runner) processInput() error {
	scanner := bufio.NewScanner(r.input)
	jsonErrorCount := 0
	totalLines := 0
	validJSONFound := false

	for scanner.Scan() {
		totalLines++
		line := scanner.Bytes()

		var event TestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			jsonErrorCount++

			if totalLines == 1 && len(line) > 0 && !validJSONFound && !bytes.Contains(line, []byte("{")) {
				return fmt.Errorf("not JSON input")
			}
			continue
		}

		validJSONFound = true
		r.processor.ProcessEvent(event)
		r.displayEventResult(event)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("reading input: %w", err)
	}

	return nil
}

func (r *Runner) displayEventResult(event TestEvent) {
	switch {
	case event.Test != "" && (event.Action == "pass" || event.Action == "fail" || event.Action == "skip"):
		r.displayTestResult(event)
	case event.Action == "build-fail":
		r.displayBuildFailure(event)
	case event.Package != "" && event.Action == "fail":
		r.displayPackageFailure(event)
	}
}

func (r *Runner) displayTestResult(event TestEvent) {
	results := r.processor.GetResults()
	key := fmt.Sprintf("%s/%s", event.Package, event.Test)
	if result, exists := results[key]; exists {
		r.display.ShowTestResult(result, event.Action != "fail")
	}
}

func (r *Runner) displayBuildFailure(event TestEvent) {
	packageName := event.ImportPath
	if idx := strings.Index(packageName, " ["); idx != -1 {
		packageName = packageName[:idx]
	}
	results := r.processor.GetResults()
	key := fmt.Sprintf("%s/[BUILD]", packageName)
	if result, exists := results[key]; exists {
		r.display.ShowTestResult(result, false)
	}
}

func (r *Runner) displayPackageFailure(event TestEvent) {
	packages := r.processor.GetPackages()
	if pkg, exists := packages[event.Package]; exists && shouldDisplayPackageFailure(pkg) {
		r.display.ShowPackageFailure(event.Package, pkg.Output)
	}
}

func (r *Runner) handleInputError(err error) {
	if err.Error() == "not JSON input" {
		fmt.Fprintln(r.output, "\nError: Input is not in JSON format.")
		fmt.Fprintln(r.output, "gotestshow expects JSON output from 'go test -json'.")
		fmt.Fprintln(r.output)
		r.display.ShowHelp()
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func (r *Runner) showResults(startTime time.Time) int {
	time.Sleep(50 * time.Millisecond)
	r.display.ClearLine()
	packages := r.processor.GetPackages()
	results := r.processor.GetResults()
	return r.display.ShowFinalResults(packages, results, startTime)
}
