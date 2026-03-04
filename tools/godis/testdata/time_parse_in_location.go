package main

import "time"

func main() {
	// ParseInLocation with RFC3339 — should parse like time.Parse (loc is ignored)
	t1, _ := time.ParseInLocation(time.RFC3339, "2024-03-15T14:30:45Z", 0)
	println(t1.Year())
	println(t1.Month())
	println(t1.Day())
	println(t1.Hour())
	println(t1.Minute())
	println(t1.Second())

	// ParseInLocation with DateOnly
	t2, _ := time.ParseInLocation(time.DateOnly, "2024-01-05", 0)
	println(t2.Year())
	println(t2.Month())
	println(t2.Day())
}
