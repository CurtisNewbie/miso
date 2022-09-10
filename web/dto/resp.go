package dto

import (
	"github.com/curtisnewbie/gocommon/weberr"

	log "github.com/sirupsen/logrus"
)

// Web Endpoint's Resp
type Resp struct {
	ErrorCode string      `json:"errorCode"`
	Msg       string      `json:"msg"`
	Error     bool        `json:"error"`
	Data      interface{} `json:"data"`
}

/** Wrap with a response object */
func WrapResp(data interface{}, e error) *Resp {
	if e != nil {
		if we, ok := e.(*weberr.WebError); ok {
			if weberr.HasCode(we) {
				return ErrorRespWCode(we.Code, we.Msg)
			} else {
				return ErrorResp(we.Msg)
			}
		}

		// not a WebError, just return some generic msg
		log.Errorf("Unknown error, %v", e)
		return ErrorResp("Unknown system error, please try again later")
	}

	return OkRespWData(data)
}

// Build error Resp
func ErrorResp(msg string) *Resp {
	return &Resp{
		ErrorCode: "XXXX", // just some random code
		Msg:       msg,
		Error:     true,
	}
}

// Build error Resp
func ErrorRespWCode(code string, msg string) *Resp {
	return &Resp{
		ErrorCode: code,
		Msg:       msg,
		Error:     true,
	}
}

// Build OK Resp
func OkResp() *Resp {
	return &Resp{
		Error: false,
	}
}

// Build OK Resp with data
func OkRespWData(data interface{}) *Resp {
	if data == nil {
		return OkResp()
	}

	return &Resp{
		Data:  data,
		Error: false,
	}
}
