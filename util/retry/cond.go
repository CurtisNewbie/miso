package retry

func GetOneCond[T any](retryCount int, f func() (T, error), doRetryFunc func(t T, err error) bool) (T, error) {
	var (
		n    = 0
		last error
	)

	for n <= retryCount {
		t, err := f()
		if !doRetryFunc(t, err) {
			return t, err
		}
		last = err
		n += 1
	}
	var t T
	return t, last
}
