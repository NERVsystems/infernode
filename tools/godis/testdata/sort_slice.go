package main

import (
	"fmt"
	"sort"
)

func main() {
	// Sort integers
	nums := []int{5, 3, 1, 4, 2}
	sort.Slice(nums, func(i, j int) bool {
		return nums[i] < nums[j]
	})
	fmt.Println(nums[0], nums[1], nums[2], nums[3], nums[4])

	// Sort strings
	strs := []string{"banana", "apple", "cherry"}
	sort.Slice(strs, func(i, j int) bool {
		return strs[i] < strs[j]
	})
	fmt.Println(strs[0], strs[1], strs[2])

	// Already sorted
	sorted := []int{1, 2, 3}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	fmt.Println(sorted[0], sorted[1], sorted[2])

	fmt.Println("sort slice ok")
}
