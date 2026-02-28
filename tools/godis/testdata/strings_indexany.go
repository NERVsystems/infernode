package main

import "strings"

func main() {
	// IndexAny: first index of any char in chars
	println(strings.IndexAny("hello", "aeiou"))  // 1 (e)
	println(strings.IndexAny("hello", "xyz"))     // -1
	println(strings.IndexAny("hello", "ol"))      // 2 (l)

	// LastIndexByte: last index of a byte
	println(strings.LastIndexByte("hello", 'l'))  // 3
	println(strings.LastIndexByte("hello", 'h'))  // 0
	println(strings.LastIndexByte("hello", 'z'))  // -1

	// LastIndexAny: last index of any char in chars
	println(strings.LastIndexAny("hello", "aeiou"))  // 4 (o)
	println(strings.LastIndexAny("hello", "xyz"))     // -1
	println(strings.LastIndexAny("hello", "le"))      // 3 (l)

	println("indexany ok")
}
