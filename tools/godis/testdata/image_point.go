package main

import "image"

func main() {
	p := image.Pt(3, 4)
	q := image.Pt(1, 2)

	// Add
	r := p.Add(q)
	println(r.X) // 4
	println(r.Y) // 6

	// Sub
	s := p.Sub(q)
	println(s.X) // 2
	println(s.Y) // 2

	// Mul
	m := p.Mul(3)
	println(m.X) // 9
	println(m.Y) // 12

	// Div
	d := p.Div(2)
	println(d.X) // 1
	println(d.Y) // 2

	// Eq
	println(p.Eq(image.Pt(3, 4))) // true
	println(p.Eq(q))              // false

	// In
	rect := image.Rect(0, 0, 5, 5)
	println(p.In(rect))              // true
	println(image.Pt(5, 5).In(rect)) // false (max is exclusive)
	println(image.Pt(-1, 0).In(rect)) // false

	println("point ok")
}
