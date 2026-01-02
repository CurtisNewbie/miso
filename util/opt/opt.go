package opt

import "reflect"

type Opt[T any] struct {
	v     T
	isNil bool
}

func (o *Opt[T]) IsNil() bool {
	return o.isNil
}

func (o *Opt[T]) Get() T {
	return o.v
}

func (o *Opt[T]) MayGet() (T, bool) {
	return o.v, o.isNil
}

func Nil[T any]() Opt[T] {
	return Opt[T]{
		isNil: true,
	}
}

func New[T any](v T) Opt[T] {
	isNil := false
	switch reflect.TypeOf(v).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		if reflect.ValueOf(v).IsNil() {
			isNil = true
		}
	}

	return Opt[T]{
		isNil: isNil,
		v:     v,
	}
}
