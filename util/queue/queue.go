package queue

import "container/list"

func New[T any]() *Queue[T] {
	return &Queue[T]{
		l: &list.List{},
	}
}

type Queue[T any] struct {
	l *list.List
}

func (q *Queue[T]) PopFront() T {
	f := q.l.Front()
	vf := f.Value.(T)
	q.l.Remove(f)
	return vf
}

func (q *Queue[T]) PopBack() T {
	f := q.l.Back()
	vf := f.Value.(T)
	q.l.Remove(f)
	return vf
}

func (q *Queue[T]) PushFront(t T) {
	q.l.PushFront(t)
}

func (q *Queue[T]) PushBack(t T) {
	q.l.PushBack(t)
}

func (q *Queue[T]) Len() int {
	return q.l.Len()
}
