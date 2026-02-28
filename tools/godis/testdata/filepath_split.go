package main

import "path/filepath"

func main() {
	dir, file := filepath.Split("/home/user/file.txt")
	println(dir)  // /home/user/
	println(file) // file.txt

	dir2, file2 := filepath.Split("file.txt")
	println(dir2)  // (empty)
	println(file2) // file.txt

	dir3, file3 := filepath.Split("/usr/local/")
	println(dir3)  // /usr/local/
	println(file3) // (empty)

	dir4, file4 := filepath.Split("/")
	println(dir4)  // /
	println(file4) // (empty)

	println("filepath split ok")
}
