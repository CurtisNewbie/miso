package retry

import (
	"time"
)

func GetOne[T any](retryCount int, f func() (T, error), doRetryFuncs ...func(err error) bool) (T, error) {
	var (
		n       = 0
		last    error
		doRetry func(err error) bool
	)
	if len(doRetryFuncs) > 0 {
		doRetry = doRetryFuncs[0]
	} else {
		doRetry = func(err error) bool { return true }
	}

	for n <= retryCount {
		t, err := f()
		if err == nil {
			return t, nil
		}
		if !doRetry(err) {
			return t, err
		}
		last = err
		n += 1
	}
	var t T
	return t, last
}

func Call(retryCount int, f func() error, doRetryFuncs ...func(err error) bool) error {
	_, err := GetOne(retryCount, func() (struct{}, error) {
		return struct{}{}, f()
	}, doRetryFuncs...)
	return err
}

func GetOneWithBackoff[T any](backoff []time.Duration, f func() (T, error)) (T, error) {
	var (
		i    = 0
		last error
	)

	for i <= len(backoff) {
		t, err := f()
		if err == nil {
			return t, nil
		}
		if i < len(backoff) {
			time.Sleep(backoff[i])
		}
		last = err
		i++
	}
	var t T
	return t, last
}

func CallWithBackoff(backoff []time.Duration, f func() error) error {
	_, err := GetOneWithBackoff(backoff, func() (struct{}, error) {
		return struct{}{}, f()
	})
	return err
}

// GetOneDyn retries f indefinitely until it succeeds or gapFunc returns doRetry=false.
// gapFunc(i, err) is called with the current attempt index (1-based) and the error to determine
// the sleep duration before the next retry and whether to continue retrying.
func GetOneDyn[T any](f func() (T, error), gapFunc func(i int, err error) (wait time.Duration, doRetry bool)) (T, error) {
	i := 1
	for {
		t, err := f()
		if err == nil {
			return t, nil
		}
		wait, doRetry := gapFunc(i, err)
		if !doRetry {
			return t, err
		}
		time.Sleep(wait)
		i++
	}
}
