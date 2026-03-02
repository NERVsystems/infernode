package main

import "encoding/hex"

func main() {
	// Encode
	src := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
	dst := make([]byte, hex.EncodedLen(len(src)))
	n := hex.Encode(dst, src)
	println(n)           // 10
	println(string(dst)) // 48656c6c6f

	// Decode
	src2 := []byte("48656c6c6f")
	dst2 := make([]byte, hex.DecodedLen(len(src2)))
	n2, _ := hex.Decode(dst2, src2)
	println(n2)            // 5
	println(string(dst2))  // Hello

	// EncodeToString
	s := hex.EncodeToString([]byte{0x41, 0x42, 0x43})
	println(s) // 414243

	println("ok")
}
