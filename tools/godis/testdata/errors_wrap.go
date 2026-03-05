package main

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("not found")

func main() {
	// Wrap an error with fmt.Errorf %w
	wrapped := fmt.Errorf("item: %w", ErrNotFound)
	fmt.Println(wrapped.Error())

	// errors.Is should find the sentinel through wrapping
	fmt.Println(errors.Is(wrapped, ErrNotFound))

	// errors.Unwrap should return the inner error
	inner := errors.Unwrap(wrapped)
	if inner != nil {
		fmt.Println(inner.Error())
	} else {
		fmt.Println("<nil>")
	}

	// Non-wrapped error: Unwrap returns nil
	plain := errors.New("plain error")
	fmt.Println(errors.Unwrap(plain) == nil)

	fmt.Println("errors wrap ok")
}
