package util

import (
	"reflect"

	"github.com/curtisnewbie/miso/util/cli"
)

var (
	voidType = reflect.TypeOf(Void{})

	Printlnf = cli.Printlnf
)

// Empty Struct
type Void struct{}

func IsVoid(t reflect.Type) bool {
	return t == voidType
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustGet[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}
	return v
}
