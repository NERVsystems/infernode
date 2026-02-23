package main

import "strconv"

func main() {
	// FormatBool
	println(strconv.FormatBool(true))
	println(strconv.FormatBool(false))

	// ParseBool
	b1, _ := strconv.ParseBool("true")
	if b1 {
		println("parsed true")
	}
	b2, _ := strconv.ParseBool("false")
	if !b2 {
		println("parsed false")
	}
	b3, _ := strconv.ParseBool("1")
	if b3 {
		println("parsed 1")
	}

	// ParseInt base 16
	n, _ := strconv.ParseInt("ff", 16, 0)
	println(n) // 255

	// ParseInt base 10
	n2, _ := strconv.ParseInt("42", 10, 0)
	println(n2) // 42
}
