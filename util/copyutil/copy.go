package copyutil

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/jinzhu/copier"
)

func Copy(fromPtr any, toPtr any) {
	if err := copier.Copy(toPtr, fromPtr); err != nil {
		util.ErrorLog("Failed to copy value, %v", miso.WrapErr(err))
	}
}

func CopyNew[V any](fromPtr any) V {
	var v V
	Copy(fromPtr, &v)
	return v
}
