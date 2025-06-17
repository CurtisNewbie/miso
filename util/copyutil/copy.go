package copyutil

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/jinzhu/copier"
)

func Copy(rail miso.Rail, fromPtr any, toPtr any) {
	if err := copier.Copy(toPtr, fromPtr); err != nil {
		rail.Errorf("Failed to copy value, %v", miso.WrapErr(err))
	}
}

func CopyNew[V any](rail miso.Rail, fromPtr any) V {
	var v V
	Copy(rail, fromPtr, &v)
	return v
}
