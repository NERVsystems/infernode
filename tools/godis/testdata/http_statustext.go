package main

import "net/http"

func main() {
	println(http.StatusText(200))
	println(http.StatusText(201))
	println(http.StatusText(204))
	println(http.StatusText(301))
	println(http.StatusText(302))
	println(http.StatusText(304))
	println(http.StatusText(400))
	println(http.StatusText(401))
	println(http.StatusText(403))
	println(http.StatusText(404))
	println(http.StatusText(405))
	println(http.StatusText(500))
	println(http.StatusText(502))
	println(http.StatusText(503))

	// Unknown code returns empty string
	println(http.StatusText(999) == "")
}
