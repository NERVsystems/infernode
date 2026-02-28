package main

import "strings"

func main() {
	// n=2: at most 2 parts, keep separator
	parts := strings.SplitAfterN("a,b,c", ",", 2)
	println(len(parts))  // 2
	println(parts[0])    // a,
	println(parts[1])    // b,c

	// n=-1: unlimited
	parts2 := strings.SplitAfterN("a,b,c", ",", -1)
	println(len(parts2)) // 3
	println(parts2[0])   // a,
	println(parts2[1])   // b,
	println(parts2[2])   // c

	// n=1: no splitting
	parts3 := strings.SplitAfterN("a,b,c", ",", 1)
	println(len(parts3)) // 1
	println(parts3[0])   // a,b,c

	println("splitaftern ok")
}
