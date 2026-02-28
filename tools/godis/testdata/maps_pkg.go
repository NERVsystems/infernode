package main

import "maps"

func main() {
	// maps.Equal
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"a": 1, "b": 2}
	m3 := map[string]int{"a": 1, "b": 3}
	if maps.Equal(m1, m2) {
		println("m1==m2: true")
	} else {
		println("m1==m2: false")
	}
	if maps.Equal(m1, m3) {
		println("m1==m3: true")
	} else {
		println("m1==m3: false")
	}

	// maps.Clone (shallow)
	m4 := maps.Clone(m1)
	if maps.Equal(m1, m4) {
		println("clone equal: true")
	} else {
		println("clone equal: false")
	}

	println("maps ok")
}
