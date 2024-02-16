package miso

import (
	"errors"
	"fmt"
)

const (
	ErrCodeGeneric = "XXXX"
)

type RespUnwrapper interface {
	Unwrap() Resp
}

// Web Endpoint's Resp
type Resp struct {
	ErrorCode string      `json:"errorCode" desc:"error code"`
	Msg       string      `json:"msg" desc:"message"`
	Error     bool        `json:"error" desc:"whether the request was successful"`
	Data      interface{} `json:"data" desc:"response data"`
}

// Generic version of Resp
type GnResp[T any] struct {
	ErrorCode string `json:"errorCode" desc:"error code"`
	Msg       string `json:"msg" desc:"message"`
	Error     bool   `json:"error" desc:"whether the request was successful"`
	Data      T      `json:"data" desc:"response data"`
}

func (r GnResp[T]) Unwrap() Resp {
	return Resp{
		ErrorCode: r.ErrorCode,
		Msg:       r.Msg,
		Error:     r.Error,
		Data:      r.Data,
	}
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

func OkGnResp[T any](data T) GnResp[T] {
	return GnResp[T]{
		Data:  data,
		Error: false,
	}
}

func VoidResp() GnResp[Void] {
	return GnResp[Void]{}
}

func WrapGnResp[T any](data T, err error) (GnResp[T], error) {
	if err != nil {
		return GnResp[T]{}, err
	}
	return OkGnResp(data), nil
}

// Wrap result (data and err) with a common Resp object.
//
// If err is not nil, returns a Resp body containing the error code and message.
// If err is nil, the data is wrapped inside a Resp object and returned with http.StatusOK.
func WrapResp(rail Rail, data interface{}, err error, url string) Resp {
	if err != nil {
		me := &MisoErr{}
		if errors.As(err, &me) {
			if !me.HasCode() {
				me.Code = ErrCodeGeneric
			}
			if me.InternalMsg != "" {
				rail.Infof("'%s' returned error, code: '%v', msg: '%v', internalMsg: '%v'", url, me.Code, me.Msg, me.InternalMsg)
			} else {
				rail.Infof("'%s' returned error, code: '%v', msg: '%v'", url, me.Code, me.Msg)
			}
			return ErrorRespWCode(me.Code, me.Msg)
		}

		ve := &ValidationError{}
		if errors.As(err, &ve) {
			msg := ve.Error()
			rail.Infof("'%s' returned error, request invalid, msg: '%v'", url, msg)
			return ErrorResp(msg)
		}

		// not a MisoErr, just return some generic msg
		rail.Errorf("Unknown error, %v", err)
		return ErrorResp("Unknown system error, please try again later")
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

	if v, ok := data.(Resp); ok {
		return v
	}

	if v, ok := data.(RespUnwrapper); ok {
		return v.Unwrap()
	}

	return Resp{
		Data:  data,
		Error: false,
	}
}
