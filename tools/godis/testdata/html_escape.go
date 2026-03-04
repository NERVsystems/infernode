package main

import "html"

func main() {
	println(html.EscapeString("hello"))
	println(html.EscapeString("<script>alert('xss')</script>"))
	println(html.EscapeString("a & b"))
	println(html.EscapeString(`"quoted"`))
	println(html.EscapeString(""))
	println(html.EscapeString("no special chars"))

	println(html.UnescapeString("&amp;"))
	println(html.UnescapeString("&lt;b&gt;bold&lt;/b&gt;"))
	println(html.UnescapeString("&#34;quoted&#34;"))
	println(html.UnescapeString("no entities"))
}
