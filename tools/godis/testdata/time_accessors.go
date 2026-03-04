package main

import "time"

func main() {
	// 2024-03-15 14:30:45 UTC
	t := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)

	// Individual accessors
	println(t.Year())   // 2024
	println(t.Month())  // 3
	println(t.Day())    // 15
	println(t.Hour())   // 14
	println(t.Minute()) // 30
	println(t.Second()) // 45

	// Weekday: 2024-03-15 is a Friday (5)
	println(t.Weekday()) // 5

	// YearDay: March 15, 2024 (leap year)
	// Jan=31, Feb=29 (leap), Mar 1-15 = 15 → 31+29+15 = 75
	println(t.YearDay()) // 75

	// Midnight test
	t2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t2.Hour())   // 0
	println(t2.Minute()) // 0
	println(t2.Second()) // 0

	// Weekday: 2024-01-01 is a Monday (1)
	println(t2.Weekday()) // 1

	// YearDay: Jan 1 = 1
	println(t2.YearDay()) // 1

	// Epoch
	t3 := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t3.Year())    // 1970
	println(t3.Month())   // 1
	println(t3.Day())     // 1
	println(t3.Weekday()) // 4 (Thursday)
	println(t3.YearDay()) // 1

	// End of year: 2023-12-31 23:59:59 (non-leap)
	t4 := time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC)
	println(t4.Year())    // 2023
	println(t4.Month())   // 12
	println(t4.Day())     // 31
	println(t4.Hour())    // 23
	println(t4.Minute())  // 59
	println(t4.Second())  // 59
	println(t4.YearDay()) // 365
}
