package util

import (
	"reflect"
)

var (
	voidType = reflect.TypeOf(Void{})
)

// Empty Struct
type Void struct{}

func IsVoid(t reflect.Type) bool {
	return t == voidType
}
