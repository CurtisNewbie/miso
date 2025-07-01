package copyutil

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/jinzhu/copier"
)

func Copy(from any, toPtr any) {
	if err := copier.Copy(toPtr, from); err != nil {
		util.ErrorLog("Failed to copy value, %v", miso.WrapErr(err))
	}
}

func CopyNew[V any](from any) V {
	var v V
	Copy(from, &v)
	return v
}
