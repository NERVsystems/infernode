package main

import (
	"fmt"
	"path"
)

func main() {
	// Base
	fmt.Println(path.Base("/foo/bar/baz.go")) // baz.go
	fmt.Println(path.Base("/foo/bar"))         // bar
	fmt.Println(path.Base("hello"))            // hello

	// Dir
	fmt.Println(path.Dir("/foo/bar/baz.go")) // /foo/bar
	fmt.Println(path.Dir("/foo"))            // /
	fmt.Println(path.Dir("hello"))           // .

	// Ext
	fmt.Println(path.Ext("foo.go"))     // .go
	fmt.Println(path.Ext("foo.tar.gz")) // .gz
	fmt.Println(path.Ext("foo"))        //

	// Join
	fmt.Println(path.Join("foo", "bar", "baz")) // foo/bar/baz
}
