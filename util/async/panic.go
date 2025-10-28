package async

import (
	"runtime/debug"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/utillog"
)

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
