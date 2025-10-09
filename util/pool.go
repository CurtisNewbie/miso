package util

import (
	"bytes"
	"container/list"
	"sync"

	"github.com/curtisnewbie/miso/util/rfutil"
)

type ByteBufPool struct {
	po *sync.Pool

	MaxCap int // default max cap is 4096 bytes.
}

func (b *ByteBufPool) Get() *bytes.Buffer {
	buf := b.po.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (b *ByteBufPool) Put(buf *bytes.Buffer) {
	if buf.Cap() > b.MaxCap {
		return
	}
	b.po.Put(buf)
}

func NewByteBufferPool(initCap int) *ByteBufPool {
	var f func() any
	if initCap > 0 {
		f = func() any { return bytes.NewBuffer(make([]byte, 0, initCap)) }
	} else {
		f = func() any { return &bytes.Buffer{} }
	}
	return &ByteBufPool{
		po: &sync.Pool{
			New: f,
		},
		MaxCap: 4096,
	}
}

type FixedPool[T any] struct {
	ch            chan T
	popFilterFunc func(t T) (dropped bool)
}

func FixedPoolFilterFunc[T any](filterFunc func(t T) (dropped bool)) func(*FixedPool[T]) {
	return func(f *FixedPool[T]) {
		f.popFilterFunc = filterFunc
	}
}

func NewFixedPool[T any](cap int, options ...func(*FixedPool[T])) *FixedPool[T] {
	f := new(FixedPool[T])
	f.ch = make(chan T, cap)
	for _, op := range options {
		op(f)
	}
	return f
}

func (r *FixedPool[T]) Push(t T) {
	r.ch <- t
}

func (r *FixedPool[T]) TryPush(t T) bool {
	select {
	case r.ch <- t:
		return true
	default:
		return false
	}
}

func (r *FixedPool[T]) Pop() (T, bool) {
	for c := range r.ch {
		if r.popFilterFunc != nil && r.popFilterFunc(c) {
			continue
		}
		return c, true
	}
	return rfutil.NewVar[T](), false
}

func (r *FixedPool[T]) TryPop() (T, bool) {
	for {
		select {
		case v := <-r.ch:
			if r.popFilterFunc != nil && r.popFilterFunc(v) {
				continue
			}
			return v, true
		default:
			return rfutil.NewVar[T](), false
		}
	}
}

type EphPool[T any] struct {
	list       *list.List
	mu         *sync.Mutex
	filterFunc func(t T) (dropped bool)
}

func NewEphPool[T any](filterFunc func(t T) (dropped bool)) *EphPool[T] {
	f := new(EphPool[T])
	f.mu = &sync.Mutex{}
	f.list = list.New()
	return f
}

func (p *EphPool[T]) Push(t T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.list.PushFront(t)
}

func (p *EphPool[T]) Pop() (T, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for {
		f := p.list.Back()
		if f == nil {
			var t T
			return t, false
		}
		vf := f.Value.(T)
		p.list.Remove(f)

		if p.filterFunc != nil && p.filterFunc(vf) {
			continue
		}
		return vf, true
	}
}
