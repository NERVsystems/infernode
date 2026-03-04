package main

// Generic data structures: Stack and Pair.

type Stack[T any] struct {
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Push(v T) {
	s.items = append(s.items, v)
}

func (s *Stack[T]) Pop() T {
	n := len(s.items)
	v := s.items[n-1]
	s.items = s.items[:n-1]
	return v
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

type Pair[A any, B any] struct {
	First  A
	Second B
}

func MakePair[A any, B any](a A, b B) Pair[A, B] {
	return Pair[A, B]{First: a, Second: b}
}

func Swap[A any, B any](p Pair[A, B]) Pair[B, A] {
	return Pair[B, A]{First: p.Second, Second: p.First}
}

func main() {
	// Stack[int]
	si := NewStack[int]()
	si.Push(10)
	si.Push(20)
	si.Push(30)
	println(si.Len())
	println(si.Pop())
	println(si.Pop())
	println(si.Len())

	// Stack[string]
	ss := NewStack[string]()
	ss.Push("hello")
	ss.Push("world")
	println(ss.Pop())
	println(ss.Pop())

	// Pair[int, string]
	p1 := MakePair(42, "answer")
	println(p1.First)
	println(p1.Second)

	// Swap
	p2 := Swap(p1)
	println(p2.First)
	println(p2.Second)
}
