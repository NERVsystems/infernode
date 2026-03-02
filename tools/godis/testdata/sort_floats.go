package main

import (
	"fmt"
	"sort"
)

func main() {
	a := []float64{1.1, 2.2, 3.3, 4.4, 5.5}
	fmt.Println(sort.SearchFloat64s(a, 2.2))  // 1
	fmt.Println(sort.SearchFloat64s(a, 1.1))  // 0
	fmt.Println(sort.SearchFloat64s(a, 5.5))  // 4
	fmt.Println(sort.SearchFloat64s(a, 0.5))  // 0 (before first)
	fmt.Println(sort.SearchFloat64s(a, 2.5))  // 2 (between 2.2 and 3.3)
	fmt.Println(sort.SearchFloat64s(a, 6.0))  // 5 (after last)
	fmt.Println("ok")
}
