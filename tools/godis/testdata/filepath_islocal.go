package main

import "path/filepath"

func main() {
	// Local paths → true
	println(filepath.IsLocal("foo"))        // true
	println(filepath.IsLocal("foo/bar"))    // true
	println(filepath.IsLocal("a.txt"))      // true

	// Empty → false
	println(filepath.IsLocal(""))           // false

	// Absolute → false
	println(filepath.IsLocal("/foo"))       // false
	println(filepath.IsLocal("/"))          // false

	// Contains ".." → false
	println(filepath.IsLocal(".."))         // false
	println(filepath.IsLocal("../foo"))     // false
	println(filepath.IsLocal("foo/.."))     // false
	println(filepath.IsLocal("foo/../bar")) // false

	// ".." not as component → true
	println(filepath.IsLocal("foo..bar"))   // true
	println(filepath.IsLocal("..foo"))      // true

	println("islocal ok")
}
