package server

import (
	"net/http"
	"strings"

	. "github.com/curtisnewbie/gocommon/common"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Route handle
type RouteHandler func(c *gin.Context) (any, error)

// Authenticated route handle
type AuthRouteHandler func(c *gin.Context, user *User) (any, error)

// Build a Route Handler for an authorized request
func BuildAuthRouteHandler(handler AuthRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		user := RequireUser(c)
		r, e := handler(c, user)
		HandleResult(c, r, e)
	}
}

// Build a Route Handler
func BuildRouteHandler(handler RouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		r, e := handler(c)
		HandleResult(c, r, e)
	}
}

// Build a Route Handler for authorized and JSON-based request
func BuildAuthJRouteHandler[T any](handler func(c *gin.Context, user *User, t *T) (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		user := RequireUser(c)
		var t T
		MustBindJson(c, &t)
		r, e := handler(c, user, &t)
		HandleResult(c, r, e)
	}
}

// Build a Route Handler for JSON-based request
func BuildJRouteHandler[T any](handler func(c *gin.Context, t *T) (any, error)) func(c *gin.Context) {
	return func(c *gin.Context) {
		var t T
		MustBindJson(c, &t)
		r, e := handler(c, &t)
		HandleResult(c, r, e)
	}
}

// Handle route's result
func HandleResult(c *gin.Context, r any, e error) {
	if e != nil {
		DispatchErrJson(c, e)
		return
	}

	if r != nil {
		DispatchOkWData(c, r)
		return
	}
	DispatchOk(c)
}

// Must bind json content to the given pointer, else panic
func MustBindJson(c *gin.Context, ptr any) {
	if err := c.ShouldBindJSON(ptr); err != nil {
		logrus.Errorf("Bind Json failed, %v", err)
		panic("Illegal Arguments")
	}
}

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, body)
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, err error) {
	c.JSON(http.StatusOK, WrapResp(nil, err))
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, ErrorResp(msg))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	c.JSON(http.StatusOK, OkResp())
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, OkRespWData(data))
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
