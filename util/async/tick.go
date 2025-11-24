package async

import (
	"sync"
	"time"
)

// Runner that triggers run on every tick.
//
// Create TickRunner using func NewTickRunner(...).
type TickRunner struct {
	task   func()
	ticker *time.Ticker
	mu     sync.Mutex
	freq   time.Duration
}

func NewTickRuner(freq time.Duration, run func()) *TickRunner {
	return &TickRunner{task: run, freq: freq}
}

func (t *TickRunner) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ticker != nil {
		return
	}

	t.ticker = time.NewTicker(t.freq)
	c := t.ticker.C
	go func() {
		for {
			t.task()
			<-c
		}
	}()
}

func (t *TickRunner) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ticker == nil {
		return
	}
	t.ticker.Stop()
	t.ticker = nil
}
