package main

import (
	"fmt"
	"reflect"
)

func main() {
	// DeepEqual on same values
	fmt.Println(reflect.DeepEqual(42, 42))       // true
	fmt.Println(reflect.DeepEqual("hi", "hi"))    // true
	fmt.Println(reflect.DeepEqual(42, 99))        // false
	fmt.Println(reflect.DeepEqual("hi", "bye"))   // false
	fmt.Println(reflect.DeepEqual(42, "hi"))       // false
	fmt.Println("reflect ok")
}
