package common

import (
	"github.com/curtisnewbie/miso/flow"
)

type User = flow.User

const (
	UsernameTraceKey = flow.XUsername
	UserNoTraceKey   = flow.XUserNo
	RoleNoTraceKey   = flow.XRoleNo
)

var (
	NilUser   = flow.NilUser
	GetUser   = flow.GetUser
	StoreUser = flow.StoreUser
)
