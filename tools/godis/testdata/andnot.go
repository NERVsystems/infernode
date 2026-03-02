package main

import "fmt"

func main() {
	// AND NOT: x &^ y = x AND (NOT y)
	fmt.Println(0xFF &^ 0x0F)            // 240 (0xF0)
	fmt.Println(0xFF &^ 0xFF)            // 0
	fmt.Println(0xFF &^ 0)               // 255
	fmt.Println(0x12345678 &^ 0x0000FF00) // 305397880 (0x12340678)
	fmt.Println("ok")
}
