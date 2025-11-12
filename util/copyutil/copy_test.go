package copyutil

import (
	"testing"

	"github.com/curtisnewbie/miso/util/atom"
)

func TestCopy(t *testing.T) {
	type DummyOne struct {
		Name string
		Age  int
		Time *atom.Time
	}

	type DummyTwo struct {
		Age  int
		Time *atom.Time
	}
	d := DummyOne{
		Name: "123",
		Age:  1,
		Time: atom.NowPtr(),
	}
	v := CopyNew[DummyTwo](&d)
	t.Logf("%#v", v)
}
