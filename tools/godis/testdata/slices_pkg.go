package main

import "slices"

func main() {
	// Contains with []int
	nums := []int{10, 20, 30, 40, 50}
	if slices.Contains(nums, 30) {
		println("contains 30: true")
	} else {
		println("contains 30: false")
	}
	if slices.Contains(nums, 99) {
		println("contains 99: true")
	} else {
		println("contains 99: false")
	}

	// Index with []int
	idx := slices.Index(nums, 40)
	println("index of 40:", idx)
	idx2 := slices.Index(nums, 99)
	println("index of 99:", idx2)

	// Contains with []string
	words := []string{"hello", "world", "foo"}
	if slices.Contains(words, "world") {
		println("contains world: true")
	} else {
		println("contains world: false")
	}

	// Index with []string
	idx3 := slices.Index(words, "foo")
	println("index of foo:", idx3)

	// Reverse
	rev := []int{1, 2, 3, 4, 5}
	slices.Reverse(rev)
	for _, v := range rev {
		println(v)
	}

	// Equal
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}
	if slices.Equal(a, b) {
		println("a==b: true")
	} else {
		println("a==b: false")
	}
	if slices.Equal(a, c) {
		println("a==c: true")
	} else {
		println("a==c: false")
	}

	// Sort
	unsorted := []int{5, 3, 1, 4, 2}
	slices.Sort(unsorted)
	for _, v := range unsorted {
		println(v)
	}

	// Min and Max
	vals := []int{42, 7, 99, 3, 55}
	println("min:", slices.Min(vals))
	println("max:", slices.Max(vals))
}
