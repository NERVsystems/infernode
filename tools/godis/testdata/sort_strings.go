package main

import (
	"fmt"
	"sort"
)

func main() {
	a := []string{"apple", "banana", "cherry", "date", "elderberry"}
	fmt.Println(sort.SearchStrings(a, "banana"))     // 1
	fmt.Println(sort.SearchStrings(a, "apple"))       // 0
	fmt.Println(sort.SearchStrings(a, "elderberry"))  // 4
	fmt.Println(sort.SearchStrings(a, "aardvark"))    // 0 (before first)
	fmt.Println(sort.SearchStrings(a, "coconut"))     // 3 (between cherry and date)
	fmt.Println(sort.SearchStrings(a, "fig"))          // 5 (after last)
	fmt.Println("ok")
}
