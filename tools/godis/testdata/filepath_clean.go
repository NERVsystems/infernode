package main

import "path/filepath"

func main() {
	// Identity (already clean)
	println(filepath.Clean("/foo/bar"))     // /foo/bar
	println(filepath.Clean("foo/bar"))      // foo/bar

	// Double slashes
	println(filepath.Clean("/foo//bar"))    // /foo/bar
	println(filepath.Clean("foo///bar"))    // foo/bar

	// Dot components
	println(filepath.Clean("/foo/./bar"))   // /foo/bar
	println(filepath.Clean("./foo"))        // foo

	// Dotdot components
	println(filepath.Clean("/foo/bar/../baz"))  // /foo/baz
	println(filepath.Clean("foo/bar/.."))       // foo

	// Trailing slash
	println(filepath.Clean("/foo/bar/"))    // /foo/bar

	// Root
	println(filepath.Clean("/"))            // /

	// Empty
	println(filepath.Clean(""))            // .

	// Just dot
	println(filepath.Clean("."))           // .

	// Dotdot at root (absorbed)
	println(filepath.Clean("/../foo"))     // /foo

	// Relative dotdot
	println(filepath.Clean("../foo"))      // ../foo
}
