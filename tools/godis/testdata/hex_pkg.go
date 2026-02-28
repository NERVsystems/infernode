package main

import "encoding/hex"

func main() {
	// EncodeToString (already works)
	s := hex.EncodeToString([]byte("Hello"))
	println(s)

	// DecodeString
	b, _ := hex.DecodeString("48656c6c6f")
	println(string(b))

	b2, _ := hex.DecodeString("deadBEEF")
	println(hex.EncodeToString(b2))

	// EncodedLen / DecodedLen
	println(hex.EncodedLen(5))
	println(hex.DecodedLen(10))

	println("hex ok")
}
