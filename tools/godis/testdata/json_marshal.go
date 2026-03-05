package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	// Marshal string
	b, err := json.Marshal("hello")
	if err != nil {
		fmt.Println("error")
	} else {
		fmt.Println(string(b))
	}

	// Marshal int
	b, err = json.Marshal(42)
	if err != nil {
		fmt.Println("error")
	} else {
		fmt.Println(string(b))
	}

	// Marshal bool
	b, err = json.Marshal(true)
	if err != nil {
		fmt.Println("error")
	} else {
		fmt.Println(string(b))
	}

	b, err = json.Marshal(false)
	if err != nil {
		fmt.Println("error")
	} else {
		fmt.Println(string(b))
	}

	fmt.Println("json marshal ok")
}
