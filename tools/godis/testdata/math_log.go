package main

import (
	"fmt"
	"math"
)

func main() {
	// Log(e) ≈ 1.0
	v := math.Log(math.E)
	// Check it's close to 1.0 (within 0.01)
	if v > 0.99 && v < 1.01 {
		fmt.Println("ok")
	}

	// Pow test
	fmt.Println(int(math.Pow(2.0, 8.0))) // 256

	// IsNaN
	if math.IsNaN(math.NaN()) {
		fmt.Println("nan")
	}
	if !math.IsNaN(1.0) {
		fmt.Println("not nan")
	}
}
