package main

import "net/url"

func main() {
	// QueryEscape
	println(url.QueryEscape("hello world"))
	println(url.QueryEscape("foo=bar&baz=qux"))
	println(url.QueryEscape("simple"))
	println(url.QueryEscape("100%"))

	// QueryUnescape
	s1, _ := url.QueryUnescape("hello+world")
	println(s1)
	s2, _ := url.QueryUnescape("foo%3Dbar%26baz%3Dqux")
	println(s2)
	s3, _ := url.QueryUnescape("simple")
	println(s3)
	s4, _ := url.QueryUnescape("100%25")
	println(s4)

	// Round-trip
	orig := "hello world & friends=100%"
	encoded := url.QueryEscape(orig)
	decoded, _ := url.QueryUnescape(encoded)
	println(decoded)

	println("url escape ok")
}
