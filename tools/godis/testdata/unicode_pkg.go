package main

import "unicode"

func main() {
	// IsLetter
	if unicode.IsLetter('A') {
		println("letter")
	}
	if !unicode.IsLetter('5') {
		println("not letter")
	}

	// IsDigit
	if unicode.IsDigit('7') {
		println("digit")
	}
	if !unicode.IsDigit('x') {
		println("not digit")
	}

	// IsSpace
	if unicode.IsSpace(' ') {
		println("space")
	}
	if !unicode.IsSpace('a') {
		println("not space")
	}

	// IsUpper / IsLower
	if unicode.IsUpper('Z') {
		println("upper")
	}
	if unicode.IsLower('a') {
		println("lower")
	}

	// ToUpper / ToLower
	println(unicode.ToUpper('a')) // 65 ('A')
	println(unicode.ToLower('Z')) // 122 ('z')
}
