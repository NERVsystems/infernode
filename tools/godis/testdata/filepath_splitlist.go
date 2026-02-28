package main

import "path/filepath"

func main() {
	parts := filepath.SplitList("/usr/bin:/usr/local/bin:/home/user/bin")
	println(len(parts))  // 3
	println(parts[0])    // /usr/bin
	println(parts[1])    // /usr/local/bin
	println(parts[2])    // /home/user/bin

	parts2 := filepath.SplitList("/single")
	println(len(parts2)) // 1
	println(parts2[0])   // /single

	parts3 := filepath.SplitList("")
	println(len(parts3)) // 1
	println(parts3[0])   // (empty)

	println("splitlist ok")
}
