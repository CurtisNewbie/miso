package lua

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cast"
	glua "github.com/yuin/gopher-lua"
)

type Log interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Run Lua Script.
//
// Use With***() funcs to set global variables.
//
// If [WithLogger] is provided, infof(...) and errorf(...) are builtin funcs that can be called inside the lua scripts for logging.
//
// E.g., [WithGlobalStr], [WithGlobalNum], [WithGlobalBool]
//
// The returned value is not unwrapped and thus remains in lua type.
func Run(script string, ops ...func(*glua.LState)) (any, error) {
	st := glua.NewState()
	defer st.Close()

	// default implementation for logging
	st.SetGlobal("infof", loggerFunc(func(s string, a ...any) {
		fmt.Printf("[INFO] "+s+"\n", a...)
	}, st))
	st.SetGlobal("errorf", loggerFunc(func(s string, a ...any) {
		fmt.Printf("[ERROR] "+s+"\n", a...)
	}, st))

	for _, op := range ops {
		op(st)
	}
	if err := st.DoString(script); err != nil {
		return nil, err
	}

	var res any = nil
	if st.GetTop() > 0 {
		res = st.Get(1)
	}
	return res, nil
}

// Run Lua Script via Reader.
//
// Use With***() funcs to set global variables.
//
// If [WithLogger] is provided, infof(...) and errorf(...) are builtin funcs that can be called inside the lua scripts for logging.
//
// E.g., [WithGlobalStr], [WithGlobalNum], [WithGlobalBool]
//
// The returned value is not unwrapped and thus remains in lua type.
func RunReader(f io.Reader, ops ...func(*glua.LState)) (any, error) {
	byt, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return Run(string(byt), ops...)
}

// Run Lua Script File.
//
// Use With***() funcs to set global variables.
//
// If [WithLogger] is provided, infof(...) and errorf(...) are builtin funcs that can be called inside the lua scripts for logging.
//
// E.g., [WithGlobalStr], [WithGlobalNum], [WithGlobalBool]
//
// The returned value is not unwrapped and thus remains in lua type.
func RunFile(path string, ops ...func(*glua.LState)) (any, error) {
	byt, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Run(string(byt), ops...)
}

func WithLogger(rail Log) func(l *glua.LState) {
	return func(l *glua.LState) {
		luaInfof(rail, l)
		luaErrorf(rail, l)
	}
}

func WithGlobalStr(name string, v string) func(*glua.LState) {
	return func(l *glua.LState) {
		l.SetGlobal(name, glua.LString(v))
	}
}

func WithGlobalNum(name string, v int64) func(*glua.LState) {
	return func(l *glua.LState) {
		l.SetGlobal(name, glua.LNumber(v))
	}
}

func WithGlobalBool(name string, v bool) func(*glua.LState) {
	return func(l *glua.LState) {
		l.SetGlobal(name, glua.LBool(v))
	}
}

func WithGlobalNil(name string) func(*glua.LState) {
	return func(l *glua.LState) {
		l.SetGlobal(name, glua.LNil)
	}
}

func WithGlobalStrTable(name string, m map[string]any) func(*glua.LState) {
	return func(l *glua.LState) {
		tb := l.NewTable()
		for k, v := range m {
			var lv glua.LValue
			switch vs := v.(type) {
			case string:
				lv = glua.LString(vs)
			case int, int8, int16, int32, int64, float32, float64:
				lv = glua.LNumber(cast.ToFloat64(vs))
			case bool:
				lv = glua.LBool(vs)
			default:
				continue
			}
			tb.RawSetString(k, lv)
		}
		l.SetGlobal(name, tb)
	}
}

func loggerFunc(logFn func(string, ...any), l *glua.LState) glua.LValue {
	return l.NewFunction(func(l *glua.LState) int {
		ln := l.GetTop()
		if ln > 0 {
			ln = ln - 1 // first one is pat
		}
		ar := make([]any, 0, ln)

		fmt := ""
		for i := range l.GetTop() {
			lv := l.Get(i + 1)
			if i == 0 {
				if lv.Type() == glua.LTString {
					fmt = glua.LVAsString(lv)
				}
			} else if lv.Type() == glua.LTTable { // not really necessary, for readability only
				tb := lv.(*glua.LTable)
				m := make(map[string]any, tb.MaxN())
				tb.ForEach(func(k, v glua.LValue) {
					ks := glua.LVAsString(k)
					m[ks] = v
				})
				ar = append(ar, m)
			} else {
				ar = append(ar, lv)
			}
		}
		logFn(fmt, ar...)
		return 0
	})
}

func luaInfof(rail Log, l *glua.LState) {
	l.SetGlobal("infof", loggerFunc(rail.Infof, l))
}

func luaErrorf(rail Log, l *glua.LState) {
	l.SetGlobal("errorf", loggerFunc(rail.Errorf, l))
}
