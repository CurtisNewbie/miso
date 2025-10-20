package copyutil

import (
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/utillog"
	"github.com/jinzhu/copier"
)

func Copy(from any, toPtr any) {
	if err := copier.Copy(toPtr, from); err != nil {
		utillog.ErrorLog("Failed to copy value, %v", errs.Wrap(err))
	}
}

func CopyNew[V any](from any) V {
	var v V
	Copy(from, &v)
	return v
}
