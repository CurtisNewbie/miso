package util

import (
	"github.com/curtisnewbie/gocommon/config"
	"gorm.io/gorm"
)

type PropagatedContext struct {
	Tx    *gorm.DB
	Param map[string]interface{}
}

// Build a PropagatedContext
func BuildPContext() *PropagatedContext {
	return &PropagatedContext{Tx: config.GetDB(), Param: make(map[string]interface{})}
}
