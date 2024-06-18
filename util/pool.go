package util

import (
	"bytes"
	"sync"
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
