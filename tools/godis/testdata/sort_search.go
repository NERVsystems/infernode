package main

import (
	"fmt"
	"sort"
)

func main() {
	a := []int{10, 20, 30, 40, 50}
	fmt.Println(sort.SearchInts(a, 20))  // 1
	fmt.Println(sort.SearchInts(a, 10))  // 0
	fmt.Println(sort.SearchInts(a, 50))  // 4
	fmt.Println(sort.SearchInts(a, 5))   // 0 (not found, insert before first)
	fmt.Println(sort.SearchInts(a, 25))  // 2 (not found, insert at 2)
	fmt.Println(sort.SearchInts(a, 55))  // 5 (not found, insert after last)
	fmt.Println("ok")
}
