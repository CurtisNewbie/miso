package miso

import (
	"errors"
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

func (r GnResp[T]) MappedRes(mapper map[string]error) (T, error) {
	if r.Error {
		if mapper != nil {
			if err, ok := mapper[r.ErrorCode]; ok && err != nil {
				return r.Data, err
			}
		}
		return r.Data, r.Err()
	}
	return r.Data, r.Err()
}

// Wrap result (data and err) with a common Resp object.
//
// If err is not nil, returns a Resp body containing the error code and message.
// If err is nil, the data is wrapped inside a Resp object and returned with http.StatusOK.
func WrapResp(data interface{}, err error, rail Rail) Resp {
	if err != nil {
		me := &MisoErr{}
		if errors.As(err, &me) {
			if !me.HasCode() {
				me.Code = ErrCodeGeneric
			}
			if me.InternalMsg != "" {
				rail.Infof("Returned error, code: '%v', msg: '%v', internalMsg: '%v'", me.Code, me.Msg, me.InternalMsg)
			} else {
				rail.Infof("Returned error, code: '%v', msg: '%v'", me.Code, me.Msg)
			}
			return ErrorRespWCode(me.Code, me.Msg)
		}

		ve := &ValidationError{}
		if errors.As(err, &ve) {
			return ErrorResp(ve.Error())
		}

		// not a MisoErr, just return some generic msg
		rail.Errorf("Unknown error, %v", err)
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
