package main

import "net/http"

func main() {
	// PNG magic bytes
	println(http.DetectContentType([]byte{0x89, 0x50, 0x4E, 0x47}))
	// JPEG magic bytes
	println(http.DetectContentType([]byte{0xFF, 0xD8, 0xFF}))
	// GIF magic bytes
	println(http.DetectContentType([]byte{0x47, 0x49, 0x46, 0x38}))
	// PDF magic bytes
	println(http.DetectContentType([]byte{0x25, 0x50, 0x44, 0x46}))
	// ZIP magic bytes
	println(http.DetectContentType([]byte{0x50, 0x4B, 0x03, 0x04}))
	// HTML (starts with <)
	println(http.DetectContentType([]byte("<html>")))
	// JSON (starts with {)
	println(http.DetectContentType([]byte("{\"key\": 1}")))
	// Plain text
	println(http.DetectContentType([]byte("Hello, world")))
	// Binary data
	println(http.DetectContentType([]byte{0x00, 0x01, 0x02}))
	// Empty
	println(http.DetectContentType([]byte{}))
}
