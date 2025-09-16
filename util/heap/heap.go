package heap

import (
	"container/heap"
)

var _ heap.Interface = &sliceHeap[any]{}

type sliceHeap[T any] struct {
	Slice    *[]T
	LessFunc func(iv T, jv T) bool
}

func (l *sliceHeap[T]) Len() int {
	return len(*l.Slice)
}

func (l *sliceHeap[T]) Less(i, j int) bool {
	return l.LessFunc((*l.Slice)[i], (*l.Slice)[j])
}

func (l *sliceHeap[T]) Swap(i, j int) {
	(*l.Slice)[i], (*l.Slice)[j] = (*l.Slice)[j], (*l.Slice)[i]
}

func (l *sliceHeap[T]) Push(x any) {
	*l.Slice = append(*l.Slice, x.(T))
}

func (l *sliceHeap[T]) Pop() any {
	prevSlice := *l.Slice
	prevLen := len(prevSlice)
	x := prevSlice[prevLen-1]
	*l.Slice = prevSlice[0 : prevLen-1]
	return x
}

func (l *sliceHeap[T]) Peek() T {
	return (*l.Slice)[0]
}

type Heap[T any] struct {
	heap *sliceHeap[T]
}

func (h *Heap[T]) Len() int {
	return h.heap.Len()
}

func (h *Heap[T]) Push(t T) {
	heap.Push(h.heap, t)
}

func (h *Heap[T]) Pop() T {
	return heap.Pop(h.heap).(T)
}

func (h *Heap[T]) Peek() T {
	return h.heap.Peek()
}

func New[T any](cap int, lessFunc func(iv T, jv T) bool) *Heap[T] {
	if cap < 0 {
		cap = 0
	}
	sl := make([]T, 0, cap)
	h := &Heap[T]{
		heap: &sliceHeap[T]{
			Slice:    &sl,
			LessFunc: lessFunc,
		},
	}
	heap.Init(h.heap)
	return h
}
