package main

import (
	"fmt"
	"math/bits"
)

func main() {
	// OnesCount
	fmt.Println(bits.OnesCount(0))  // 0
	fmt.Println(bits.OnesCount(1))  // 1
	fmt.Println(bits.OnesCount(7))  // 3
	fmt.Println(bits.OnesCount(255)) // 8

	// Len
	fmt.Println(bits.Len(0))  // 0
	fmt.Println(bits.Len(1))  // 1
	fmt.Println(bits.Len(8))  // 4
	fmt.Println(bits.Len(255)) // 8

	// TrailingZeros
	fmt.Println(bits.TrailingZeros(8))  // 3
	fmt.Println(bits.TrailingZeros(12)) // 2

	// LeadingZeros - returns 64 - Len(x)
	fmt.Println(bits.LeadingZeros(1))  // 63
	fmt.Println(bits.LeadingZeros(0))  // 64
}
