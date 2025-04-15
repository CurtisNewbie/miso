// auto generated by misoapi v0.1.20 at 2025/04/15 23:13:25 (CST), please do not modify
package api

import (
	"github.com/curtisnewbie/miso/middleware/dbquery"
	"github.com/curtisnewbie/miso/miso"
)

func init() {
	miso.IPost("/api/v1",
		func(inb *miso.Inbound, req PostReq) (PostRes, error) {
			return api1(inb, req)
		}).
		Extra(miso.ExtraName, "api1")

	miso.IPost("/api/v2",
		func(inb *miso.Inbound, req *PostReq) (PostRes, error) {
			return api2(inb, req)
		}).
		Extra(miso.ExtraName, "api2")

	miso.IPost("/api/v3",
		func(inb *miso.Inbound, req *PostReq) (*PostRes, error) {
			return api3(inb, req)
		}).
		Extra(miso.ExtraName, "api3")

	miso.IPost("/api/v4",
		func(inb *miso.Inbound, req ApiReq) (*PostRes, error) {
			return api4(inb, req)
		}).
		Extra(miso.ExtraName, "api4")

	miso.IPost("/api/v5",
		func(inb *miso.Inbound, req *ApiReq) (*PostRes, error) {
			return api5(inb, req)
		}).
		Extra(miso.ExtraName, "api5")

	miso.IPost("/api/v6",
		func(inb *miso.Inbound, req *ApiReq) (*PostRes, error) {
			return api6(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api6")

	miso.IPost("/api/v7",
		func(inb *miso.Inbound, req *ApiReq) (ApiRes, error) {
			return api7(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api7")

	miso.IPost("/api/v8",
		func(inb *miso.Inbound, req *ApiReq) (*ApiRes, error) {
			return api8(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api8")

	miso.IPost("/api/v9",
		func(inb *miso.Inbound, req *ApiReq) (*[]ApiRes, error) {
			return api9(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api9")

	miso.IPost("/api/v10",
		func(inb *miso.Inbound, req *ApiReq) ([]ApiRes, error) {
			return api10(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api10")

	miso.IPost("/api/v11",
		func(inb *miso.Inbound, req *ApiReq) ([]PostRes, error) {
			return api11(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api11")

	miso.IPost("/api/v12",
		func(inb *miso.Inbound, req []ApiReq) ([]PostRes, error) {
			return api12(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api12")

	miso.IPost("/api/v13",
		func(inb *miso.Inbound, req []ApiReq) (any, error) {
			return api13(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api13")

	miso.IPost("/api/v14",
		func(inb *miso.Inbound, req ApiReq) ([]PostRes, error) {
			return api14(inb, req, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api14")

	miso.Get("/api/v15",
		func(inb *miso.Inbound) ([]PostRes, error) {
			return api15(inb, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api15")

	miso.Get("/api/v16",
		func(inb *miso.Inbound) (miso.PageRes[PostRes], error) {
			return api16(inb, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api16").
		Extra(miso.ExtraNgTable, true)

	miso.Get("/api/v17",
		func(inb *miso.Inbound) ([]PostRes, error) {
			return api17(inb, dbquery.GetDB()), nil
		}).
		Extra(miso.ExtraName, "api17")

	miso.Post("/api/v18",
		func(inb *miso.Inbound) (any, error) {
			api18(inb, dbquery.GetDB())
			return nil, nil
		}).
		Extra(miso.ExtraName, "api18")

	miso.Get("/api/v19",
		func(inb *miso.Inbound) (any, error) {
			return nil, api19(inb, dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api19")

	miso.IPost("/api/v20",
		func(inb *miso.Inbound, req ApiReq) (any, error) {
			api20(inb, req, dbquery.GetDB())
			return nil, nil
		}).
		Extra(miso.ExtraName, "api20")

	miso.RawPost("/api/v21",
		func(inb *miso.Inbound) {
			var req ApiReq
			inb.MustBind(&req)
			api21(inb, req, dbquery.GetDB())
		}).
		DocJsonReq(ApiReq{}).
		Extra(miso.ExtraName, "api21")

	miso.RawPost("/api/v22",
		func(inb *miso.Inbound) {
			var req ApiReq
			inb.MustBind(&req)
			api22(inb, req, dbquery.GetDB())
		}).
		DocJsonReq(ApiReq{}).
		DocJsonResp(PostRes{}).
		Extra(miso.ExtraName, "api22")

	miso.RawPost("/api/v23", api23).
		DocJsonResp(PostRes{}).
		Extra(miso.ExtraName, "api23")

	miso.RawPost("/api/v24",
		func(inb *miso.Inbound) {
			api24(inb, inb.Rail(), dbquery.GetDB())
		}).
		Extra(miso.ExtraName, "api24")

	miso.RawPost("/api/v25",
		func(inb *miso.Inbound) {
			api25(inb, inb.Rail(), dbquery.GetDB())
		}).
		DocJsonResp(PostRes{}).
		Extra(miso.ExtraName, "api25")

	miso.RawOptions("/api/v26", api26).
		Extra(miso.ExtraName, "api26")

	miso.RawHead("/api/v27", api27).
		Extra(miso.ExtraName, "api27")

	miso.RawPatch("/api/v28", api28).
		Extra(miso.ExtraName, "api28")

	miso.RawConnect("/api/v29", api29).
		Extra(miso.ExtraName, "api29")

	miso.RawTrace("/api/v30", api30).
		Extra(miso.ExtraName, "api30")

}
