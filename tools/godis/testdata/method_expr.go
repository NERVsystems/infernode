package main

type Adder struct{ n int }

func (a Adder) Add(x int) int { return a.n + x }
func (a *Adder) Scale(factor int) { a.n *= factor }

func main() {
	// Method expression: Adder.Add
	f := Adder.Add
	println(f(Adder{10}, 5))

	// Method expression on pointer receiver
	a := &Adder{3}
	g := (*Adder).Scale
	g(a, 4)
	println(a.n)
}
