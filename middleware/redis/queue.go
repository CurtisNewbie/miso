package redis

import "github.com/curtisnewbie/miso/miso"

type rqueue[T any] struct {
	key string
}

func (r rqueue[T]) Push(rail miso.Rail, t T) error {
	return RPushJson(rail, r.key, t)
}

func (r rqueue[T]) Pop(rail miso.Rail) (T, bool, error) {
	return RPopJson[T](rail, r.key)
}

func NewRQueue[T any](key string) rqueue[T] {
	return rqueue[T]{}
}
