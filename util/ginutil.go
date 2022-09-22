package util

import (
	"net/http"

	"github.com/curtisnewbie/gocommon/web/dto"

	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"
)

type RouteHandler func(c *gin.Context) any
type AuthRouteHandler func(c *gin.Context, user *User) any

// Build a Route Handler for an authorized request
func BuildAuthRouteHandler(handler AuthRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		user := RequireUser(c)
		r := handler(c, user)
		if r != nil {
			DispatchOkWData(c, r)
			return
		}
		DispatchOk(c)
	}
}

// Build a Route Handler
func BuildRouteHandler(handler RouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		r := handler(c)
		if r != nil {
			DispatchOkWData(c, r)
			return
		}
		DispatchOk(c)
	}
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
