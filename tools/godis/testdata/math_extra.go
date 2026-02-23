package main

import (
	"fmt"
	"math"
)

func main() {
	// Floor
	fmt.Println(int(math.Floor(3.7)))   // 3
	fmt.Println(int(math.Floor(-3.7)))  // -4
	fmt.Println(int(math.Floor(5.0)))   // 5

	// Ceil
	fmt.Println(int(math.Ceil(3.2)))    // 4
	fmt.Println(int(math.Ceil(-3.2)))   // -3

	// Round
	fmt.Println(int(math.Round(3.5)))   // 4
	fmt.Println(int(math.Round(-3.5)))  // -4
	fmt.Println(int(math.Round(2.4)))   // 2

	// Mod
	fmt.Println(int(math.Mod(7.0, 3.0)))  // 1

	// Pow
	fmt.Println(int(math.Pow(2.0, 10.0)))  // 1024
	fmt.Println(int(math.Pow(3.0, 3.0)))   // 27
}
