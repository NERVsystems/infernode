package main

import (
	"bytes"
	"fmt"
)

func main() {
	var b bytes.Buffer
	b.WriteString("hello")
	b.WriteString(" ")
	b.WriteString("world")
	fmt.Println(b.String()) // hello world
	fmt.Println(b.Len())    // 11

	// Reset and reuse
	b.Reset()
	fmt.Println(b.Len()) // 0

	// Write bytes
	b.Write([]byte("abc"))
	fmt.Println(b.String()) // abc

	// WriteByte
	b.WriteByte('!')
	fmt.Println(b.String()) // abc!

	// Bytes()
	data := b.Bytes()
	fmt.Println(len(data)) // 4

	// Cap
	fmt.Println(b.Cap()) // 4
}
