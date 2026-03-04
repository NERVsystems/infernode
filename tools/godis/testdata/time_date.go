package main

import "time"

func main() {
	// Unix epoch
	t1 := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t1.Unix()) // 0

	// Known date: 2000-01-01 00:00:00 UTC = 946684800
	t2 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t2.Unix()) // 946684800

	// Known date: 2024-03-15 12:30:45 UTC = 1710505845
	t3 := time.Date(2024, 3, 15, 12, 30, 45, 0, time.UTC)
	println(t3.Unix()) // 1710505845

	// Leap year date: 2024-02-29 00:00:00 UTC = 1709164800
	t4 := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	println(t4.Unix()) // 1709164800

	// 1999-12-31 23:59:59 UTC = 946684799
	t5 := time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC)
	println(t5.Unix()) // 946684799

	// Non-leap year: 2023-03-01 00:00:00 UTC = 1677628800
	t6 := time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC)
	println(t6.Unix()) // 1677628800

	// Century non-leap: 1900 is NOT a leap year
	// 2001-01-01 00:00:00 UTC = 978307200
	t7 := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	println(t7.Unix()) // 978307200
}
