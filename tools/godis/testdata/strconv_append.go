package main

import "strconv"

func main() {
	// AppendInt
	b1 := strconv.AppendInt(nil, 42, 10)
	println(string(b1)) // 42

	// AppendUint
	b2 := strconv.AppendUint(nil, 123, 10)
	println(string(b2)) // 123

	// AppendBool
	b3 := strconv.AppendBool(nil, true)
	println(string(b3)) // true
	b4 := strconv.AppendBool(nil, false)
	println(string(b4)) // false

	// AppendQuote
	b5 := strconv.AppendQuote(nil, "hello")
	println(string(b5)) // "hello"

	// AppendQuoteRune
	b6 := strconv.AppendQuoteRune(nil, 'A')
	println(string(b6)) // 'A'

	println("ok")
}
