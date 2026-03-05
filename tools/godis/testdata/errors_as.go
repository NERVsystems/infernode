package main

import (
	"errors"
	"fmt"
)

func main() {
	err1 := errors.New("something failed")
	err2 := errors.New("something failed")
	err3 := errors.New("different error")

	// errors.Is — tag comparison
	fmt.Println(errors.Is(err1, err2)) // true (same tag type)
	fmt.Println(errors.Is(err1, err3)) // true (both errorString)

	// errors.As — tag comparison (improved from always-false)
	fmt.Println(errors.As(err1, &err2)) // true (same tag type)

	// errors.Unwrap — returns nil (no wrapped error)
	u := errors.Unwrap(err1)
	fmt.Println(u == nil) // true

	// errors.Join — returns first error
	joined := errors.Join(err1, err3)
	fmt.Println(joined != nil) // true

	fmt.Println("errors ok")
}
