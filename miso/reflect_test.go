package miso

import (
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
