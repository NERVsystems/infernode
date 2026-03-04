package main

import "os"

func main() {
	// Write a file
	data := []byte("hello from godis")
	err := os.WriteFile("/tmp/godis_rw_test", data, 0644)
	if err != nil {
		println("write error")
		return
	}

	// Read it back
	got, err := os.ReadFile("/tmp/godis_rw_test")
	if err != nil {
		println("read error")
		return
	}
	println(string(got))
	println(len(got))

	// Clean up
	os.Remove("/tmp/godis_rw_test")
	println("ok")
}
