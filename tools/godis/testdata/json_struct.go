package main

import (
	"encoding/json"
	"fmt"
)

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Tagged struct {
	Visible string `json:"visible"`
	Skip    string `json:"-"`
	Plain   int
}

func main() {
	// Struct with json tags
	p := Person{Name: "Alice", Age: 30}
	b1, _ := json.Marshal(p)
	fmt.Println(string(b1))

	// Struct with skip tag and untagged field
	t := Tagged{Visible: "yes", Skip: "no", Plain: 42}
	b2, _ := json.Marshal(t)
	fmt.Println(string(b2))

	// Map
	m := map[string]int{"x": 1}
	b3, _ := json.Marshal(m)
	fmt.Println(string(b3))

	fmt.Println("json struct ok")
}
