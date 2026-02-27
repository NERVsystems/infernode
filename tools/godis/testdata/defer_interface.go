package main

import "fmt"

type Closer interface {
	Close()
}

type File struct {
	name string
}

func (f *File) Close() {
	fmt.Println("closed " + f.name)
}

func process(c Closer) {
	defer c.Close()
	fmt.Println("processing")
}

func main() {
	f := &File{name: "test.txt"}
	process(f)
}
