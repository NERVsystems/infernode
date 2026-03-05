package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	// Slice of ints
	b1, _ := json.Marshal([]int{1, 2, 3})
	fmt.Println(string(b1))

	// Slice of strings
	b2, _ := json.Marshal([]string{"a", "b", "c"})
	fmt.Println(string(b2))

	// Empty slice
	b3, _ := json.Marshal([]int{})
	fmt.Println(string(b3))

	// Single element
	b4, _ := json.Marshal([]int{42})
	fmt.Println(string(b4))

	fmt.Println("json slice ok")
}
