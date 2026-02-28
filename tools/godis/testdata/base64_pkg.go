package main

import "encoding/base64"

func main() {
	// EncodeToString
	s1 := base64.StdEncoding.EncodeToString([]byte("Hello, World!"))
	println(s1)

	s2 := base64.StdEncoding.EncodeToString([]byte("ab"))
	println(s2)

	s3 := base64.StdEncoding.EncodeToString([]byte("a"))
	println(s3)

	s4 := base64.StdEncoding.EncodeToString([]byte(""))
	println(s4)

	// DecodeString
	dec, _ := base64.StdEncoding.DecodeString("SGVsbG8sIFdvcmxkIQ==")
	println(string(dec))

	dec2, _ := base64.StdEncoding.DecodeString("YWI=")
	println(string(dec2))

	dec3, _ := base64.StdEncoding.DecodeString("YQ==")
	println(string(dec3))

	// EncodedLen / DecodedLen
	println(base64.StdEncoding.EncodedLen(0))
	println(base64.StdEncoding.EncodedLen(1))
	println(base64.StdEncoding.EncodedLen(3))
	println(base64.StdEncoding.EncodedLen(13))

	println(base64.StdEncoding.DecodedLen(0))
	println(base64.StdEncoding.DecodedLen(4))
	println(base64.StdEncoding.DecodedLen(8))
	println(base64.StdEncoding.DecodedLen(20))

	println("base64 ok")
}
