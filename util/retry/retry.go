package retry

func GetOne[T any](retryCount int, f func() (T, error), doRetry func(err error) bool) (T, error) {
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

func Call(retryCount int, f func() error, doRetry func(err error) bool) error {
	_, err := GetOne(retryCount, func() (struct{}, error) {
		return struct{}{}, f()
	}, doRetry)
	return err
}
