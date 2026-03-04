package main

import "time"

func main() {
	t1 := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)

	// MarshalText returns RFC3339 as []byte
	textBytes, _ := t1.MarshalText()
	println(string(textBytes))

	// MarshalJSON returns RFC3339 with quotes as []byte
	jsonBytes, _ := t1.MarshalJSON()
	println(string(jsonBytes))

	// Zero-padded edge case
	t2 := time.Date(2024, 1, 5, 8, 3, 7, 0, time.UTC)
	textBytes2, _ := t2.MarshalText()
	println(string(textBytes2))
	jsonBytes2, _ := t2.MarshalJSON()
	println(string(jsonBytes2))

	// Epoch
	t3 := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	textBytes3, _ := t3.MarshalText()
	println(string(textBytes3))
}
