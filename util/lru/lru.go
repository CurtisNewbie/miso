package lru

import (
	"context"

	"github.com/curtisnewbie/miso/miso"
	"github.com/maypok86/otter/v2"
)

type LRU[T any] interface {
	Get(k string) (T, bool)
	GetElse(k string, f func() (T, error)) (T, error)
	Set(k string, t T)
}

type lru[T any] struct {
	c *otter.Cache[string, T]
}

func (l *lru[T]) Get(k string) (T, bool) {
	v, ok := l.c.GetEntry(k)
	if !ok {
		var t T
		return t, false
	}
	return v.Value, true
}

func (l *lru[T]) GetElse(k string, f func() (T, error)) (T, error) {
	return l.c.Get(context.Background(), k, otter.LoaderFunc[string, T](func(ctx context.Context, k string) (T, error) {
		return f()
	}))
}

func (l *lru[T]) Set(k string, t T) {
	l.c.Set(k, t)
}

func New[T any](cap int) (LRU[T], error) {
	c, err := otter.New(&otter.Options[string, T]{MaximumSize: cap, Logger: otterLogger{r: miso.EmptyRail()}})
	if err != nil {
		return nil, err
	}
	return &lru[T]{
		c: c,
	}, nil
}

type otterLogger struct {
	r miso.Rail
}

func (o otterLogger) Warn(ctx context.Context, msg string, err error) {
	o.r.Warn(msg, err)
}

func (o otterLogger) Error(ctx context.Context, msg string, err error) {
	o.r.Error(msg, err)
}
