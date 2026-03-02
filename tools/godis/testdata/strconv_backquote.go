package main

import (
	"fmt"
	"strconv"
)

func main() {
	fmt.Println(strconv.CanBackquote("hello"))       // true
	fmt.Println(strconv.CanBackquote("hello world"))  // true
	fmt.Println(strconv.CanBackquote("hello\tworld")) // true (tab allowed)
	fmt.Println(strconv.CanBackquote("hello`world"))  // false (backquote)
	fmt.Println(strconv.CanBackquote("hello\nworld")) // false (newline)
	fmt.Println(strconv.CanBackquote(""))             // true
	fmt.Println(strconv.IsPrint('A'))  // true
	fmt.Println(strconv.IsPrint(' '))  // true
	fmt.Println(strconv.IsPrint('\n')) // false
	fmt.Println(strconv.IsGraphic('A')) // true
	fmt.Println(strconv.IsGraphic(' ')) // false (space not graphic)
	fmt.Println("ok")
}
