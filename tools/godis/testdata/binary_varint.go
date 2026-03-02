package main

import (
	"encoding/binary"
	"fmt"
)

func main() {
	buf := make([]byte, 10)

	// PutUvarint / Uvarint roundtrip
	n := binary.PutUvarint(buf, 0)
	fmt.Println(n) // 1
	v, m := binary.Uvarint(buf[:n])
	fmt.Println(v, m) // 0 1

	n = binary.PutUvarint(buf, 127)
	fmt.Println(n) // 1
	v, m = binary.Uvarint(buf[:n])
	fmt.Println(v, m) // 127 1

	n = binary.PutUvarint(buf, 128)
	fmt.Println(n) // 2
	v, m = binary.Uvarint(buf[:n])
	fmt.Println(v, m) // 128 2

	n = binary.PutUvarint(buf, 300)
	fmt.Println(n) // 2
	v, m = binary.Uvarint(buf[:n])
	fmt.Println(v, m) // 300 2

	// PutVarint / Varint roundtrip
	n = binary.PutVarint(buf, 0)
	fmt.Println(n) // 1
	sv, sm := binary.Varint(buf[:n])
	fmt.Println(sv, sm) // 0 1

	n = binary.PutVarint(buf, -1)
	fmt.Println(n) // 1
	sv, sm = binary.Varint(buf[:n])
	fmt.Println(sv, sm) // -1 1

	n = binary.PutVarint(buf, 63)
	fmt.Println(n) // 1
	sv, sm = binary.Varint(buf[:n])
	fmt.Println(sv, sm) // 63 1

	n = binary.PutVarint(buf, -64)
	fmt.Println(n) // 1
	sv, sm = binary.Varint(buf[:n])
	fmt.Println(sv, sm) // -64 1

	n = binary.PutVarint(buf, 64)
	fmt.Println(n) // 2
	sv, sm = binary.Varint(buf[:n])
	fmt.Println(sv, sm) // 64 2

	fmt.Println("ok")
}
