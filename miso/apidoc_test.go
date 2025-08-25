package miso

import (
	"reflect"
	"testing"
)

func TestBuildJsonPayloadDesc(t *testing.T) {
	d := BuildJsonPayloadDesc(reflect.ValueOf(Resp{Data: true}))
	t.Logf("%#v", d)

	type body struct {
		Names  []string
		Params map[string]string
	}
	d = BuildJsonPayloadDesc(reflect.ValueOf(Resp{Data: body{}}))
	t.Logf("%#v", d)
	for _, f := range d.Fields {
		t.Logf("%v -> %v", f.GoFieldName, f.pureGoTypeName())
		for _, ff := range f.Fields {
			t.Logf("\t%v -> %v,%v", ff.GoFieldName, ff.pureGoTypeName(), ff.TypeNameAlias)
		}
	}
}
