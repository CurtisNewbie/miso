package src

import (
	"runtime"
	"sync"
	"unsafe"
)

var (
	GetCallerFn    = getCallerFn
	GetCallerFnUpN = getCallerFnUpN
)

const (
	fnWidth = 30
)

// reduce alloc, logger calls getCallerFn very frequently, we have to optimize it as much as possible.
var callerUintptrPool = sync.Pool{
	New: func() any {
		p := make([]uintptr, 4)
		return &p
	},
}

// type callerFileLine struct {
// 	Func string
// 	File string
// 	Line int
// }

// func getCallerFileLine() callerFileLine {
// 	pcs := callerUintptrPool.Get().(*[]uintptr)
// 	defer putCallerUintptrPool(pcs)

// 	depth := runtime.Callers(3, *pcs)
// 	frames := runtime.CallersFrames((*pcs)[:depth])

// 	// we only need the first frame
// 	for f, next := frames.Next(); next; {
// 		return callerFileLine{
// 			Func: unsafeGetShortFnName(f.Function),
// 			File: path.Base(f.File),
// 			Line: f.Line,
// 		}
// 	}
// 	return callerFileLine{}
// }

func getCallerFnUpN(n int) string {
	pcs := callerUintptrPool.Get().(*[]uintptr)
	defer putCallerUintptrPool(pcs)

	depth := runtime.Callers(3+n, *pcs)
	frames := runtime.CallersFrames((*pcs)[:depth])

	// we only need the first frame
	for f, next := frames.Next(); next; {
		return unsafeGetShortFnName(f.Function)
	}
	return ""
}

func getCallerFn() string {
	pcs := callerUintptrPool.Get().(*[]uintptr)
	defer putCallerUintptrPool(pcs)

	depth := runtime.Callers(3, *pcs)
	frames := runtime.CallersFrames((*pcs)[:depth])

	// we only need the first frame
	for f, next := frames.Next(); next; {
		return unsafeGetShortFnName(f.Function)
	}
	return ""
}

func putCallerUintptrPool(pcs *[]uintptr) {
	for i := range *pcs {
		(*pcs)[i] = 0 // zero the values, just in case
	}
	callerUintptrPool.Put(pcs)
}

func unsafeGetShortFnName(fn string) string {
	if fn == "" {
		return fn
	}

	trimLengthyName := func(fnb []byte) string {
		const maxDotCnt = 2
		if len(fnb) > fnWidth {
			dcnt := 0
			for i := len(fnb) - 1; i >= 0; i-- {
				ib := fnb[i]
				if ib == '.' && (i-1 < 0 || fnb[i-1] != '.') {
					dcnt += 1
					if dcnt > maxDotCnt {
						return unsafeByt2Str(fnb[i+1:])
					}
				}
			}
		}
		return unsafeByt2Str(fnb)
	}

	fnb := unsafeStr2Byt(fn)
	for i := len(fnb) - 1; i >= 0; i-- {
		ib := fnb[i]
		switch ib {
		case '/':
			if i+1 < len(fnb) {
				return trimLengthyName(fnb[i+1:])
			}
			return trimLengthyName(fnb[i:])
		case '(':
			return trimLengthyName(fnb[i:])
		}
	}
	return fn
}

func unsafeByt2Str(b []byte) string {
	if len(b) < 1 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func unsafeStr2Byt(s string) (b []byte) {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
