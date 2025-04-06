package dbquery

import (
	"sync"

	"github.com/curtisnewbie/miso/miso"
	"gorm.io/gorm"
)

var getPrimaryDbOnce sync.Once
var getPrimaryDb func() *gorm.DB = func() *gorm.DB {
	miso.Error("GetPrimaryDBFunc not implemented, returning nil")
	return nil
}

func GetDB() *gorm.DB {
	return getPrimaryDb()
}

func ImplGetPrimaryDBFunc(impl func() *gorm.DB) (implSet bool) {
	var set bool = false
	getPrimaryDbOnce.Do(func() {
		getPrimaryDb = impl
		set = true
	})
	return set
}
