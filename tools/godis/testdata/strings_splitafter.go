package main

import "strings"

func main() {
	// SplitAfter keeps the separator in each part
	parts := strings.SplitAfter("a,b,c", ",")
	println(len(parts))  // 3
	println(parts[0])    // a,
	println(parts[1])    // b,
	println(parts[2])    // c

	parts2 := strings.SplitAfter("hello", ",")
	println(len(parts2)) // 1
	println(parts2[0])   // hello

	parts3 := strings.SplitAfter("a::b::c", "::")
	println(len(parts3)) // 3
	println(parts3[0])   // a::
	println(parts3[1])   // b::
	println(parts3[2])   // c

	println("splitafter ok")
}
