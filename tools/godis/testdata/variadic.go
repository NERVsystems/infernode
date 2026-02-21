package main

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total = total + n
	}
	return total
}

func main() {
	println(sum(1, 2, 3))    // 6
	println(sum(10, 20))     // 30
	println(sum())           // 0
}
