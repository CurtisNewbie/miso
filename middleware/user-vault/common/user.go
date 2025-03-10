package common

import (
	"github.com/curtisnewbie/miso/miso"
)

type User struct {
	UserNo   string
	Username string
	RoleNo   string
	IsNil    bool `json:"-"`
}

const (
	UserNoTraceKey   = "x-userno"
	UsernameTraceKey = "x-username"
	RoleNoTraceKey   = "x-roleno"
)

var (
	nilUser = User{IsNil: true}
)

// load builtin propagation keys, so all dependents get the same behaviour
func LoadBuiltinPropagationKeys() {
	miso.AddPropagationKeys(UserNoTraceKey, UsernameTraceKey, RoleNoTraceKey)
}

// Get a 'nil' User.
func NilUser() User {
	return nilUser
}

// Get User from Rail (trace).
func GetUser(rail miso.Rail) User {
	userNo := rail.CtxValStr(UserNoTraceKey)
	if userNo == "" {
		return NilUser()
	}

	return User{
		Username: rail.CtxValStr(UsernameTraceKey),
		UserNo:   userNo,
		RoleNo:   rail.CtxValStr(RoleNoTraceKey),
		IsNil:    false,
	}
}

// Store User in Rail (trace).
func StoreUser(rail miso.Rail, u User) miso.Rail {
	rail = rail.
		WithCtxVal(UsernameTraceKey, u.Username).
		WithCtxVal(UserNoTraceKey, u.UserNo).
		WithCtxVal(RoleNoTraceKey, u.RoleNo)
	return rail
}
