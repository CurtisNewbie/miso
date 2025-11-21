package rfutil

import (
	"reflect"
	"testing"

	"github.com/spf13/cast"
)

func TestCloneArray(t *testing.T) {
	a := [10]string{}
	for i := range 10 {
		a[i] = cast.ToString(i)
	}
	t.Logf("origin: %#v", a)
	cp, ok := CloneArray(reflect.ValueOf(a))
	if !ok {
		t.Fatal("not ok")
	}
	a[0] = "-1"
	t.Logf("modified: %#v", a)
	t.Logf("cp: %#v", cp)
}

func TestCloneSlice(t *testing.T) {
	a := []string{}
	for i := range 10 {
		a = append(a, cast.ToString(i))
	}
	t.Logf("origin: %#v", a)
	cp, ok := CloneSlice(reflect.ValueOf(a))
	if !ok {
		t.Fatal("not ok")
	}
	t.Logf("origin: %#v", a)
	a[0] = "-1"
	t.Logf("modified: %#v", a)
	t.Logf("cp: %#v", cp)
}

func TestCloneMap(t *testing.T) {
	a := map[string]string{
		"1": "v1",
		"2": "v2",
		"3": "v3",
	}

	t.Logf("origin: %#v", a)
	cp, ok := CloneMap(reflect.ValueOf(a))
	if !ok {
		t.Fatal("not ok")
	}
	a["1"] = "+1"
	t.Logf("modified: %#v", a)
	t.Logf("cp: %#v", cp)
}

func TestCloneNil(t *testing.T) {
	var a map[string]string = nil
	cpa, ok := CloneMap(reflect.ValueOf(a))
	if !ok {
		t.Fatal("not ok")
	}
	t.Logf("a: %#v", cpa)

	var b []string = nil
	cpb, ok := CloneSlice(reflect.ValueOf(b))
	if !ok {
		t.Fatal("not ok")
	}
	t.Logf("b: %#v", cpb)
}
