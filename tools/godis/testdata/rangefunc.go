package main

import "fmt"

// Iterator function that yields values 1, 2, 3
func countTo3(yield func(int) bool) {
	for i := 1; i <= 3; i++ {
		if !yield(i) {
			return
		}
	}
}

// Iterator that yields key-value pairs
func enumerate(items []string) func(func(int, string) bool) {
	return func(yield func(int, string) bool) {
		for i, item := range items {
			if !yield(i, item) {
				return
			}
		}
	}
}

func main() {
	// Range over func(yield func(int) bool)
	sum := 0
	for v := range countTo3 {
		sum += v
	}
	fmt.Println(sum) // 6

	// Range over func(yield func(int, string) bool)
	items := []string{"a", "b", "c"}
	for i, s := range enumerate(items) {
		fmt.Println(i, s)
	}

	// Range with break
	for v := range countTo3 {
		if v == 2 {
			break
		}
		fmt.Println(v) // 1
	}
}
