package main

import (
	"fmt"
	"strconv"
)

func main() {
	// ParseInt base 16 (no prefix — Go doesn't strip 0x when base specified)
	v, _ := strconv.ParseInt("ff", 16, 64)
	fmt.Println(v) // 255

	v3, _ := strconv.ParseInt("1a", 16, 64)
	fmt.Println(v3) // 26

	v3b, _ := strconv.ParseInt("DEAD", 16, 64)
	fmt.Println(v3b) // 57005

	// ParseInt base 8
	v4, _ := strconv.ParseInt("77", 8, 64)
	fmt.Println(v4) // 63

	v4b, _ := strconv.ParseInt("10", 8, 64)
	fmt.Println(v4b) // 8

	// ParseInt base 2
	v6, _ := strconv.ParseInt("1010", 2, 64)
	fmt.Println(v6) // 10

	v6b, _ := strconv.ParseInt("11111111", 2, 64)
	fmt.Println(v6b) // 255

	// ParseInt base 10 (decimal, existing behavior)
	v8, _ := strconv.ParseInt("42", 10, 64)
	fmt.Println(v8) // 42

	// Negative hex
	v9, _ := strconv.ParseInt("-ff", 16, 64)
	fmt.Println(v9) // -255

	// ParseUint base 16
	u1, _ := strconv.ParseUint("ff", 16, 64)
	fmt.Println(u1) // 255

	fmt.Println("ok")
}
