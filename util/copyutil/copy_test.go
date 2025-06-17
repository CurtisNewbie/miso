package copyutil

import (
	"testing"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

func TestCopy(t *testing.T) {
	type DummyOne struct {
		Name string
		Age  int
		Time *util.ETime
	}

	type DummyTwo struct {
		Age  int
		Time *util.ETime
	}
	d := DummyOne{
		Name: "123",
		Age:  1,
		Time: util.NowPtr(),
	}
	v := CopyNew[DummyTwo](miso.EmptyRail(), &d)
	t.Logf("%#v", v)
}
