package miso

import (
	"fmt"
)

const (
	ErrCodeGeneric = "XXXX"
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

func (r GnResp[T]) Res() (T, error) {
	return r.Data, r.Err()
}

/** Wrap with a response object */
func WrapResp(data interface{}, e error, rail Rail) Resp {
	if e != nil {
		if me, ok := e.(*MisoErr); ok {
			if !me.HasCode() {
				me.Code = ErrCodeGeneric
			}
			rail.Infof("Returned error, code: '%v', msg: '%v', internalMsg: '%v'", me.Code, me.Msg, me.InternalMsg)
			return ErrorRespWCode(me.Code, me.Msg)
		}

		if ve, ok := e.(*ValidationError); ok {
			return ErrorResp(ve.Error())
		}

		// not a MisoErr, just return some generic msg
		rail.Errorf("Unknown error, %v", e)
		return ErrorResp("Unknown system error, please try again later")
	}

	if v, ok := data.(Resp); ok {
		return v
	}

	return OkRespWData(data)
}

// Build error Resp
func ErrorResp(msg string) Resp {
	return Resp{
		ErrorCode: ErrCodeGeneric, // just some random code
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
