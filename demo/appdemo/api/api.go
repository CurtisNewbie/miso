package api

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/gorm"
)

type PostReq struct {
	RequestId string
}
type PostRes struct {
	ResultId string
	Time     util.ETime
}

type ApiReq struct {
	Name   string
	Extras []ApiReqExtra
}
type ApiReqExtra struct {
	Special bool
}
type ApiRes struct{}

// misoapi-http: POST /api/v1
func api1(inb *miso.Inbound, req PostReq) (PostRes, error) {
	return PostRes{}, nil
}

// misoapi-http: POST /api/v2
func api2(inb *miso.Inbound, req *PostReq) (PostRes, error) {
	return PostRes{}, nil
}

// misoapi-http: POST /api/v3
func api3(inb *miso.Inbound, req *PostReq) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v4
func api4(inb *miso.Inbound, req ApiReq) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v5
func api5(inb *miso.Inbound, req *ApiReq) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v6
func api6(inb *miso.Inbound, req *ApiReq, db *gorm.DB) (*PostRes, error) {
	return &PostRes{}, nil
}

// misoapi-http: POST /api/v7
func api7(inb *miso.Inbound, req *ApiReq, db *gorm.DB) (ApiRes, error) {
	return ApiRes{}, nil
}

// misoapi-http: POST /api/v8
func api8(inb *miso.Inbound, req *ApiReq, db *gorm.DB) (*ApiRes, error) {
	return &ApiRes{}, nil
}

// misoapi-http: POST /api/v9
func api9(inb *miso.Inbound, req *ApiReq, db *gorm.DB) (*[]ApiRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v10
func api10(inb *miso.Inbound, req *ApiReq, db *gorm.DB) ([]ApiRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v11
func api11(inb *miso.Inbound, req *ApiReq, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v12
func api12(inb *miso.Inbound, req []ApiReq, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: POST /api/v13
func api13(inb *miso.Inbound, req []ApiReq, db *gorm.DB) (any, error) {
	return nil, nil
}

// misoapi-http: POST /api/v14
func api14(inb *miso.Inbound, req ApiReq, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: GET /api/v15
func api15(inb *miso.Inbound, db *gorm.DB) ([]PostRes, error) {
	return nil, nil
}

// misoapi-http: GET /api/v16
// misoapi-ngtable
func api16(inb *miso.Inbound, db *gorm.DB) (miso.PageRes[PostRes], error) {
	return miso.PageRes[PostRes]{}, nil
}

// misoapi-http: GET /api/v17
func api17(inb *miso.Inbound, db *gorm.DB) []PostRes {
	return []PostRes{}
}

// misoapi-http: POST /api/v18
func api18(inb *miso.Inbound, db *gorm.DB) {
}

// misoapi-http: GET /api/v19
func api19(inb *miso.Inbound, db *gorm.DB) error {
	return nil
}

// misoapi-http: POST /api/v20
func api20(inb *miso.Inbound, req ApiReq, db *gorm.DB) {
}

// misoapi-http: POST /api/v21
// misoapi-raw
func api21(inb *miso.Inbound, req ApiReq, db *gorm.DB) {
}

// misoapi-http: POST /api/v22
// misoapi-json-resp-type: PostRes
// misoapi-raw
func api22(inb *miso.Inbound, req ApiReq, db *gorm.DB) {
}

// misoapi-http: POST /api/v23
// misoapi-json-resp-type: PostRes
// misoapi-raw
func api23(inb *miso.Inbound) {
}

// misoapi-http: POST /api/v24
// misoapi-raw
func api24(inb *miso.Inbound, rail miso.Rail, db *gorm.DB) {
}

// misoapi-http: POST /api/v25
// misoapi-json-resp-type: PostRes
// misoapi-raw
func api25(inb *miso.Inbound, rail miso.Rail, db *gorm.DB) {
}
