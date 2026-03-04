package main

// Type constraints with interface unions and ~T approximation.

type Number interface {
	~int | ~float64
}

type Ordered interface {
	~int | ~float64 | ~string
}

func Sum[T Number](vals []T) T {
	var total T
	for _, v := range vals {
		total += v
	}
	return total
}

func SortedInsert[T Ordered](sorted []T, v T) []T {
	for i, x := range sorted {
		if v < x {
			// Insert before i
			sorted = append(sorted, v) // grow by 1
			copy(sorted[i+1:], sorted[i:len(sorted)-1])
			sorted[i] = v
			return sorted
		}
	}
	return append(sorted, v)
}

// Named type with underlying int satisfies ~int constraint.
type Score int

func main() {
	// Sum[int]
	ints := []int{1, 2, 3, 4, 5}
	println(Sum(ints))

	// SortedInsert[int]
	s := []int{1, 3, 5}
	s = SortedInsert(s, 2)
	s = SortedInsert(s, 4)
	for _, v := range s {
		println(v)
	}

	// SortedInsert[string]
	ws := []string{"apple", "cherry"}
	ws = SortedInsert(ws, "banana")
	for _, w := range ws {
		println(w)
	}

	// Named type Score satisfies ~int via Number
	scores := []Score{10, 20, 30}
	total := Sum(scores)
	println(int(total))
}
