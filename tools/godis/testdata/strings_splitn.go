package main

import "strings"

func main() {
	// n = 2: split into at most 2 parts
	parts := strings.SplitN("a,b,c", ",", 2)
	println(len(parts))        // 2
	println(parts[0])          // a
	println(parts[1])          // b,c

	// n = -1: unlimited (same as Split)
	parts2 := strings.SplitN("a,b,c", ",", -1)
	println(len(parts2))       // 3
	println(parts2[0])         // a
	println(parts2[1])         // b
	println(parts2[2])         // c

	// n = 1: no splitting
	parts3 := strings.SplitN("a,b,c", ",", 1)
	println(len(parts3))       // 1
	println(parts3[0])         // a,b,c

	// n = 3: split into at most 3 parts
	parts4 := strings.SplitN("a::b::c::d", "::", 3)
	println(len(parts4))       // 3
	println(parts4[0])         // a
	println(parts4[1])         // b
	println(parts4[2])         // c::d

	println("splitn ok")
}
