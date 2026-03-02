package main

import (
	"fmt"
	"unicode/utf8"
)

func main() {
	// FullRune: check if bytes begin with a full UTF-8 encoding
	fmt.Println(utf8.FullRune([]byte{0x41}))             // true - ASCII 'A'
	fmt.Println(utf8.FullRune([]byte{0xC3, 0xA9}))       // true - 2-byte 'é'
	fmt.Println(utf8.FullRune([]byte{0xC3}))             // false - incomplete 2-byte
	fmt.Println(utf8.FullRune([]byte{0xE4, 0xB8, 0xAD})) // true - 3-byte '中'
	fmt.Println(utf8.FullRune([]byte{0xE4, 0xB8}))       // false - incomplete 3-byte
	fmt.Println(utf8.FullRune([]byte{0xF0, 0x9F, 0x98, 0x80})) // true - 4-byte '😀'
	fmt.Println(utf8.FullRune([]byte{0xF0, 0x9F, 0x98})) // false - incomplete 4-byte
	fmt.Println(utf8.FullRune([]byte{}))                  // false - empty

	// RuneCount: count runes in byte slice
	fmt.Println(utf8.RuneCount([]byte("hello")))          // 5
	fmt.Println(utf8.RuneCount([]byte("héllo")))          // 5
	fmt.Println(utf8.RuneCount([]byte("")))               // 0

	fmt.Println("utf8 ok")
}
