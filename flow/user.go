package flow

var (
	nilUser = User{IsNil: true}
)

type User struct {
	UserNo   string `json:"userNo"`
	Username string `json:"username"`
	RoleNo   string `json:"roleNo"`
	IsNil    bool   `json:"-"`
}

func (u User) IsZero() bool {
	return u.IsNil
}

// Get a 'nil' User.
func NilUser() User {
	return nilUser
}

// Get User from Rail (trace).
func GetUser(rail Rail) User {
	userNo := rail.CtxValStr(XUserNo)
	if userNo == "" {
		return NilUser()
	}

	return User{
		Username: rail.Username(),
		UserNo:   userNo,
		RoleNo:   rail.CtxValStr(XRoleNo),
		IsNil:    false,
	}
}

// Store User in Rail (trace).
func StoreUser(rail Rail, u User) Rail {
	rail = rail.
		WithCtxVal(XUsername, u.Username).
		WithCtxVal(XUserNo, u.UserNo).
		WithCtxVal(XRoleNo, u.RoleNo)
	return rail
}
