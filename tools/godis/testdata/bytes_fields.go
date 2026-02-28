package main

import "bytes"

func main() {
	parts := bytes.Fields([]byte("  hello  world  "))
	println(len(parts))          // 2
	println(string(parts[0]))    // hello
	println(string(parts[1]))    // world

	parts2 := bytes.Fields([]byte("one\ttwo\nthree"))
	println(len(parts2))         // 3
	println(string(parts2[0]))   // one
	println(string(parts2[1]))   // two
	println(string(parts2[2]))   // three

	parts3 := bytes.Fields([]byte(""))
	println(len(parts3))         // 0

	parts4 := bytes.Fields([]byte("single"))
	println(len(parts4))         // 1
	println(string(parts4[0]))   // single

	println("bytes fields ok")
}
