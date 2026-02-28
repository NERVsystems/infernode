package main

import "regexp"

func main() {
	println(regexp.QuoteMeta("hello"))           // hello (no change)
	println(regexp.QuoteMeta("hello.world"))     // hello\.world
	println(regexp.QuoteMeta("a+b*c?"))          // a\+b\*c\?
	println(regexp.QuoteMeta("[foo](bar)"))       // \[foo\]\(bar\)
	println(regexp.QuoteMeta("$100"))             // \$100
	println(regexp.QuoteMeta("a{1,2}"))           // a\{1\,2\}  -- wait, comma isn't a metachar
	println(regexp.QuoteMeta("^start|end$"))      // \^start\|end\$
	println(regexp.QuoteMeta(""))                 // (empty)
	println("regexp quote ok")
}
