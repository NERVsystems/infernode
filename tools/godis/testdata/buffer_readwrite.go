package main

import "bytes"

func main() {
	// Write then Read
	var buf bytes.Buffer
	buf.WriteString("Hello, World!")
	println(buf.Len())    // 13
	println(buf.String())  // Hello, World!

	// ReadByte
	b, _ := buf.ReadByte()
	println(b)             // 72 (ASCII 'H')
	println(buf.Len())    // 12

	// ReadString
	s, _ := buf.ReadString(',')
	println(s)             // ello,
	println(buf.Len())    // 7

	// Read into byte slice
	p := make([]byte, 3)
	n, _ := buf.Read(p)
	println(n)             // 3
	println(string(p))     // _Wo
	println(buf.Len())    // 4

	// Reset
	buf.Reset()
	println(buf.Len())    // 0

	println("ok")
}
