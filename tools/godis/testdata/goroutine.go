package main

func worker(id int) {
	println(id)
}

func main() {
	go worker(1)
	go worker(2)
	go worker(3)
	// busy wait (crude, will be replaced by channels)
	x := 0
	for x < 100000 {
		x = x + 1
	}
	println("done")
}
