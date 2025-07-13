package broken

import "testing"

func TestBroken(t *testing.T) {
	t.Parallel()
	// This test will not be executed
	t.Log("This test will never run")
}
