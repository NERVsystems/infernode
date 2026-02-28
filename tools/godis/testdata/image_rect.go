package main

import "image"

func main() {
	r := image.Rect(1, 2, 5, 6)

	// Dx, Dy
	println(r.Dx()) // 4
	println(r.Dy()) // 4

	// Size
	sz := r.Size()
	println(sz.X) // 4
	println(sz.Y) // 4

	// Empty
	println(r.Empty())                       // false
	println(image.Rect(0, 0, 0, 0).Empty())  // true
	println(image.Rect(5, 5, 3, 3).Empty())  // true

	// Add/Sub point
	p := image.Pt(10, 20)
	ra := r.Add(p)
	println(ra.Min.X) // 11
	println(ra.Max.Y) // 26
	rs := r.Sub(p)
	println(rs.Min.X) // -9
	println(rs.Max.Y) // -14

	// Inset
	ri := r.Inset(1)
	println(ri.Min.X) // 2
	println(ri.Min.Y) // 3
	println(ri.Max.X) // 4
	println(ri.Max.Y) // 5

	// Intersect
	s := image.Rect(3, 4, 8, 9)
	inter := r.Intersect(s)
	println(inter.Min.X) // 3
	println(inter.Min.Y) // 4
	println(inter.Max.X) // 5
	println(inter.Max.Y) // 6

	// Union
	u := r.Union(s)
	println(u.Min.X) // 1
	println(u.Min.Y) // 2
	println(u.Max.X) // 8
	println(u.Max.Y) // 9

	// Eq
	println(r.Eq(image.Rect(1, 2, 5, 6))) // true
	println(r.Eq(s))                       // false

	// Overlaps
	println(r.Overlaps(s))                          // true
	println(r.Overlaps(image.Rect(10, 10, 20, 20))) // false

	// Canon
	c := image.Rect(5, 6, 1, 2).Canon()
	println(c.Min.X) // 1
	println(c.Min.Y) // 2
	println(c.Max.X) // 5
	println(c.Max.Y) // 6

	println("rect ok")
}
