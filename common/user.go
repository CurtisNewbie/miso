package common

var (
	nilUser = User{IsNil: true}
)

type User struct {
	UserId   int
	UserNo   string
	Username string
	RoleNo   string
	IsNil    bool
}

// Get a 'nil' User
func NilUser() User {
	return nilUser
}
