package main

import (
	"testing"
	"time"
)

func TestConfigParsing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		threshold        string
		expectedDuration time.Duration
		shouldError      bool
	}{
		{
			name:             "default threshold",
			threshold:        "500ms",
			expectedDuration: 500 * time.Millisecond,
			shouldError:      false,
		},
		{
			name:             "1 second threshold",
			threshold:        "1s",
			expectedDuration: 1 * time.Second,
			shouldError:      false,
		},
		{
			name:             "1.5 second threshold",
			threshold:        "1.5s",
			expectedDuration: 1500 * time.Millisecond,
			shouldError:      false,
		},
		{
			name:             "200 millisecond threshold",
			threshold:        "200ms",
			expectedDuration: 200 * time.Millisecond,
			shouldError:      false,
		},
		{
			name:             "invalid threshold format",
			threshold:        "invalid",
			expectedDuration: 0,
			shouldError:      true,
		},
		{
			name:             "negative threshold",
			threshold:        "-1s",
			expectedDuration: -1 * time.Second,
			shouldError:      false, // time.ParseDuration allows negative values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			duration, err := time.ParseDuration(tt.threshold)

			if tt.shouldError && err == nil {
				t.Errorf("expected error but got nil")
			}

			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.shouldError && duration != tt.expectedDuration {
				t.Errorf("expected duration %v, got %v", tt.expectedDuration, duration)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		seconds  float64
		expected string
	}{
		{
			name:     "milliseconds under 1 second",
			seconds:  0.123,
			expected: "123ms",
		},
		{
			name:     "milliseconds under 1 second rounded",
			seconds:  0.5678,
			expected: "568ms",
		},
		{
			name:     "exactly 1 second",
			seconds:  1.0,
			expected: "1.000s",
		},
		{
			name:     "seconds with decimals",
			seconds:  2.345,
			expected: "2.345s",
		},
		{
			name:     "very small milliseconds",
			seconds:  0.001,
			expected: "1ms",
		},
		{
			name:     "zero duration",
			seconds:  0.0,
			expected: "0ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatDuration(tt.seconds)
			if result != tt.expected {
				t.Errorf("formatDuration(%f) = %s, want %s", tt.seconds, result, tt.expected)
			}
		})
	}
}
