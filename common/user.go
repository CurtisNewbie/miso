package common

import (
	"strconv"
)

var (
	nilUser = User{IsNil: true}
)

type User struct {
	UserId   string
	UserNo   string
	Username string
	RoleNo   string
	IsNil    bool
}

func (u User) UserIdInt() int {
	if u.UserId == "" {
		return 0
	}

	v, _ := strconv.Atoi(u.UserId)
	return v
}

func NilUser() User {
	return nilUser
}