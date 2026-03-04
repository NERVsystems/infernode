package main

import "os"

func main() {
	// Create a file and write to it
	f, err := os.Create("/tmp/godis_file_ops_test")
	if err != nil {
		println("create error")
		return
	}
	n, err := f.Write([]byte("hello"))
	println(n) // 5

	// WriteString
	n2, err := f.WriteString(" world")
	println(n2) // 6

	f.Close()

	// Open and read
	f2, err := os.Open("/tmp/godis_file_ops_test")
	if err != nil {
		println("open error")
		return
	}
	buf := make([]byte, 20)
	n3, err := f2.Read(buf)
	println(n3) // 11

	f2.Close()

	// Clean up
	os.Remove("/tmp/godis_file_ops_test")
	println("ok")
}
