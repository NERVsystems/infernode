package main

import "fmt"

func main() {
	// Basic AND NOT
	a := 0xFF
	b := 0x0F
	fmt.Println(a &^ b) // 0xFF &^ 0x0F = 0xF0 = 240

	// Clear specific bits
	flags := 7 // binary 111
	mask := 2  // binary 010
	fmt.Println(flags &^ mask) // 5 (binary 101)

	// AND NOT with zero (no-op)
	fmt.Println(42 &^ 0) // 42

	// AND NOT with all ones
	fmt.Println(42 &^ -1) // 0

	// Use in expression
	x := 0xAB
	y := 0xA0
	z := x &^ y
	fmt.Println(z) // 0x0B = 11
}
