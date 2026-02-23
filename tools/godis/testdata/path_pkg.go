package main

import "path"

func main() {
	// Base
	println(path.Base("/a/b/c"))
	println(path.Base("/"))
	println(path.Base(""))

	// Dir
	println(path.Dir("/a/b/c"))
	println(path.Dir("/a"))

	// Ext
	println(path.Ext("hello.go"))
	println(path.Ext("noext"))

	// Join
	println(path.Join("a", "b", "c"))
}
