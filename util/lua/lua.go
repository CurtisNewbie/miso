package lua

import (
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/curtisnewbie/miso/util/errs"
	"github.com/spf13/cast"
	glua "github.com/yuin/gopher-lua"
)

type Log interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type LuaRetTypes interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64 | ~string | ~bool
}

// Run Lua Script.
//
// Use With***() funcs to set global variables.
//
// If [WithLogger] is provided, infof(...) and errorf(...) are builtin funcs that can be called inside the lua scripts for logging.
//
// If [WithLogger] is not provided, you can still use infof(...), errorf(...) and printf(...), logs are written directly to stdout.
//
// E.g., [WithGlobalStr], [WithGlobalNum], [WithGlobalBool]
func Run[T LuaRetTypes](script string, ops ...func(*glua.LState)) (T, error) {
	st := glua.NewState()
	defer st.Close()

	// default implementation for logging
	st.SetGlobal("printf", loggerFunc(func(s string, a ...any) {
		fmt.Printf(s+"\n", a...)
	}, st))
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
		var t T
		return t, errs.WrapErr(err)
	}

	var res glua.LValue
	if st.GetTop() > 0 {
		res = st.Get(1)
	}
	return unpackLuaRetType[T](res), nil
}

func unpackLuaRetType[T LuaRetTypes](v glua.LValue) T {
	var t T
	if v == nil {
		return t
	}
	tt := reflect.TypeOf(t)
	tv := reflect.New(tt).Elem()
	switch v.Type() {
	case glua.LTBool:
		if tt.Kind() == reflect.Bool {
			tv.SetBool(glua.LVAsBool(v))
			return tv.Interface().(T)
		}
		return t
	case glua.LTNumber:
		switch tt.Kind() {
		case reflect.Float64, reflect.Float32:
			tv.SetFloat(float64(glua.LVAsNumber(v)))
			return tv.Interface().(T)
		case reflect.Int, reflect.Int32, reflect.Int64:
			tv.SetInt(int64(glua.LVAsNumber(v)))
			return tv.Interface().(T)
		}
		return t
	case glua.LTString:
		vs := glua.LVAsString(v)
		switch tt.Kind() {
		case reflect.String:
			tv.SetString(vs)
			return tv.Interface().(T)
		case reflect.Float64, reflect.Float32:
			tv.SetFloat(cast.ToFloat64(vs))
			return tv.Interface().(T)
		case reflect.Int, reflect.Int32, reflect.Int64:
			tv.SetInt(int64(cast.ToFloat64(vs)))
			return tv.Interface().(T)
		}
		return t
	}
	return t
}

// Run Lua Script via Reader.
//
// See [Run]
func RunReader[T LuaRetTypes](f io.Reader, ops ...func(*glua.LState)) (T, error) {
	byt, err := io.ReadAll(f)
	if err != nil {
		var t T
		return t, errs.WrapErr(err)
	}
	return Run[T](string(byt), ops...)
}

// Run Lua Script File.
//
// See [Run]
func RunFile[T LuaRetTypes](path string, ops ...func(*glua.LState)) (T, error) {
	byt, err := os.ReadFile(path)
	if err != nil {
		var t T
		return t, errs.WrapErr(err)
	}
	return Run[T](string(byt), ops...)
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

func WithGlobalNum(name string, v float64) func(*glua.LState) {
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

// Set table to global field.
//
// Values in map must be one of the types: string, int, int8, int16, int32, int64, float32, float64, and bool; if not, the value is ignored.
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
