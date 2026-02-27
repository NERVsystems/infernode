package main

import (
	"fmt"
	"strings"
)

func main() {
	var b strings.Builder
	b.WriteString("hello")
	b.WriteString(" ")
	b.WriteString("world")
	fmt.Println(b.String())
	fmt.Println(b.Len())

	// Reset and reuse
	b.Reset()
	fmt.Println(b.Len())

	// Build with loop
	for i := 0; i < 3; i++ {
		b.WriteString("go")
	}
	fmt.Println(b.String())
}
