package main

import (
	"fmt"
	"strings"
)

func main() {
	// Count
	fmt.Println(strings.Count("hello", "l"))    // 2
	fmt.Println(strings.Count("abcabc", "abc"))  // 2

	// TrimPrefix
	fmt.Println(strings.TrimPrefix("Hello World", "Hello ")) // World
	fmt.Println(strings.TrimPrefix("Hello", "Bye"))          // Hello

	// TrimSuffix
	fmt.Println(strings.TrimSuffix("hello.go", ".go")) // hello
	fmt.Println(strings.TrimSuffix("hello", ".go"))     // hello

	// EqualFold (case-insensitive)
	if strings.EqualFold("Hello", "hello") {
		fmt.Println("equal")
	}
	if !strings.EqualFold("Hello", "world") {
		fmt.Println("not equal")
	}

	// LastIndex
	fmt.Println(strings.LastIndex("go go go", "go")) // 6
	fmt.Println(strings.LastIndex("hello", "xyz"))    // -1

	// ContainsRune
	if strings.ContainsRune("hello", 'e') {
		fmt.Println("has e")
	}

	// IndexByte
	fmt.Println(strings.IndexByte("hello", 'l')) // 2
	fmt.Println(strings.IndexByte("hello", 'z')) // -1
}
