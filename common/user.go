package common

import (
	"fmt"
)

type Role string

const (
	ADMIN Role = "admin"
	GUEST Role = "guest"
	USER  Role = "user"
)

type User struct {
	UserId   string
	UserNo   string
	Username string
	Role     string
	Services []string
}

// Check if role matches, else panic
func RequireRole(user User, role Role) {
	if !IsRole(user, role) {
		panic(fmt.Sprintf("Role %s is required", role))
	}
}

// Check if the user is a guest
func IsGuest(user User) bool {
	return IsRole(user, GUEST)
}

// Check if the user is an admin
func IsAdmin(user User) bool {
	return IsRole(user, ADMIN)
}

// Check if the user is the specified role, if the user doesn't have a role at all it will panic
func IsRole(user User, role Role) bool {
	return user.Role == string(role)
}
