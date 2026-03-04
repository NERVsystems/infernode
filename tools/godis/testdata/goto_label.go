package main

func main() {
	// Basic goto
	i := 0
loop:
	if i >= 5 {
		goto done
	}
	i++
	goto loop
done:
	println(i)

	// Goto skipping code
	x := 10
	goto skip
	x = 99
skip:
	println(x)
}
