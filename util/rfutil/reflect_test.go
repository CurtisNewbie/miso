package rfutil

import (
	"reflect"
	"testing"
)

type ReflectDummy struct {
	Name string `json:"name" validation:"notEmpty"`
}

func TestIntrospector(t *testing.T) {
	var dummy ReflectDummy = ReflectDummy{}
	it := Introspect(dummy)

	if len(it.Fields) < 1 {
		t.FailNow()
	}

	f, ok := it.Field("Name")
	if !ok {
		t.FailNow()
	}

	_, ok = it.Field("someField")
	if ok {
		t.FailNow()
	}

	if f.Name != "Name" {
		t.FailNow()
	}

	tag, ok := it.Tag("Name", "json")
	if !ok {
		t.FailNow()
	}

	if tag != "name" {
		t.Fatalf("tag: %v", tag)
	}

	tagRetriever, ok := it.TagRetriever("Name")
	if !ok {
		t.FailNow()
	}

	if n := tagRetriever("json"); n != "name" {
		t.Fatalf("%v", n)
	}

	if n := tagRetriever("validation"); n != "notEmpty" {
		t.Fatalf("%v", n)
	}
}

func TestCollectFields(t *testing.T) {
	var dummy *ReflectDummy
	fields := CollectFields(dummy)
	if len(fields) < 1 {
		t.FailNow()
	}

	foundName := false
	for _, v := range fields {
		if v.Name == "Name" {
			foundName = true
		}
	}
	if !foundName {
		t.Error("Name field not found")
	}

	t.Log(fields)
}

func TestWalkTagShallow(t *testing.T) {
	type dummy struct {
		Name string `alias:"dummyName"`
		Desc string `alias:"dummyDesc"`
	}
	d := dummy{Name: "name 1", Desc: "desc 2"}
	err := WalkTagShallow(&d, WalkTagCallback{
		Tag: "alias",
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			fieldVal.SetString("yo")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", d)
}

func TestWalkTagShallowFalseCase(t *testing.T) {
	d := 0
	err := WalkTagShallow(&d, WalkTagCallback{
		Tag: "alias",
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			fieldVal.SetString("yo")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(d)
}

func BenchmarkWalkTagShallow(b *testing.B) {
	type dummy struct {
		Name string `alias:"dummyName"`
		Desc string `alias:"dummyDesc"`
	}
	d := dummy{Name: "name 1", Desc: "desc 2"}
	callback := WalkTagCallback{
		Tag: "alias",
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			fieldVal.SetString("yo")
			return nil
		},
	}
	b.ResetTimer()

	var err error
	for i := 0; i < b.N; i++ {
		err = WalkTagShallow(&d, callback)
	}
	if err != nil {
		b.Fatal(err)
	}
}
