package util

import (
	"reflect"
	"runtime"
	"unicode"
)

type ForEachField func(index int, field reflect.StructField) (breakIteration bool)

type TagRetriever func(tagName string) string

type Introspector struct {
	Type          reflect.Type
	Fields        []reflect.StructField
	fieldIndexMap map[string]int
}

// Iterate fields
func (it *Introspector) IterFields(forEach ForEachField) {
	for i, f := range it.Fields {
		if doBreak := forEach(i, f); doBreak {
			return
		}
	}
}

// Get tag retriever for a field
func (it *Introspector) TagRetriever(fieldName string) (t TagRetriever, isFieldFound bool) {
	f, ok := it.Field(fieldName)
	if !ok {
		return nil, false
	}

	t = func(tagName string) string {
		return f.Tag.Get(tagName)
	}
	isFieldFound = true
	return
}

// Get tag by of field
func (it *Introspector) Tag(fieldName string, tagName string) (tag string, isFieldFound bool) {
	f, ok := it.Field(fieldName)
	if !ok {
		return
	}

	tag = f.Tag.Get(tagName)
	isFieldFound = true
	return
}

// Get field index
func (it *Introspector) FieldIdx(fieldName string) (index int, isFieldFound bool) {
	i, isFieldFound := it.fieldIndexMap[fieldName]
	return i, isFieldFound
}

// Get field at index
func (it *Introspector) FieldAt(idx int) (field reflect.StructField) {
	return it.Fields[idx]
}

// Get field by name
func (it *Introspector) Field(fieldName string) (field reflect.StructField, isFieldFound bool) {
	i, isFieldFound := it.fieldIndexMap[fieldName]
	if !isFieldFound {
		return
	}

	field = it.Fields[i]
	return
}

// Create new Introspector
func Introspect(ptr any) Introspector {
	t := reflect.TypeOf(ptr)
	el := t
	if t.Kind() == reflect.Pointer {
		el = t.Elem()
	}
	fields := CollectTypeFields(t)
	indexMap := map[string]int{}

	for i, v := range fields {
		indexMap[v.Name] = i
	}

	return Introspector{Type: el, Fields: fields, fieldIndexMap: indexMap}
}

// Get Fields of A Type
func CollectFields(ptr any) []reflect.StructField {
	el := reflect.TypeOf(ptr).Elem()
	return CollectTypeFields(el)
}

// Get Fields of A Type
func CollectTypeFields(eleType reflect.Type) []reflect.StructField {
	fields := []reflect.StructField{}
	for i := 0; i < eleType.NumField(); i++ {
		fields = append(fields, eleType.Field(i))
	}
	return fields
}

// Check if field is exposed
func IsFieldExposed(fieldName string) bool {
	for _, c := range fieldName {
		return unicode.IsUpper(c) // only check first unicode character
	}
	return false
}

// Get name of func
func FuncName(f any) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func NewVar[T any]() T {
	var t T
	return t
}

func TypeName(t reflect.Type) string {
	if t.Name() != "" {
		return t.Name()
	} else {
		return t.String()
	}
}

type WalkTagCallback struct {
	Tag      string
	OnWalked func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error
}

// Walk fields of *struct, won't go deeper even if the field is a struct.
func WalkTagShallow(ptr any, callbacks ...WalkTagCallback) error {
	if len(callbacks) < 1 {
		return nil
	}

	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Pointer {
		return nil
	}
	idr := reflect.Indirect(v)
	if idr.Kind() != reflect.Struct {
		return nil
	}

	t := reflect.TypeOf(ptr).Elem()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := idr.Field(i)
		if !fv.CanSet() {
			continue
		}
		for _, cb := range callbacks {
			tagVal := f.Tag.Get(cb.Tag)
			if tagVal == "" {
				continue
			}
			if err := cb.OnWalked(tagVal, fv, f); err != nil {
				return err
			}
		}
	}
	return nil
}

// Construct Struct/Interface/Pointer values to map[string]any.
//
// This method doesn't convert recursively.
func ReflectGenMap(t any) map[string]any {
	v := reflect.ValueOf(t)
	rt := v.Type()

	switch v.Kind() {
	case reflect.Struct:
		m := map[string]any{}
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if f.IsExported() {
				m[f.Name] = v.Field(i).Interface()
			}
		}
		return m
	case reflect.Interface, reflect.Pointer:
		ele := v.Elem()
		return ReflectGenMap(ele.Interface())
	default:
		return map[string]any{}
	}
}

func ReflectFuncName(fun any) string {
	var funcName string = "nil"
	if fun != nil {
		funcName = runtime.FuncForPC(reflect.ValueOf(fun).Pointer()).Name()
	}
	return funcName
}

func IsBasicKind(k reflect.Kind) bool {
	yes := false
	switch k {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Complex64,
		reflect.Complex128:
		yes = true
	}
	return yes
}

func ReflectBasicValue(rv reflect.Value) (any, bool) {
	ftk := rv.Kind()
	if IsBasicKind(ftk) {
		return rv.Interface(), true
	}
	if ftk == reflect.Pointer {
		if rv.IsNil() {
			return nil, true
		}

		rve := rv.Elem()
		if IsBasicKind(rve.Kind()) {
			return rve.Interface(), true
		}
	}
	return nil, false
}
