package main

import (
	"fmt"
	"math"
	"time"
)

func nbody(n int) int {
	size := 5
	x := make([]int, size)
	y := make([]int, size)
	vx := make([]int, size)
	vy := make([]int, size)
	mass := make([]int, size)

	x[0] = 0; y[0] = 0; vx[0] = 0; vy[0] = 0; mass[0] = 1000
	x[1] = 100; y[1] = 0; vx[1] = 0; vy[1] = 10; mass[1] = 1
	x[2] = 200; y[2] = 0; vx[2] = 0; vy[2] = 7; mass[2] = 1
	x[3] = 0; y[3] = 150; vx[3] = 8; vy[3] = 0; mass[3] = 1
	x[4] = 0; y[4] = 250; vx[4] = 6; vy[4] = 0; mass[4] = 1

	for step := 0; step < n; step++ {
		for i := 0; i < size; i++ {
			for j := i + 1; j < size; j++ {
				dx := x[j] - x[i]
				dy := y[j] - y[i]
				dist2 := dx*dx + dy*dy
				if dist2 < 1 {
					dist2 = 1
				}
				dist := int(math.Sqrt(float64(dist2)))
				if dist < 1 {
					dist = 1
				}
				force := mass[i] * mass[j] / dist2
				fx := force * dx / dist
				fy := force * dy / dist
				vx[i] += fx / mass[i]
				vy[i] += fy / mass[i]
				vx[j] -= fx / mass[j]
				vy[j] -= fy / mass[j]
			}
		}
		for i := 0; i < size; i++ {
			x[i] += vx[i]
			y[i] += vy[i]
		}
	}
	return x[0] + y[0] + x[1] + y[1]
}

func main() {
	t1 := time.Now()
	iterations := 20
	result := 0
	for iter := 0; iter < iterations; iter++ {
		result += nbody(10000)
	}
	elapsed := time.Since(t1).Milliseconds()
	fmt.Printf("BENCH nbody %d ms %d iters %d\n", elapsed, iterations, result)
}
