// ABOUTME: Test for a package with compilation errors.
// ABOUTME: This test will never be executed.

package broken

import "testing"

func TestBroken(t *testing.T) {
	// This test will not be executed
	t.Log("This test will never run")
}
