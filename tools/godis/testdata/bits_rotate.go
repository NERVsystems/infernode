package main

import (
	"fmt"
	"math/bits"
)

func main() {
	// RotateLeft
	// RotateLeft(1, 3) = 8 (1 shifted left by 3)
	fmt.Println(bits.RotateLeft(1, 3)) // 8
	// RotateLeft(0x80, 1) on 64-bit = 0x100
	fmt.Println(bits.RotateLeft(128, 1)) // 256

	// ReverseBytes64(0x0102030405060708) = 0x0807060504030201
	// Let's use smaller values we can verify
	// ReverseBytes(1) on 64-bit: byte 0 = 1, rest = 0 → reversed byte 7 = 1 = 0x0100000000000000
	// That's a huge number. Let's check bits.Len instead.
	fmt.Println(bits.Len(255))  // 8
	fmt.Println(bits.Len(1023)) // 10

	// OnesCount of powers of 2
	fmt.Println(bits.OnesCount(16))  // 1
	fmt.Println(bits.OnesCount(15))  // 4
}
