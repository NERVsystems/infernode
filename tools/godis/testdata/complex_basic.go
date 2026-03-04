package main

func main() {
	// Basic construction and extraction
	c1 := complex(3.0, 4.0)
	println(int(real(c1)))
	println(int(imag(c1)))

	// Arithmetic: addition (3+4i) + (1+2i) = (4+6i)
	c2 := complex(1.0, 2.0)
	sum := c1 + c2
	println(int(real(sum)))
	println(int(imag(sum)))

	// Subtraction (3+4i) - (1+2i) = (2+2i)
	diff := c1 - c2
	println(int(real(diff)))
	println(int(imag(diff)))

	// Multiplication: (3+4i)(1+2i) = (3-8) + (6+4)i = (-5+10i)
	prod := c1 * c2
	println(int(real(prod)))
	println(int(imag(prod)))

	// Equality
	c3 := complex(3.0, 4.0)
	if c1 == c3 {
		println("equal")
	}
	if c1 != c2 {
		println("not equal")
	}

	// Negation
	neg := -c1
	println(int(real(neg)))
	println(int(imag(neg)))

	// Zero complex
	var z complex128
	println(int(real(z)))
	println(int(imag(z)))

	// Division: (10+0i)/(2+0i) = (5+0i)
	c4 := complex(10.0, 0.0)
	c5 := complex(2.0, 0.0)
	quot := c4 / c5
	println(int(real(quot)))
	println(int(imag(quot)))
}
