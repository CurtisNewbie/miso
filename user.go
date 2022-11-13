package gocommon

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
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
func RequireRole(user *User, role Role) {
	if !IsRole(user, role) {
		panic(fmt.Sprintf("Role %s is required", role))
	}
}

// Extract user from request headers, panic if failed
func RequireUser(c *gin.Context) *User {
	u, e := ExtractUser(c)
	if e != nil {
		panic(e)
	}
	return u
}

/* Extract User from request headers */
func ExtractUser(c *gin.Context) (*User, error) {
	id := c.GetHeader("id")
	if id == "" {
		return nil, NewWebErr("Please sign up first")
	}

	var services []string
	servicesStr := c.GetHeader("services")
	if servicesStr == "" {
		services = make([]string, 0)
	} else {
		services = strings.Split(servicesStr, ",")
	}

	return &User{
		UserId:   id,
		Username: c.GetHeader("username"),
		UserNo:   c.GetHeader("userno"),
		Role:     c.GetHeader("role"),
		Services: services,
	}, nil
}

// Check if the user is a guest
func IsGuest(user *User) bool {
	return IsRole(user, GUEST)
}

// Check if the user is an admin
func IsAdmin(user *User) bool {
	return IsRole(user, ADMIN)
}

// Check if the user is the specified role, if the user doesn't have a role at all it will panic
func IsRole(user *User, role Role) bool {
	if user == nil {
		panic("user == nil")
	}

	return user.Role == string(role)
}
