package async

import (
	"context"
	"time"
)

type SignalOnce struct {
	c      context.Context
	cancel func()
}

func (s *SignalOnce) Closed() bool {
	return s.c.Err() != nil
}

func (s *SignalOnce) Wait() {
	s.TimedWait(0)
}

func (s *SignalOnce) TimedWait(timeout time.Duration) (isTimeout bool) {
	if timeout < 1 {
		<-s.c.Done()
		return false
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-s.c.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *SignalOnce) Notify() {
	s.cancel()
}

func NewSignalOnce() *SignalOnce {
	c, f := context.WithCancel(context.Background())
	return &SignalOnce{
		c:      c,
		cancel: f,
	}
}
