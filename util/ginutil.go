package util

import (
	"net/http"

	"github.com/curtisnewbie/gocommon/web/dto"

	"github.com/gin-gonic/gin"
)

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, body)
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, err error) {
	c.JSON(http.StatusOK, dto.WrapResp(nil, err))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	c.JSON(http.StatusOK, dto.OkResp())
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.OkRespWData(data))
}
