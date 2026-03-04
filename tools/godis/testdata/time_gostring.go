package main

import "time"

func main() {
	t1 := time.Date(2024, 3, 15, 14, 30, 45, 0, 0)
	println(t1.GoString())

	t2 := time.Date(2000, 1, 1, 0, 0, 0, 0, 0)
	println(t2.GoString())
}
