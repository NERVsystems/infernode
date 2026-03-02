package main

import (
	"crypto/hmac"
	"fmt"
)

func main() {
	a := []byte{1, 2, 3, 4, 5}
	b := []byte{1, 2, 3, 4, 5}
	c := []byte{1, 2, 3, 4, 6}
	d := []byte{1, 2, 3}

	fmt.Println(hmac.Equal(a, b)) // true
	fmt.Println(hmac.Equal(a, c)) // false - different last byte
	fmt.Println(hmac.Equal(a, d)) // false - different length
	fmt.Println("hmac ok")
}
