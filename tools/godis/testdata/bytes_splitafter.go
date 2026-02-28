package main

import "bytes"

func main() {
	parts := bytes.SplitAfter([]byte("a,b,c"), []byte(","))
	println(len(parts))           // 3
	println(string(parts[0]))     // a,
	println(string(parts[1]))     // b,
	println(string(parts[2]))     // c

	parts2 := bytes.SplitAfter([]byte("hello"), []byte(","))
	println(len(parts2))          // 1
	println(string(parts2[0]))    // hello

	println("bytes splitafter ok")
}
