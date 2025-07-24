package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

// Mock reader that simulates slow input and can be interrupted
type interruptibleReader struct {
	events []string
	delay  time.Duration
	ctx    context.Context
}

func (r *interruptibleReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, nil
	case <-time.After(r.delay):
		if len(r.events) == 0 {
			return 0, nil
		}
		event := r.events[0] + "\n"
		r.events = r.events[1:]
		copy(p, event)
		return len(event), nil
	}
}

func TestRunnerInterruptHandling(t *testing.T) {
	testEvents := []string{
		`{"Time":"2024-01-01T00:00:00Z","Action":"run","Package":"test/pkg","Test":"TestA"}`,
		`{"Time":"2024-01-01T00:00:01Z","Action":"output","Package":"test/pkg","Test":"TestA","Output":"=== RUN   TestA\n"}`,
		`{"Time":"2024-01-01T00:00:02Z","Action":"pass","Package":"test/pkg","Test":"TestA","Elapsed":1.0}`,
		`{"Time":"2024-01-01T00:00:03Z","Action":"run","Package":"test/pkg","Test":"TestB"}`,
		`{"Time":"2024-01-01T00:00:04Z","Action":"output","Package":"test/pkg","Test":"TestB","Output":"=== RUN   TestB\n"}`,
		`{"Time":"2024-01-01T00:00:05Z","Action":"fail","Package":"test/pkg","Test":"TestB","Elapsed":1.0}`,
	}

	ctx, cancel := context.WithCancel(context.Background())
	input := &interruptibleReader{
		events: testEvents,
		delay:  10 * time.Millisecond,
		ctx:    ctx,
	}

	var output bytes.Buffer
	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)

	runner := NewRunner(processor, display, input, &output)

	// Simulate interrupt after some events are processed
	go func() {
		time.Sleep(35 * time.Millisecond) // Allow 3 events to be processed
		runner.interruptMu.Lock()
		runner.interrupted = true
		runner.interruptMu.Unlock()
		cancel()
	}()

	exitCode := runner.Run()

	outputStr := output.String()

	// Verify that interruption message is shown
	if !strings.Contains(outputStr, "Interrupted by user (Ctrl-C)") {
		t.Error("Expected interruption message not found")
	}

	// Verify that partial results are shown
	if !strings.Contains(outputStr, "Total:") {
		t.Error("Expected summary not found")
	}

	// Exit code should be based on test results, not interruption
	if exitCode != 0 && !strings.Contains(outputStr, "Failed:") {
		t.Error("Exit code should be 0 when no tests failed")
	}
}

func TestRunnerInterruptDuringInit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	input := &interruptibleReader{
		events: []string{},
		delay:  100 * time.Millisecond,
		ctx:    ctx,
	}

	var output bytes.Buffer
	processor := NewEventProcessor()
	display := NewTerminalDisplay(&output, false)

	runner := NewRunner(processor, display, input, &output)

	// Simulate immediate interrupt
	go func() {
		time.Sleep(10 * time.Millisecond)
		runner.interruptMu.Lock()
		runner.interrupted = true
		runner.interruptMu.Unlock()
		cancel()
	}()

	exitCode := runner.Run()

	outputStr := output.String()

	// Verify that interruption message is shown
	if !strings.Contains(outputStr, "Interrupted by user (Ctrl-C)") {
		t.Error("Expected interruption message not found")
	}

	// Should show empty summary
	if !strings.Contains(outputStr, "Total: 0 tests") {
		t.Error("Expected empty summary not found")
	}

	// Exit code should be 0 for empty results
	if exitCode != 0 {
		t.Error("Exit code should be 0 for empty results")
	}
}
