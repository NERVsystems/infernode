package main

import "time"

func main() {
	// Simple seconds
	d1, _ := time.ParseDuration("5s")
	println(d1) // 5000000000

	// Milliseconds
	d2, _ := time.ParseDuration("300ms")
	println(d2) // 300000000

	// Minutes
	d3, _ := time.ParseDuration("2m")
	println(d3) // 120000000000

	// Hours
	d4, _ := time.ParseDuration("1h")
	println(d4) // 3600000000000

	// Multi-component: 1h30m
	d5, _ := time.ParseDuration("1h30m")
	println(d5) // 5400000000000

	// Nanoseconds
	d6, _ := time.ParseDuration("100ns")
	println(d6) // 100

	// Microseconds
	d7, _ := time.ParseDuration("500us")
	println(d7) // 500000

	// Negative
	d8, _ := time.ParseDuration("-3s")
	println(d8) // -3000000000

	// Zero
	d9, _ := time.ParseDuration("0")
	println(d9) // 0

	// Complex: 1h30m10s
	d10, _ := time.ParseDuration("1h30m10s")
	println(d10) // 5410000000000

	// Positive sign
	d11, _ := time.ParseDuration("+2s")
	println(d11) // 2000000000
}
