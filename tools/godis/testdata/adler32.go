package main

import (
	"fmt"
	"hash/adler32"
)

func main() {
	// Adler-32 of empty data
	fmt.Println(adler32.Checksum([]byte{})) // 1

	// Adler-32 of "Wikipedia"
	fmt.Println(adler32.Checksum([]byte("Wikipedia"))) // 300286872

	// Adler-32 of "hello"
	fmt.Println(adler32.Checksum([]byte("hello"))) // 103547413

	fmt.Println("adler32 ok")
}
