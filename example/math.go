package example

// Add returns the sum of two integers
func Add(a, b int) int {
	return a + b
}

// Subtract returns the difference of two integers
func Subtract(a, b int) int {
	return a - b
}

// Multiply returns the product of two integers
func Multiply(a, b int) int {
	// Bug: Intentionally incorrect implementation
	return a + b
}

// Divide returns the division of two integers
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, nil // Bug: Should return an error
	}
	return a / b, nil
}
