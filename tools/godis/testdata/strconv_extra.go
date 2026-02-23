package main

import (
	"fmt"
	"strconv"
)

func main() {
	// FormatBool
	fmt.Println(strconv.FormatBool(true))  // true
	fmt.Println(strconv.FormatBool(false)) // false

	// ParseBool
	b1, _ := strconv.ParseBool("true")
	b2, _ := strconv.ParseBool("false")
	b3, _ := strconv.ParseBool("1")
	if b1 {
		fmt.Println("yes")
	}
	if !b2 {
		fmt.Println("no")
	}
	if b3 {
		fmt.Println("one")
	}

	// ParseInt
	n, _ := strconv.ParseInt("42", 10, 64)
	fmt.Println(n) // 42
}
