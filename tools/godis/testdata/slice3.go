package main

import "fmt"

func main() {
	s := []int{1, 2, 3, 4, 5}

	// 3-index slice: s[low:high:max]
	t := s[1:3:4]
	fmt.Println(len(t)) // 2
	fmt.Println(t[0])   // 2
	fmt.Println(t[1])   // 3

	// 2-index slice for comparison
	u := s[1:3]
	fmt.Println(len(u)) // 2
	fmt.Println(u[0])   // 2
}
