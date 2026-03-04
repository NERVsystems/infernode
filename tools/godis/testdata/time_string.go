package main

import "time"

func main() {
	// Known date
	t1 := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	println(t1.String())

	// Midnight with zero-padding
	t2 := time.Date(2024, 1, 5, 8, 3, 7, 0, time.UTC)
	println(t2.String())

	// Epoch
	t3 := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t3.String())

	// Y2K
	t4 := time.Date(2000, 12, 31, 23, 59, 59, 0, time.UTC)
	println(t4.String())
}
