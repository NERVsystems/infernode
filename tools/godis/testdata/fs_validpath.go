package main

import "io/fs"

func main() {
	// Valid paths
	println(fs.ValidPath("."))
	println(fs.ValidPath("a"))
	println(fs.ValidPath("a/b/c"))
	println(fs.ValidPath("hello.txt"))
	println(fs.ValidPath("dir/file.go"))

	// Invalid paths
	println(fs.ValidPath(""))
	println(fs.ValidPath("/"))
	println(fs.ValidPath("/a"))
	println(fs.ValidPath("a/"))
	println(fs.ValidPath("a//b"))
	println(fs.ValidPath("."))  // duplicate valid for consistency
	println(fs.ValidPath("a/./b"))
	println(fs.ValidPath(".."))
	println(fs.ValidPath("a/../b"))
}
