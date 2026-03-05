package main

import "fmt"

// Type constraint with multiple types
type Number interface {
	~int | ~float64
}

func double[T Number](x T) T {
	return x + x
}

func toString[T ~int | ~string](x T) string {
	return fmt.Sprint(x)
}

func main() {
	fmt.Println(double(21))    // 42
	fmt.Println(double(3.14))  // 6.28
	fmt.Println(toString(42))  // 42
}
