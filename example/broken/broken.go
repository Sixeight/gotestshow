// ABOUTME: File containing compilation errors.
// ABOUTME: Used to test package-level failures.

package broken

// This function intentionally causes a compilation error
func BrokenFunction() {
	// Reference to undefined variable
	return undefinedVariable
}
