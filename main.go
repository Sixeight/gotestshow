package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// TestEvent represents a single test event from go test -json
type TestEvent struct {
	Time       time.Time `json:"Time"`
	Action     string    `json:"Action"`
	Package    string    `json:"Package"`
	Test       string    `json:"Test"`
	Elapsed    float64   `json:"Elapsed"`
	Output     string    `json:"Output"`
	ImportPath string    `json:"ImportPath"`
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
	CIMode     bool
}

func parseConfig() (*Config, error) {
	help := flag.Bool("help", false, "Show help message")
	timing := flag.Bool("timing", false, "Enable timing mode to show only slow tests and failures")
	threshold := flag.String("threshold", "500ms", "Threshold for slow tests (e.g., 1s, 500ms)")
	ci := flag.Bool("ci", false, "Enable CI mode - no escape sequences, only show failures and summary")
	flag.Parse()

	if *help {
		return nil, fmt.Errorf("help requested")
	}

	thresholdDuration, err := time.ParseDuration(*threshold)
	if err != nil {
		return nil, fmt.Errorf("invalid threshold format: %w", err)
	}

	return &Config{
		TimingMode: *timing,
		Threshold:  thresholdDuration,
		CIMode:     *ci,
	}, nil
}

func hasStdinInput() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func main() {
	config, err := parseConfig()
	if err != nil {
		if err.Error() == "help requested" {
			display := NewTerminalDisplay(os.Stdout, true)
			display.ShowHelp()
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	display := NewTerminalDisplay(os.Stdout, true)
	display.SetConfig(config)

	if !hasStdinInput() {
		display.ShowHelp()
		os.Exit(0)
	}

	processor := NewEventProcessor()
	runner := NewRunner(processor, display, os.Stdin, os.Stdout)
	runner.SetConfig(config)
	os.Exit(runner.Run())
}
