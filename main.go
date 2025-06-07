// ABOUTME: gotestshow is a CLI tool that parses and formats the output of `go test -json`.
// ABOUTME: It receives JSON-formatted test results from stdin and displays them in a readable format in real-time.

package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// TestEvent represents a single test event from go test -json
type TestEvent struct {
	Time    time.Time `json:"Time"`
	Action  string    `json:"Action"`
	Package string    `json:"Package"`
	Test    string    `json:"Test"`
	Elapsed float64   `json:"Elapsed"`
	Output  string    `json:"Output"`
}

// TestResult holds the summary of a test
type TestResult struct {
	Package    string
	Test       string
	Passed     bool
	Skipped    bool
	Failed     bool
	Elapsed    float64
	Output     []string
	Started    bool
	Location   string // File name and line number (e.g., "math_test.go:47")
	HasSubtest bool   // Whether this test has subtests
}

// PackageState tracks the state of tests in a package
type PackageState struct {
	Name                 string
	Total                int
	Passed               int
	Failed               int
	Skipped              int
	Running              int
	Elapsed              float64
	Output               []string // Store package-level output
	IndividualTestFailed int      // Number of individual test failures
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	clearLine   = "\r"
)

type summaryStats struct {
	totalTests   int
	totalPassed  int
	totalFailed  int
	totalSkipped int
	hasFailures  bool
}

// Config holds the configuration for gotestshow
type Config struct {
	TimingMode bool
	Threshold  time.Duration
}

func main() {
	help := flag.Bool("help", false, "Show help message")
	timing := flag.Bool("timing", false, "Enable timing mode to show only slow tests and failures")
	threshold := flag.String("threshold", "500ms", "Threshold for slow tests (e.g., 1s, 500ms)")
	flag.Parse()

	// Parse threshold duration
	thresholdDuration, err := time.ParseDuration(*threshold)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid threshold format: %s\n", err)
		os.Exit(1)
	}

	config := &Config{
		TimingMode: *timing,
		Threshold:  thresholdDuration,
	}

	display := NewTerminalDisplay(os.Stdout, true)
	display.SetConfig(config)

	if *help {
		display.ShowHelp()
		os.Exit(0)
	}

	// Check if stdin has input
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// No input from pipe
		display.ShowHelp()
		os.Exit(0)
	}

	processor := NewEventProcessor()
	runner := NewRunner(processor, display, os.Stdin, os.Stdout)
	runner.SetConfig(config)
	exitCode := runner.Run()
	os.Exit(exitCode)
}
