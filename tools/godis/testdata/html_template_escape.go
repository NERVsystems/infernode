package main

import "html/template"

func main() {
	println(template.HTMLEscapeString("<div>hello & world</div>"))
	println(template.HTMLEscapeString("safe text"))
	println(template.HTMLEscapeString(`"air quotes"`))
}
