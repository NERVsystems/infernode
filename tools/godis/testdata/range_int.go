package main

func main() {
	// for range over int (Go 1.22)
	sum := 0
	for i := range 10 {
		sum += i
	}
	println(sum)

	// Nested range over int
	count := 0
	for range 3 {
		for range 4 {
			count++
		}
	}
	println(count)
}
