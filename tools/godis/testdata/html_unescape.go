package main

import "html"

func main() {
	// EscapeString (already works)
	e := html.EscapeString("<div class=\"foo\">a & b</div>")
	println(e)

	// UnescapeString
	u := html.UnescapeString("&lt;div class=&#34;foo&#34;&gt;a &amp; b&lt;/div&gt;")
	println(u)

	// Round-trip
	orig := "Hello <world> & 'friends' \"everyone\""
	println(html.UnescapeString(html.EscapeString(orig)))

	println("html ok")
}
