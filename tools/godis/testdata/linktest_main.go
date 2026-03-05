package main

import "libpkg"

func main() {
	x := libpkg.Add(3, 4)
	println(x)
	y := libpkg.Multiply(5, 6)
	println(y)
	s := libpkg.Greet("World")
	println(s)
}
