package main

import "fmt"

func main() {
	pi := 3.14159265358979
	fmt.Printf("%.2f\n", pi)
	fmt.Printf("%.0f\n", pi)
	fmt.Printf("%.4f\n", pi)
	fmt.Printf("%f\n", pi)

	neg := -2.5
	fmt.Printf("%.1f\n", neg)

	s := fmt.Sprintf("val=%.3f", 1.0/3.0)
	fmt.Println(s)
}
