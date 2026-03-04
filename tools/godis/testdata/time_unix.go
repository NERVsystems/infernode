package main

import "time"

func main() {
	// Basic: 1 second
	t1 := time.Unix(1, 0)
	println(t1.Unix()) // 1

	// With nanoseconds: 1 second + 500ms
	t2 := time.Unix(1, 500000000)
	println(t2.UnixMilli()) // 1500

	// Large timestamp: 2024-01-01 00:00:00 UTC = 1704067200
	t3 := time.Unix(1704067200, 0)
	println(t3.Unix()) // 1704067200

	// Zero
	t4 := time.Unix(0, 0)
	println(t4.Unix()) // 0

	// Nanoseconds only: 1ms = 1000000ns
	t5 := time.Unix(0, 1000000)
	println(t5.UnixMilli()) // 1
}
