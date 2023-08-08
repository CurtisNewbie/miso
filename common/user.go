package common

import (
	"strconv"
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
