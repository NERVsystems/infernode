package main

import (
	"fmt"
	"unicode"
)

func main() {
	// IsLetter
	if unicode.IsLetter('A') {
		fmt.Println("letter")
	}
	if !unicode.IsLetter('5') {
		fmt.Println("not letter")
	}

	// IsDigit
	if unicode.IsDigit('9') {
		fmt.Println("digit")
	}
	if !unicode.IsDigit('x') {
		fmt.Println("not digit")
	}

	// IsSpace
	if unicode.IsSpace(' ') {
		fmt.Println("space")
	}
	if !unicode.IsSpace('A') {
		fmt.Println("not space")
	}

	// IsUpper / IsLower
	if unicode.IsUpper('Z') {
		fmt.Println("upper")
	}
	if unicode.IsLower('a') {
		fmt.Println("lower")
	}

	// ToUpper / ToLower
	fmt.Println(string(unicode.ToUpper('a'))) // A
	fmt.Println(string(unicode.ToLower('B'))) // b
}
