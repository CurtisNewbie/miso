package server

import (
	"net/http"
	"strings"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/gin-gonic/gin"
)

// Router handler with context, user (optional, may be nil), and logger prepared
type TRouteHandler func(c *gin.Context, ec common.ExecContext) (any, error)

// Router handler with the required json object, context, user (optional, may be nil), and logger prepared
type JTRouteHandler[T any] func(c *gin.Context, ec common.ExecContext, t T) (any, error)

// Build JTRouteHandler with the required json object, context, user (optional, may be nil), and logger prepared
func NewJTRouteHandler[T any](handler JTRouteHandler[T]) func(c *gin.Context) {
	return func(c *gin.Context) {
		user, _ := ExtractUser(c) // optional
		ctx := c.Request.Context()

		// json binding
		var t T
		MustBindJson(c, &t)

		// json validation
		if e := common.Validate(t); e != nil {
			HandleResult(c, nil, e)
			return
		}

		// actual handling
		r, e := handler(c, common.NewExecContext(ctx, user), t)

		// wrap result and error
		HandleResult(c, r, e)
	}
}

// Build TRouteHandler with context, user (optional, may be nil), and logger prepared
func NewTRouteHandler(handler TRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		user, _ := ExtractUser(c) // optional
		ctx := c.Request.Context()
		r, e := handler(c, common.NewExecContext(ctx, user))
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
		common.TraceLogger(c.Request.Context()).Errorf("Bind Json failed, %v", err)
		panic("Illegal Arguments")
	}
}

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, body)
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, err error) {
	c.JSON(http.StatusOK, common.WrapResp(nil, err))
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, common.ErrorResp(msg))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	c.JSON(http.StatusOK, common.OkResp())
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, common.OkRespWData(data))
}

// Extract user from request headers, panic if failed
func RequireUser(c *gin.Context) *common.User {
	u, e := ExtractUser(c)
	if e != nil {
		panic(e)
	}
	return u
}

// Extract role from request header
//
// return:
// 	role, isOk
func Role(c *gin.Context) (string, bool) {
	id := c.GetHeader("role")
	if id == "" {
		return "", false
	}
	return id, true
}

// Extract userNo from request header
//
// return:
// 	userNo, isOk
func UserNo(c *gin.Context) (string, bool) {
	id := c.GetHeader("userno")
	if id == "" {
		return "", false
	}
	return id, true
}

// Extract user id from request header
//
// return:
// 	userId, isOk
func UserId(c *gin.Context) (string, bool) {
	id := c.GetHeader("id")
	if id == "" {
		return "", false
	}
	return id, true
}

/* Extract common.User from request headers */
func ExtractUser(c *gin.Context) (*common.User, error) {
	id := c.GetHeader("id")
	if id == "" {
		return nil, common.NewWebErr("Please sign up first")
	}

	var services []string
	servicesStr := c.GetHeader("services")
	if servicesStr == "" {
		services = make([]string, 0)
	} else {
		services = strings.Split(servicesStr, ",")
	}

	return &common.User{
		UserId:   id,
		Username: c.GetHeader("username"),
		UserNo:   c.GetHeader("userno"),
		Role:     c.GetHeader("role"),
		Services: services,
	}, nil
}

// Check whether current request is authenticated
func IsRequestAuthenticated(c *gin.Context) bool {
	id := c.GetHeader("id")
	return id != ""
}
