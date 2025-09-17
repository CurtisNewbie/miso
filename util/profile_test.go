package util

import (
	"reflect"
	"testing"

	"github.com/curtisnewbie/miso/util/rfutil"
)

func TestCpuProfileFunc(t *testing.T) {
	type dummy struct {
		Name string `alias:"dummyName"`
		Desc string `alias:"dummyDesc"`
	}
	d := dummy{Name: "name 1", Desc: "desc 2"}
	callback := rfutil.WalkTagCallback{
		Tag: "alias",
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			fieldVal.SetString("yo")
			return nil
		},
	}
	var err error
	if err := CpuProfileFunc("out.prof", func() {
		for i := 0; i < 9999999; i++ { // if func runs too fast, the profile will be empty
			err = rfutil.WalkTagShallow(&d, callback)
			if err != nil {
				t.Fatal(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMemProfileFunc(t *testing.T) {
	type dummy struct {
		Name string `alias:"dummyName"`
		Desc string `alias:"dummyDesc"`
	}
	d := dummy{Name: "name 1", Desc: "desc 2"}
	callback := rfutil.WalkTagCallback{
		Tag: "alias",
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			fieldVal.SetString("yo")
			return nil
		},
	}
	var err error
	if err := MemoryProfileFunc("out.prof", func() {
		for i := 0; i < 99999999; i++ {
			err = rfutil.WalkTagShallow(&d, callback)
			if err != nil {
				t.Fatal(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
}
