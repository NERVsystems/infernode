package main

import "path/filepath"

func main() {
	// Exact match
	m1, _ := filepath.Match("hello.go", "hello.go")
	println(m1) // true

	// No match
	m2, _ := filepath.Match("hello.go", "world.go")
	println(m2) // false

	// Wildcard all
	m3, _ := filepath.Match("*", "anything.txt")
	println(m3) // true

	// Suffix wildcard: *.go
	m4, _ := filepath.Match("*.go", "main.go")
	println(m4) // true

	m5, _ := filepath.Match("*.go", "main.txt")
	println(m5) // false

	// Prefix wildcard: test*
	m6, _ := filepath.Match("test*", "testing")
	println(m6) // true

	m7, _ := filepath.Match("test*", "hello")
	println(m7) // false

	println("ok")
}
