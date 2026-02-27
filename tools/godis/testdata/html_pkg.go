package main

import (
	"fmt"
	"html"
)

func main() {
	// Basic escaping
	fmt.Println(html.EscapeString("hello"))
	fmt.Println(html.EscapeString("<b>bold</b>"))
	fmt.Println(html.EscapeString("a & b"))
	fmt.Println(html.EscapeString("x < y > z"))
	fmt.Println(html.EscapeString(`say "hello"`))
	fmt.Println(html.EscapeString("it's"))
	// All special chars
	fmt.Println(html.EscapeString(`<a href="x&y">`))
	// Empty string
	fmt.Println(html.EscapeString(""))
}
