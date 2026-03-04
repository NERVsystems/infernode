package main

import "time"

func main() {
	t := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)

	// RFC3339
	println(t.Format(time.RFC3339))
	// DateOnly
	println(t.Format(time.DateOnly))
	// TimeOnly
	println(t.Format(time.TimeOnly))
	// DateTime
	println(t.Format(time.DateTime))

	// Zero-padded test
	t2 := time.Date(2024, 1, 5, 8, 3, 7, 0, time.UTC)
	println(t2.Format(time.RFC3339))
	println(t2.Format(time.DateOnly))

	// Epoch
	t3 := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t3.Format(time.RFC3339))
}
