package main

import (
	"testing"
	"time"
)

func TestTimingModeConfig(t *testing.T) {
	t.Parallel()
	config := &Config{
		TimingMode: true,
		Threshold:  500 * time.Millisecond,
	}

	// Test that the config has the right values
	if !config.TimingMode {
		t.Error("expected TimingMode to be true")
	}

	if config.Threshold != 500*time.Millisecond {
		t.Errorf("expected threshold to be 500ms, got %v", config.Threshold)
	}
}

func TestTimingModeDisabled(t *testing.T) {
	t.Parallel()
	config := &Config{
		TimingMode: false,
		Threshold:  500 * time.Millisecond,
	}

	// Test that timing mode can be disabled
	if config.TimingMode {
		t.Error("expected TimingMode to be false")
	}
}
