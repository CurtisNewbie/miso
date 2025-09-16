package stack

import (
	"fmt"
	"slices"
)

type Stack[T any] struct {
	st []T
	p  int
}

func (s *Stack[T]) Push(v T) {
	s.st = append(s.st, v)
	s.p++
}

func (s *Stack[T]) Pop() (T, bool) {
	var v T
	if s.p < 0 {
		return v, false
	}
	v = s.st[s.p]

	s.st = slices.Delete(s.st, s.p, s.p+1)
	s.p--
	return v, true
}

func (s *Stack[T]) Peek() (T, bool) {
	var v T
	if s.p < 0 {
		return v, false
	}
	return s.st[s.p], true
}

func (s *Stack[T]) Empty() bool {
	return s.p < 0
}

func (s *Stack[T]) Len() int {
	return s.p
}

func (s Stack[T]) String() string {
	return fmt.Sprintf("%+v", s.st)
}

// Iterate Stack from top to bottom, iter func can return false to break the loop.
func (s *Stack[T]) ForEach(f func(v T) bool) {
	if s.p < 0 {
		return
	}
	for i := s.p; i >= 0; i-- {
		v := s.st[i]
		if !f(v) {
			return
		}
	}
}

func (s *Stack[T]) Slice() []T {
	return slices.Clone(s.st)
}

func New[T any](cap int) *Stack[T] {
	if cap < 0 {
		cap = 0
	}
	return &Stack[T]{
		st: make([]T, 0, cap),
		p:  -1,
	}
}
