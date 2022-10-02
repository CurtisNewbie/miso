package util

import (
	"net/http"

	"github.com/curtisnewbie/gocommon/web/dto"

	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

// Route handle
type RouteHandler func(c *gin.Context) (any, error)

// Authenticated route handle
type AuthRouteHandler func(c *gin.Context, user *User) (any, error)

func BuildGRouteHandler[T any](c *gin.Context) (*T, error) {
	var t T
	MustBindJson(c, &t)
	return &t, nil
}

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
		log.Errorf("Bind Json failed, %v", err)
		panic("Illegal Arguments")
	}
}

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, body)
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, err error) {
	c.JSON(http.StatusOK, dto.WrapResp(nil, err))
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, dto.ErrorResp(msg))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	c.JSON(http.StatusOK, dto.OkResp())
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.OkRespWData(data))
}
