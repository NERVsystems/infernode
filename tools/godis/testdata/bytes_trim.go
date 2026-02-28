package main

import "bytes"

func main() {
	// Trim
	println(string(bytes.Trim([]byte("  hello  "), " ")))     // hello
	println(string(bytes.Trim([]byte("xxhelloxx"), "x")))     // hello
	println(string(bytes.Trim([]byte("abchelloabc"), "abc"))) // hello

	// TrimLeft
	println(string(bytes.TrimLeft([]byte("  hello  "), " ")))  // hello  (trailing spaces preserved)
	println(string(bytes.TrimLeft([]byte("xxhello"), "x")))    // hello

	// TrimRight
	println(string(bytes.TrimRight([]byte("  hello  "), " "))) // (leading spaces preserved)  hello
	println(string(bytes.TrimRight([]byte("helloxx"), "x")))   // hello

	println("bytes trim ok")
}
