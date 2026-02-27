package main

import (
	"fmt"
	"sync/atomic"
)

func printBool(b bool) {
	if b {
		fmt.Println("true")
	} else {
		fmt.Println("false")
	}
}

func main() {
	// AddInt32
	var x int32 = 10
	result := atomic.AddInt32(&x, 5)
	fmt.Println(result) // 15
	fmt.Println(x)      // 15

	// LoadInt32
	val := atomic.LoadInt32(&x)
	fmt.Println(val) // 15

	// StoreInt32
	atomic.StoreInt32(&x, 42)
	fmt.Println(x) // 42

	// SwapInt32
	old := atomic.SwapInt32(&x, 100)
	fmt.Println(old) // 42
	fmt.Println(x)   // 100

	// CompareAndSwap - success (x is 100, expect 100)
	swapped := atomic.CompareAndSwapInt32(&x, 100, 200)
	printBool(swapped) // true
	fmt.Println(x)     // 200

	// CompareAndSwap - failure (x is 200, expect 999)
	swapped = atomic.CompareAndSwapInt32(&x, 999, 300)
	printBool(swapped) // false
	fmt.Println(x)     // 200

	// Int64 operations
	var y int64 = 0
	atomic.AddInt64(&y, 1000)
	fmt.Println(y) // 1000
	atomic.StoreInt64(&y, 2000)
	fmt.Println(atomic.LoadInt64(&y)) // 2000
}
