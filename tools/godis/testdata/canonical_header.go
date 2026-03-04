package main

import "net/http"

func main() {
	println(http.CanonicalHeaderKey("content-type"))
	println(http.CanonicalHeaderKey("accept-encoding"))
	println(http.CanonicalHeaderKey("x-forwarded-for"))
	println(http.CanonicalHeaderKey("Content-Type"))
	println(http.CanonicalHeaderKey("CONTENT-TYPE"))
	println(http.CanonicalHeaderKey("host"))
	println(http.CanonicalHeaderKey(""))
	println(http.CanonicalHeaderKey("a"))
	println(http.CanonicalHeaderKey("A"))
}
