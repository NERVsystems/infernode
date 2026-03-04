package main

import "time"

func main() {
	// Parse RFC3339
	t1, _ := time.Parse(time.RFC3339, "2024-03-15T14:30:45Z")
	println(t1.Year())
	println(t1.Month())
	println(t1.Day())
	println(t1.Hour())
	println(t1.Minute())
	println(t1.Second())

	// Parse DateOnly
	t2, _ := time.Parse(time.DateOnly, "2024-01-05")
	println(t2.Year())
	println(t2.Month())
	println(t2.Day())

	// Round-trip: format then parse
	t3 := time.Date(2000, 12, 31, 23, 59, 59, 0, time.UTC)
	s := t3.Format(time.RFC3339)
	t4, _ := time.Parse(time.RFC3339, s)
	println(t4.Year())
	println(t4.Month())
	println(t4.Day())
	println(t4.Hour())
	println(t4.Minute())
	println(t4.Second())
}
