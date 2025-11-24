package async

import (
	"time"
)

type DoneWatcher struct {
	tr         *TickRunner
	onFinished func()
}

func (w *DoneWatcher) Done() {
	w.tr.Stop()
	w.onFinished()
}

func NewDoneWatcher(interval time.Duration, onEveryCheck func(), onFinished func()) *DoneWatcher {
	if onEveryCheck == nil {
		onEveryCheck = func() {}
	}
	tr := NewTickRuner(interval, onEveryCheck)
	tr.Start()
	return &DoneWatcher{tr: tr}
}
