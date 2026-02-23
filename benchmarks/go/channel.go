package main

import "inferno/sys"

func producer(ch chan int, n int) {
	i := 0
	for i < n {
		ch <- i
		i = i + 1
	}
	close(ch)
}

func main() {
	t1 := sys.Millisec()
	iterations := 10
	total := 0
	for iter := 0; iter < iterations; iter++ {
		ch := make(chan int, 100)
		go producer(ch, 10000)
		sum := 0
		for v := range ch {
			sum = sum + v
		}
		total = total + sum
	}
	t2 := sys.Millisec()
	println("BENCH channel", t2-t1, "ms", iterations, "iters", total)
}
