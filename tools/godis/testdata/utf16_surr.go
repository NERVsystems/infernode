package main

import (
	"fmt"
	"unicode/utf16"
)

func main() {
	// Surrogate range: 0xD800-0xDFFF
	fmt.Println(utf16.IsSurrogate(0xD800)) // true
	fmt.Println(utf16.IsSurrogate(0xDBFF)) // true
	fmt.Println(utf16.IsSurrogate(0xDC00)) // true
	fmt.Println(utf16.IsSurrogate(0xDFFF)) // true
	fmt.Println(utf16.IsSurrogate(0xD799)) // false
	fmt.Println(utf16.IsSurrogate(0xE000)) // false
	fmt.Println(utf16.IsSurrogate(0x0041)) // false - 'A'
	fmt.Println(utf16.IsSurrogate(-1))     // false
	fmt.Println("utf16 ok")
}
