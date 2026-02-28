package main

import "bytes"

func main() {
	// n = 2: split into at most 2 parts
	parts := bytes.SplitN([]byte("a,b,c"), []byte(","), 2)
	println(len(parts))           // 2
	println(string(parts[0]))     // a
	println(string(parts[1]))     // b,c

	// n = -1: unlimited (same as Split)
	parts2 := bytes.SplitN([]byte("a,b,c"), []byte(","), -1)
	println(len(parts2))          // 3
	println(string(parts2[0]))    // a
	println(string(parts2[1]))    // b
	println(string(parts2[2]))    // c

	// n = 1: no splitting
	parts3 := bytes.SplitN([]byte("a,b,c"), []byte(","), 1)
	println(len(parts3))          // 1
	println(string(parts3[0]))    // a,b,c

	// n = 0: return nil
	parts4 := bytes.SplitN([]byte("a,b,c"), []byte(","), 0)
	println(parts4 == nil)        // true

	println("bytes splitn ok")
}
