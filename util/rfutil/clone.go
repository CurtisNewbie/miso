package rfutil

import (
	"reflect"
)

// Clone Map, Slice or Array.
func Clone(rv reflect.Value) (any, bool) {
	if cp, ok := CloneMap(rv); ok {
		return cp, true
	}
	if cp, ok := CloneSlice(rv); ok {
		return cp, true
	}
	if cp, ok := CloneArray(rv); ok {
		return cp, true
	}
	return nil, false
}

// Clone Map.
func CloneMap(rv reflect.Value) (any, bool) {
	kd := rv.Kind()
	if kd != reflect.Map {
		return nil, false
	}
	keyType := rv.Type().Key()
	elemType := rv.Type().Elem()
	dm := reflect.MakeMap(reflect.MapOf(keyType, elemType))
	iter := rv.MapRange()
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		if vv, ok := Clone(value); ok {
			value = reflect.ValueOf(vv)
		}
		dm.SetMapIndex(key, value)
	}
	return dm.Interface(), true
}

// Clone Slice.
func CloneSlice(rv reflect.Value) (any, bool) {
	kd := rv.Kind()
	if kd != reflect.Slice {
		return nil, false
	}

	dv := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Cap())
	reflect.Copy(dv, rv)
	return dv.Interface(), true
}

// Clone Array.
func CloneArray(rv reflect.Value) (any, bool) {
	kd := rv.Kind()
	if kd != reflect.Array {
		return nil, false
	}

	dv := reflect.New(reflect.ArrayOf(rv.Len(), rv.Type().Elem())).Elem()
	reflect.Copy(dv, rv)
	return dv.Interface(), true
}
