package main

import (
	"fmt"
	"reflect"
)

func main() {
	// TypeOf returns the type name
	t1 := reflect.TypeOf(42)
	fmt.Println(t1.String()) // int

	t2 := reflect.TypeOf("hello")
	fmt.Println(t2.String()) // string

	t3 := reflect.TypeOf(true)
	fmt.Println(t3.String()) // bool

	t4 := reflect.TypeOf(3.14)
	fmt.Println(t4.String()) // float64

	// ValueOf + Int
	v := reflect.ValueOf(42)
	fmt.Println(v.Int()) // 42

	fmt.Println("reflect typeof ok")
}
