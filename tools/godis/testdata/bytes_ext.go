package main

import "bytes"

func main() {
	// IndexAny
	s := []byte("hello world")
	println(bytes.IndexAny(s, "ow"))   // 4 ('o')
	println(bytes.IndexAny(s, "xyz"))  // -1

	// LastIndexAny
	println(bytes.LastIndexAny(s, "ol"))  // 9 ('l')
	println(bytes.LastIndexAny(s, "xyz")) // -1

	// LastIndexByte
	println(bytes.LastIndexByte(s, 'l')) // 9
	println(bytes.LastIndexByte(s, 'z')) // -1

	// IndexRune
	println(bytes.IndexRune(s, 'w')) // 6
	println(bytes.IndexRune(s, 'z')) // -1

	println("bytes ext ok")
}
