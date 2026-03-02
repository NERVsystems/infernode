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
	// Atan
	fmt.Println(approx(math.Atan(0), 0, 1e-10))          // true
	fmt.Println(approx(math.Atan(1), math.Pi/4, 1e-10))   // true
	fmt.Println(approx(math.Atan(-1), -math.Pi/4, 1e-10)) // true
	fmt.Println(approx(math.Atan(10), 1.4711, 0.001))     // true

	// Asin
	fmt.Println(approx(math.Asin(0), 0, 1e-10))              // true
	fmt.Println(approx(math.Asin(1), math.Pi/2, 1e-6))        // true
	fmt.Println(approx(math.Asin(0.5), 0.5235987756, 1e-6))   // true

	// Acos
	fmt.Println(approx(math.Acos(1), 0, 1e-10))               // true
	fmt.Println(approx(math.Acos(0), math.Pi/2, 1e-6))        // true
	fmt.Println(approx(math.Acos(0.5), 1.0471975512, 1e-6))   // true

	// Sinh, Cosh, Tanh
	fmt.Println(approx(math.Sinh(0), 0, 1e-10))            // true
	fmt.Println(approx(math.Sinh(1), 1.1752, 0.001))       // true
	fmt.Println(approx(math.Cosh(0), 1, 1e-10))            // true
	fmt.Println(approx(math.Cosh(1), 1.5431, 0.001))       // true
	fmt.Println(approx(math.Tanh(0), 0, 1e-10))            // true
	fmt.Println(approx(math.Tanh(1), 0.7616, 0.001))       // true

	// Exp2
	fmt.Println(approx(math.Exp2(0), 1, 1e-10))   // true
	fmt.Println(approx(math.Exp2(3), 8, 1e-6))    // true
	fmt.Println(approx(math.Exp2(10), 1024, 0.1)) // true

	// Hypot
	fmt.Println(approx(math.Hypot(3, 4), 5, 1e-6))        // true
	fmt.Println(approx(math.Hypot(5, 12), 13, 1e-6))      // true

	// Cbrt
	fmt.Println(approx(math.Cbrt(27), 3, 1e-6))    // true
	fmt.Println(approx(math.Cbrt(8), 2, 1e-6))     // true
	fmt.Println(approx(math.Cbrt(-8), -2, 1e-6))   // true

	// Atan2
	fmt.Println(approx(math.Atan2(0, 1), 0, 1e-10))        // true
	fmt.Println(approx(math.Atan2(1, 0), math.Pi/2, 1e-6)) // true
	fmt.Println(approx(math.Atan2(1, 1), math.Pi/4, 1e-6)) // true

	// Log1p
	fmt.Println(approx(math.Log1p(0), 0, 1e-10))                   // true
	fmt.Println(approx(math.Log1p(1), 0.6931471805599453, 1e-6))   // true

	fmt.Println("math ok")
}
