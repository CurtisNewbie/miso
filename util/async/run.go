package async

import (
	"context"
	"time"
)

// Keep running until cancelled.
func RunCancellable(f func()) (cancel func()) {
	cr, c := context.WithCancel(context.Background())
	cancel = c
	go func() {
		for {
			select {
			case <-cr.Done():
				return
			default:
				PanicSafeRun(f)
			}
		}
	}()
	return
}

// Keep consuming from ch until cancelled.
func RunCancellableChan[T any](ch <-chan T, f func(t T) (stop bool)) (cancel func()) {
	cr, c := context.WithCancel(context.Background())
	cancel = c
	go func() {
		for {
			select {
			case <-cr.Done():
				return
			case t := <-ch:
				stop := false
				PanicSafeRun(func() {
					stop = f(t)
				})
				if stop {
					return
				}
			}
		}
	}()
	return
}

// Keep running f until dur pass.
func RunUntil[T any](dur time.Duration, f func() (stop bool, t T, e error)) (T, error) {
	ct, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	return Run[T](func() (T, error) {
		for {
			select {
			case <-ct.Done():
				var t T
				return t, nil
			default:
				stop, t, err := f()
				if stop {
					return t, err
				}
			}
		}
	}).Get()
}
