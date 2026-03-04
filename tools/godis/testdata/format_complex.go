package main

import "strconv"

func main() {
	// Basic positive real and imag
	c1 := complex(1.5, 2.3)
	println(strconv.FormatComplex(c1, 'f', -1, 128))

	// Negative imaginary
	c2 := complex(3.0, -4.5)
	println(strconv.FormatComplex(c2, 'f', -1, 128))

	// Zero
	c3 := complex(0.0, 0.0)
	println(strconv.FormatComplex(c3, 'f', -1, 128))

	// Negative real, positive imag
	c4 := complex(-1.0, 1.0)
	println(strconv.FormatComplex(c4, 'f', -1, 128))

	// Both negative
	c5 := complex(-2.5, -3.5)
	println(strconv.FormatComplex(c5, 'f', -1, 128))
}
