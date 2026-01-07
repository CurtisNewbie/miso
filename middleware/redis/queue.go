package redis

import (
	"iter"

	"github.com/curtisnewbie/miso/miso"
)

type rqueue[T any] struct {
	key string
}

func (r rqueue[T]) Push(rail miso.Rail, t T) error {
	return RPushJson(rail, r.key, t)
}

func (r rqueue[T]) Pop(rail miso.Rail) (T, bool, error) {
	return RPopJson[T](rail, r.key)
}

func (r rqueue[T]) PopAll(rail miso.Rail) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for {
			c, ok, err := r.Pop(rail)
			if err != nil {
				if !yield(c, err) {
					return
				}
				continue
			}
			if !ok { // empty
				return
			}
			if !yield(c, nil) {
				return
			}
		}
	}
}

func NewRQueue[T any](key string) rqueue[T] {
	return rqueue[T]{}
}
