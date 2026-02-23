package main

import "math"

func main() {
	// Log(e) should be ~1.0
	v := math.Log(math.E)
	if v > 0.99 && v < 1.01 {
		println("ln ok")
	}

	// Log2(8) should be ~3.0
	v2 := math.Log2(8.0)
	if v2 > 2.99 && v2 < 3.01 {
		println("log2 ok")
	}

	// Log10(1000) should be ~3.0
	v3 := math.Log10(1000.0)
	if v3 > 2.99 && v3 < 3.01 {
		println("log10 ok")
	}

	// Inf
	inf := math.Inf(1)
	if math.IsInf(inf, 0) {
		println("inf ok")
	}
	negInf := math.Inf(-1)
	if math.IsInf(negInf, -1) {
		println("neginf ok")
	}
}
