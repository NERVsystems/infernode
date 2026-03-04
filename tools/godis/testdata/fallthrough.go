package main

func classify(n int) string {
	switch {
	case n < 0:
		return "negative"
	case n == 0:
		return "zero"
	case n < 10:
		fallthrough
	case n < 100:
		return "small"
	default:
		return "large"
	}
}

func main() {
	println(classify(-1))
	println(classify(0))
	println(classify(5))
	println(classify(50))
	println(classify(500))
}
