// ABOUTME: Test file containing tests with long execution time.
// ABOUTME: Used to verify gotestshow's real-time display functionality.

package example

import (
	"testing"
	"time"
)

func TestSlowOperation1(t *testing.T) {
	t.Log("Starting slow operation 1...")
	time.Sleep(800 * time.Millisecond)
	// This test will pass
}

func TestSlowOperation2(t *testing.T) {
	t.Log("Starting slow operation 2...")
	time.Sleep(600 * time.Millisecond)
	// This test will also pass
}

func TestSlowOperation3(t *testing.T) {
	t.Log("Starting slow operation 3...")
	time.Sleep(1 * time.Second)
	// This test will fail
	t.Error("Intentional failure in slow operation 3")
}

func TestParallelSlow(t *testing.T) {
	t.Run("parallel1", func(t *testing.T) {
		t.Parallel()
		time.Sleep(700 * time.Millisecond)
	})
	t.Run("parallel2", func(t *testing.T) {
		t.Parallel()
		time.Sleep(700 * time.Millisecond)
	})
	t.Run("parallel3", func(t *testing.T) {
		t.Parallel()
		time.Sleep(700 * time.Millisecond)
	})
}
