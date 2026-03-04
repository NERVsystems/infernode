package main

import "crypto/subtle"

func main() {
	// ConstantTimeCompare
	a := []byte("hello")
	b := []byte("hello")
	c := []byte("world")
	println(subtle.ConstantTimeCompare(a, b)) // 1
	println(subtle.ConstantTimeCompare(a, c)) // 0

	// ConstantTimeEq
	println(subtle.ConstantTimeEq(42, 42)) // 1
	println(subtle.ConstantTimeEq(42, 43)) // 0

	// ConstantTimeSelect
	println(subtle.ConstantTimeSelect(1, 10, 20)) // 10
	println(subtle.ConstantTimeSelect(0, 10, 20)) // 20

	// ConstantTimeByteEq
	println(subtle.ConstantTimeByteEq(0x41, 0x41)) // 1
	println(subtle.ConstantTimeByteEq(0x41, 0x42)) // 0

	// ConstantTimeLessOrEq
	println(subtle.ConstantTimeLessOrEq(3, 5))  // 1
	println(subtle.ConstantTimeLessOrEq(5, 5))  // 1
	println(subtle.ConstantTimeLessOrEq(6, 5))  // 0

	// XORBytes
	x := []byte{0x0F, 0xF0, 0xAA}
	y := []byte{0xFF, 0x0F, 0x55}
	dst := make([]byte, 3)
	n := subtle.XORBytes(dst, x, y)
	println(n)          // 3
	println(int(dst[0])) // 240 (0xF0)
	println(int(dst[1])) // 255 (0xFF)
	println(int(dst[2])) // 255 (0xFF)

	// ConstantTimeCopy
	src := []byte{1, 2, 3}
	target := []byte{0, 0, 0}
	subtle.ConstantTimeCopy(1, target, src)
	println(int(target[0])) // 1
	println(int(target[1])) // 2
	println(int(target[2])) // 3

	// ConstantTimeCopy with v=0 (no copy)
	target2 := []byte{9, 8, 7}
	subtle.ConstantTimeCopy(0, target2, src)
	println(int(target2[0])) // 9
	println(int(target2[1])) // 8
	println(int(target2[2])) // 7
}
