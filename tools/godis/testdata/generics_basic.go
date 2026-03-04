package main

// User-defined generic functions compiled via monomorphization.

func Min[T int | float64 | string](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T int | float64 | string](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func Map[T any, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

func Filter[T any](s []T, f func(T) bool) []T {
	var result []T
	for _, v := range s {
		if f(v) {
			result = append(result, v)
		}
	}
	return result
}

func Reduce[T any, U any](s []T, init U, f func(U, T) U) U {
	acc := init
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

func main() {
	// Min/Max with int
	println(Min(3, 7))
	println(Max(3, 7))

	// Min/Max with string
	println(Min("apple", "banana"))
	println(Max("apple", "banana"))

	// Contains
	nums := []int{10, 20, 30, 40}
	if Contains(nums, 30) {
		println("found")
	}
	if !Contains(nums, 99) {
		println("not found")
	}

	// Map: double each element
	doubled := Map(nums, func(x int) int { return x * 2 })
	for _, v := range doubled {
		println(v)
	}

	// Filter: keep evens > 20
	big := Filter(nums, func(x int) bool { return x > 20 })
	println(len(big))

	// Reduce: sum
	sum := Reduce(nums, 0, func(acc, x int) int { return acc + x })
	println(sum)

	// Map string->int (cross-type)
	words := []string{"hi", "hello", "hey"}
	lengths := Map(words, func(s string) int { return len(s) })
	for _, l := range lengths {
		println(l)
	}
}
