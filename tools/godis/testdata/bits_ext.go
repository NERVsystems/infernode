package main

import (
	"fmt"
	"math/bits"
)

func main() {
	// LeadingZeros
	fmt.Println(bits.LeadingZeros(1))   // 63
	fmt.Println(bits.LeadingZeros(255)) // 56
	fmt.Println(bits.LeadingZeros(0))   // 64

	// ReverseBytes16
	fmt.Println(bits.ReverseBytes16(0x0102)) // 513 (0x0201)
	fmt.Println(bits.ReverseBytes16(0xFF00)) // 255 (0x00FF)

	// OnesCount8
	fmt.Println(bits.OnesCount8(0))    // 0
	fmt.Println(bits.OnesCount8(0xFF)) // 8
	fmt.Println(bits.OnesCount8(0x0F)) // 4

	// TrailingZeros32
	fmt.Println(bits.TrailingZeros32(8))  // 3
	fmt.Println(bits.TrailingZeros32(1))  // 0
	fmt.Println(bits.TrailingZeros32(0))  // 32

	// Len64
	fmt.Println(bits.Len64(0))   // 0
	fmt.Println(bits.Len64(1))   // 1
	fmt.Println(bits.Len64(255)) // 8

	// Add
	sum, carry := bits.Add(^uint(0), 1, 0)
	fmt.Println(sum, carry) // 0 1

	// Sub
	diff, borrow := bits.Sub(0, 1, 0)
	fmt.Println(diff, borrow) // max 1

	fmt.Println("ok")
}
