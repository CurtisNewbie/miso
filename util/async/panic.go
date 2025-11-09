package async

import (
	"fmt"
	"runtime/debug"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/utillog"
)

func CapturePanicErr(op func()) error {
	var err error
	func() {
		defer func() {
			if v := recover(); v != nil {
				if ve, ok := v.(error); ok {
					err = ve
				} else {
					err = fmt.Errorf("panic captured: %v", v)
				}
			}
		}()
		op()
	}()
	return err
}

func CapturePanic[T any](op func() (T, error)) (T, error) {
	var err error
	var t T
	func() {
		defer func() {
			if v := recover(); v != nil {
				if ve, ok := v.(error); ok {
					err = ve
				} else {
					err = fmt.Errorf("panic captured: %v", v)
				}
			}
		}()
		t, err = op()
	}()
	return t, err
}

func PanicSafeFunc(op func()) func() {
	return func() {
		defer func() {
			if v := recover(); v != nil {
				utillog.ErrorLog("panic recovered, %v\n%v", v, string(debug.Stack()))
			}
		}()
		op()
	}
}

func PanicSafeErrFunc(op func() error) func() error {
	return func() (err error) {
		defer func() {
			if v := recover(); v != nil {
				utillog.ErrorLog("panic recovered, %v\n%v", v, string(debug.Stack()))
				if verr, ok := v.(error); ok {
					err = verr
				} else {
					err = errs.NewErrf("panic recovered, %v", v)
				}
			}
		}()
		err = op()
		return
	}
}

func PanicSafeRun(op func()) {
	PanicSafeFunc(op)()
}

func PanicSafeRunErr(op func() error) error {
	return PanicSafeErrFunc(op)()
}

func recoverPanic() {
	if v := recover(); v != nil {
		utillog.ErrorLog("panic recovered, %v\n%v", v, string(debug.Stack()))
	}
}
