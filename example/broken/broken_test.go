package broken

import "testing"

func TestBroken(t *testing.T) {
	// This test will not be executed
	t.Log("This test will never run")
}
