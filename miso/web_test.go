package miso

import (
	"testing"

	"github.com/curtisnewbie/miso/util/rfutil"
)

func BenchmarkSetHeaderTag(b *testing.B) {
	type dummy struct {
		Name string  `header:"name"`
		Desc *string `header:"desc"`
		Age  int     `header:"age"`
	}
	GetHeader := func(k string) string {
		switch k {
		case "name":
			return "myname"
		case "desc":
			return "this is a test"
		case "age":
			return "???"
		}
		return ""
	}

	d := dummy{}
	var err error
	callback := walkHeaderTagCallback(GetHeader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = rfutil.WalkTagShallow(&d, callback)
	}

	if err != nil {
		b.Fatal(err)
	}
}
