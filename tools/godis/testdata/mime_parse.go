package main

import "mime"

func main() {
	mediatype, _, _ := mime.ParseMediaType("text/html; charset=utf-8")
	println(mediatype)

	mediatype2, _, _ := mime.ParseMediaType("APPLICATION/JSON")
	println(mediatype2)

	mediatype3, _, _ := mime.ParseMediaType("image/png")
	println(mediatype3)

	mediatype4, _, _ := mime.ParseMediaType("  Text/Plain  ")
	println(mediatype4)
}
