package libpkg

// Add returns the sum of two integers.
func Add(a, b int) int {
	return a + b
}

// Multiply returns the product of two integers.
func Multiply(a, b int) int {
	return a * b
}

// Greet returns a greeting string.
func Greet(name string) string {
	return "Hello, " + name + "!"
}

// helper is an unexported function (should NOT appear in Links).
func helper() int {
	return 42
}
