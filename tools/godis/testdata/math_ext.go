package main

import "math"

func main() {
	// Floor
	println(math.Floor(3.7))  // 3
	println(math.Floor(-2.3)) // -3
	println(math.Floor(5.0))  // 5

	// Ceil
	println(math.Ceil(3.2))  // 4
	println(math.Ceil(-2.7)) // -2
	println(math.Ceil(5.0))  // 5

	// Round
	println(math.Round(3.5))  // 4
	println(math.Round(3.4))  // 3
	println(math.Round(-2.5)) // -3

	// IsNaN
	nan := math.NaN()
	if math.IsNaN(nan) {
		println("nan")
	}
	if !math.IsNaN(3.14) {
		println("not nan")
	}

	// Pow (integer exponent)
	println(math.Pow(2.0, 10.0)) // 1024
	println(math.Pow(3.0, 0.0))  // 1

	// Constants
	if math.Pi > 3.14 && math.Pi < 3.15 {
		println("pi ok")
	}
	if math.E > 2.71 && math.E < 2.72 {
		println("e ok")
	}
}
