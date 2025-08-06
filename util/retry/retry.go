package retry

func GetOne[T any](f func() (T, error), retryCount int, doRetry func(err error) bool) (T, error) {
	var (
		n    = 0
		last error
	)
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

func Call(f func() error, retryCount int, doRetry func(err error) bool) error {
	_, err := GetOne(func() (struct{}, error) {
		return struct{}{}, f()
	}, retryCount, doRetry)
	return err
}
