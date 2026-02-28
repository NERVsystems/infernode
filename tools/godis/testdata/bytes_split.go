package main

import "bytes"

func main() {
	// Split
	parts := bytes.Split([]byte("a,b,c"), []byte(","))
	println(len(parts)) // 3
	println(string(parts[0])) // a
	println(string(parts[1])) // b
	println(string(parts[2])) // c

	parts2 := bytes.Split([]byte("hello"), []byte(","))
	println(len(parts2)) // 1
	println(string(parts2[0])) // hello

	parts3 := bytes.Split([]byte("a::b::c"), []byte("::"))
	println(len(parts3)) // 3
	println(string(parts3[0])) // a
	println(string(parts3[1])) // b
	println(string(parts3[2])) // c

	// Join
	joined := bytes.Join([][]byte{[]byte("a"), []byte("b"), []byte("c")}, []byte(","))
	println(string(joined)) // a,b,c

	joined2 := bytes.Join([][]byte{[]byte("hello")}, []byte("-"))
	println(string(joined2)) // hello

	println("bytes split ok")
}
