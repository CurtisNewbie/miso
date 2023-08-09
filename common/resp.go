package common

import (
	"fmt"
)

// Web Endpoint's Resp
type Resp struct {
	ErrorCode string      `json:"errorCode"`
	Msg       string      `json:"msg"`
	Error     bool        `json:"error"`
	Data      interface{} `json:"data"`
}

// Generic version of Resp
type GnResp[T any] struct {
	ErrorCode string `json:"errorCode"`
	Msg       string `json:"msg"`
	Error     bool   `json:"error"`
	Data      T      `json:"data"`
}

func (r GnResp[T]) Err() error {
	if r.Error {
		return fmt.Errorf("Resp has error, code: %v, msg: %v", r.ErrorCode, r.Msg)
	}
	return nil
}

/** Wrap with a response object */
func WrapResp(data interface{}, e error, rail Rail) Resp {
	if e != nil {
		if we, ok := e.(*WebError); ok {
			rail.Infof("Returned error, code: '%v', msg: '%v', internalMsg: '%v'", we.Code, we.Msg, we.InternalMsg)
			if HasCode(we) {
				return ErrorRespWCode(we.Code, we.Msg)
			} else {
				return ErrorResp(we.Msg)
			}
		}

		if ve, ok := e.(*ValidationError); ok {
			return ErrorResp(ve.Error())
		}

		// not a WebError, just return some generic msg
		rail.Errorf("Unknown error, %v", e)
		return ErrorResp("Unknown system error, please try again later")
	}

	return OkRespWData(data)
}

// Build error Resp
func ErrorResp(msg string) Resp {
	return Resp{
		ErrorCode: "XXXX", // just some random code
		Msg:       msg,
		Error:     true,
	}
}

// Build error Resp
func ErrorRespWCode(code string, msg string) Resp {
	return Resp{
		ErrorCode: code,
		Msg:       msg,
		Error:     true,
	}
}

// Build OK Resp
func OkResp() Resp {
	return Resp{
		Error: false,
	}
}

// Build OK Resp with data
func OkRespWData(data interface{}) Resp {
	if data == nil {
		return OkResp()
	}

	return Resp{
		Data:  data,
		Error: false,
	}
}
