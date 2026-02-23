package main

import "strings"

func main() {
	// Count
	println(strings.Count("cheese", "e")) // 3
	println(strings.Count("hello", "ll")) // 1

	// TrimPrefix
	println(strings.TrimPrefix("Hello, World", "Hello, ")) // World
	println(strings.TrimPrefix("Hello", "Bye"))            // Hello

	// TrimSuffix
	println(strings.TrimSuffix("hello.go", ".go")) // hello
	println(strings.TrimSuffix("hello.go", ".py")) // hello.go

	// EqualFold (case-insensitive compare)
	if strings.EqualFold("Hello", "hello") {
		println("fold eq")
	}
	if !strings.EqualFold("Hello", "world") {
		println("fold ne")
	}

	// LastIndex
	println(strings.LastIndex("go gopher", "go")) // 3

	// ReplaceAll
	println(strings.ReplaceAll("oink oink oink", "oink", "moo")) // moo moo moo
}
