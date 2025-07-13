package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestShowTestResultWithTiming(t *testing.T) {
	tests := []struct {
		name           string
		result         *TestResult
		config         *Config
		expectedOutput []string
		notExpected    []string
	}{
		{
			name: "fast successful test in timing mode (should not show)",
			result: &TestResult{
				Test:    "TestFast",
				Passed:  true,
				Elapsed: 0.05,
			},
			config: &Config{
				TimingMode: true,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{},
		},
		{
			name: "slow test in timing mode",
			result: &TestResult{
				Test:    "TestSlow",
				Passed:  true,
				Elapsed: 1.5,
			},
			config: &Config{
				TimingMode: true,
				Threshold:  1 * time.Second,
			},
			expectedOutput: []string{
				"✓",
				"TestSlow",
				"(1.500s)",
				"[SLOW]",
			},
		},
		{
			name: "failed test in timing mode",
			result: &TestResult{
				Test:    "TestFailed",
				Failed:  true,
				Elapsed: 0.123,
				Output:  []string{"error: expected true, got false\n"},
			},
			config: &Config{
				TimingMode: true,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{
				"✗",
				"TestFailed",
				"(123ms)",
				"error: expected true, got false",
			},
		},
		{
			name: "skipped test in timing mode (should not show)",
			result: &TestResult{
				Test:    "TestSkipped",
				Skipped: true,
				Elapsed: 0.001,
			},
			config: &Config{
				TimingMode: true,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{},
		},
		{
			name: "fast test with location in timing mode (should not show)",
			result: &TestResult{
				Test:     "TestWithLocation",
				Passed:   true,
				Elapsed:  0.234,
				Location: "example_test.go:42",
			},
			config: &Config{
				TimingMode: true,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{},
		},
		{
			name: "test in normal mode (not timing)",
			result: &TestResult{
				Test:    "TestNormal",
				Passed:  true,
				Elapsed: 0.123,
			},
			config: &Config{
				TimingMode: false,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{},
		},
		{
			name: "slow test in timing mode",
			result: &TestResult{
				Test:    "TestSlow",
				Passed:  true,
				Elapsed: 1.5,
			},
			config: &Config{
				TimingMode: true,
				Threshold:  1 * time.Second,
			},
			expectedOutput: []string{
				"✓",
				"TestSlow",
				"(1.500s)",
				"[SLOW]",
			},
		},
		{
			name: "fast successful test in timing mode (should not show)",
			result: &TestResult{
				Test:    "TestFast",
				Passed:  true,
				Elapsed: 0.1,
			},
			config: &Config{
				TimingMode: true,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{},
		},
		{
			name: "fast failed test in timing mode (should show)",
			result: &TestResult{
				Test:    "TestFastFail",
				Failed:  true,
				Elapsed: 0.1,
				Output:  []string{"assertion failed\n"},
			},
			config: &Config{
				TimingMode: true,
				Threshold:  500 * time.Millisecond,
			},
			expectedOutput: []string{
				"✗",
				"TestFastFail",
				"(100ms)",
				"assertion failed",
			},
			notExpected: []string{
				"[SLOW]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			display := NewTerminalDisplay(&buf, false).(*TerminalDisplay)
			display.SetConfig(tt.config)

			display.ShowTestResult(tt.result, !tt.result.Failed)

			output := buf.String()

			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("expected output to contain %q, but it didn't.\nGot: %s", expected, output)
				}
			}

			for _, notExpected := range tt.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("expected output NOT to contain %q, but it did.\nGot: %s", notExpected, output)
				}
			}
		})
	}
}

func TestShowSlowTestsSummary(t *testing.T) {
	var buf bytes.Buffer
	display := NewTerminalDisplay(&buf, false).(*TerminalDisplay)
	display.SetConfig(&Config{
		TimingMode: true,
		Threshold:  500 * time.Millisecond,
	})

	results := map[string]*TestResult{
		"pkg/TestFast": {
			Test:    "TestFast",
			Passed:  true,
			Elapsed: 0.1,
		},
		"pkg/TestSlow1": {
			Test:     "TestSlow1",
			Passed:   true,
			Elapsed:  2.5,
			Location: "slow_test.go:10",
		},
		"pkg/TestSlow2": {
			Test:    "TestSlow2",
			Passed:  true,
			Elapsed: 1.2,
		},
		"pkg/TestMedium": {
			Test:    "TestMedium",
			Passed:  true,
			Elapsed: 0.4,
		},
	}

	display.showSlowTestsSummary(results)
	output := buf.String()

	expectedOutput := []string{
		"Slow Tests (>500ms)",
		"TestSlow1",
		"[slow_test.go:10]",
		"(2.500s)",
		"TestSlow2",
		"(1.200s)",
	}

	for _, expected := range expectedOutput {
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but it didn't.\nGot: %s", expected, output)
		}
	}

	// Should not contain fast or medium tests
	notExpected := []string{
		"TestFast",
		"TestMedium",
	}

	for _, ne := range notExpected {
		if strings.Contains(output, ne) {
			t.Errorf("expected output NOT to contain %q, but it did.\nGot: %s", ne, output)
		}
	}

	// Verify TestSlow1 appears before TestSlow2 (sorted by slowest first)
	slow1Index := strings.Index(output, "TestSlow1")
	slow2Index := strings.Index(output, "TestSlow2")
	if slow1Index > slow2Index {
		t.Errorf("expected TestSlow1 to appear before TestSlow2 in the output (sorted by slowest first)")
	}
}
