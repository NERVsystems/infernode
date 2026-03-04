package main

import "mime"

func main() {
	println(mime.TypeByExtension(".html"))
	println(mime.TypeByExtension(".json"))
	println(mime.TypeByExtension(".png"))
	println(mime.TypeByExtension(".jpg"))
	println(mime.TypeByExtension(".pdf"))
	println(mime.TypeByExtension(".css"))
	println(mime.TypeByExtension(".js"))
	println(mime.TypeByExtension(".wasm"))
	// Unknown extension
	println(mime.TypeByExtension(".xyz") == "")
}
