package main

import "unicode/utf8"

func main() {
	// Valid ASCII
	println(utf8.Valid([]byte("Hello")))
	// Valid 2-byte (é = 0xC3 0xA9)
	println(utf8.Valid([]byte{0xC3, 0xA9}))
	// Valid 3-byte (世 = 0xE4 0xB8 0x96)
	println(utf8.Valid([]byte{0xE4, 0xB8, 0x96}))
	// Valid 4-byte (😀 = 0xF0 0x9F 0x98 0x80)
	println(utf8.Valid([]byte{0xF0, 0x9F, 0x98, 0x80}))
	// Empty is valid
	println(utf8.Valid([]byte{}))

	// Invalid: lone continuation byte
	println(utf8.Valid([]byte{0x80}))
	// Invalid: overlong 2-byte (0xC0 0x80 encodes NUL)
	println(utf8.Valid([]byte{0xC0, 0x80}))
	// Invalid: truncated 2-byte
	println(utf8.Valid([]byte{0xC3}))
	// Invalid: surrogate (0xED 0xA0 0x80 = U+D800)
	println(utf8.Valid([]byte{0xED, 0xA0, 0x80}))
	// Invalid: byte 0xFF
	println(utf8.Valid([]byte{0xFF}))
	// Invalid: truncated 3-byte
	println(utf8.Valid([]byte{0xE4, 0xB8}))
	// Invalid: overlong 3-byte (0xE0 0x80 0x80)
	println(utf8.Valid([]byte{0xE0, 0x80, 0x80}))

	// ValidString always true for Dis strings
	println(utf8.ValidString("Hello"))
	println(utf8.ValidString(""))
}
