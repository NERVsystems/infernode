package main

import (
	"fmt"
	"math"
)

func approx(x, y, eps float64) bool {
	d := x - y
	if d < 0 {
		d = -d
	}
	return d < eps
}

func main() {
	// Asinh
	fmt.Println(approx(math.Asinh(0), 0, 0.001))          // true
	fmt.Println(approx(math.Asinh(1), 0.8813736, 0.001))   // true
	fmt.Println(approx(math.Asinh(-1), -0.8813736, 0.001)) // true

	// Acosh
	fmt.Println(approx(math.Acosh(1), 0, 0.001))          // true
	fmt.Println(approx(math.Acosh(2), 1.3169579, 0.001))  // true
	fmt.Println(approx(math.Acosh(10), 2.9932228, 0.01))  // true

	// Atanh
	fmt.Println(approx(math.Atanh(0), 0, 0.001))           // true
	fmt.Println(approx(math.Atanh(0.5), 0.5493061, 0.001)) // true
	fmt.Println(approx(math.Atanh(-0.5), -0.5493061, 0.001)) // true

	// FMA
	fmt.Println(approx(math.FMA(2, 3, 4), 10, 0.001))      // true
	fmt.Println(approx(math.FMA(1.5, 2, 0.5), 3.5, 0.001)) // true

	// Modf
	intPart, fracPart := math.Modf(3.75)
	fmt.Println(approx(intPart, 3, 0.001))     // true
	fmt.Println(approx(fracPart, 0.75, 0.001))  // true
	intPart2, fracPart2 := math.Modf(-2.5)
	fmt.Println(approx(intPart2, -2, 0.001))    // true
	fmt.Println(approx(fracPart2, -0.5, 0.001)) // true

	// Frexp
	frac, exp := math.Frexp(8.0)
	fmt.Println(approx(frac, 0.5, 0.001)) // true
	fmt.Println(exp == 4)                  // true
	frac2, exp2 := math.Frexp(0.25)
	fmt.Println(approx(frac2, 0.5, 0.001)) // true
	fmt.Println(exp2 == -1)                 // true

	// Ldexp
	fmt.Println(approx(math.Ldexp(0.5, 4), 8.0, 0.001))    // true
	fmt.Println(approx(math.Ldexp(1.0, 10), 1024.0, 0.001)) // true
	fmt.Println(approx(math.Ldexp(1.0, -3), 0.125, 0.001))  // true

	// Ilogb
	fmt.Println(math.Ilogb(8.0))  // 3
	fmt.Println(math.Ilogb(1.0))  // 0
	fmt.Println(math.Ilogb(0.5))  // -1

	// Sincos
	s, c := math.Sincos(0)
	fmt.Println(approx(s, 0, 0.001))  // true
	fmt.Println(approx(c, 1, 0.001))  // true
	s2, c2 := math.Sincos(math.Pi / 2)
	fmt.Println(approx(s2, 1, 0.01))  // true
	fmt.Println(approx(c2, 0, 0.01))  // true

	// Nextafter
	x := math.Nextafter(1.0, 2.0)
	fmt.Println(x > 1.0)  // true
	x2 := math.Nextafter(1.0, 0.0)
	fmt.Println(x2 < 1.0) // true
	fmt.Println(math.Nextafter(1.0, 1.0) == 1.0) // true

	// Sqrt(0) edge case
	fmt.Println(approx(math.Sqrt(0), 0, 0.001)) // true

	fmt.Println("math advanced ok")
}
