package main

import "bytes"

func main() {
	// Cut
	before, after, found := bytes.Cut([]byte("hello=world"), []byte("="))
	println(string(before)) // hello
	println(string(after))  // world
	if found {
		println("found")
	}

	before2, after2, found2 := bytes.Cut([]byte("nope"), []byte("="))
	println(string(before2)) // nope
	println(string(after2))  // (empty)
	if !found2 {
		println("not found")
	}

	// CutPrefix
	rest, ok := bytes.CutPrefix([]byte("hello world"), []byte("hello "))
	println(string(rest)) // world
	if ok {
		println("prefix ok")
	}

	rest2, ok2 := bytes.CutPrefix([]byte("abc"), []byte("xyz"))
	println(string(rest2)) // abc
	if !ok2 {
		println("prefix not ok")
	}

	// CutSuffix
	rest3, ok3 := bytes.CutSuffix([]byte("hello world"), []byte(" world"))
	println(string(rest3)) // hello
	if ok3 {
		println("suffix ok")
	}

	rest4, ok4 := bytes.CutSuffix([]byte("abc"), []byte("xyz"))
	println(string(rest4)) // abc
	if !ok4 {
		println("suffix not ok")
	}

	println("bytes cut ok")
}
