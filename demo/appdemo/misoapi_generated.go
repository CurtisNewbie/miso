// auto generated by misoapi v0.1.10 at 2024/10/08 18:18:20, please do not modify
package main

import (
	"github.com/curtisnewbie/miso/demo/api"
	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/miso"
)

func init() {
	miso.IPost("/api/v1",
		func(inb *miso.Inbound, req PostReq) (PostRes, error) {
			return api1(inb, req)
		})
	miso.IPost("/api/v2",
		func(inb *miso.Inbound, req *PostReq) (PostRes, error) {
			return api2(inb, req)
		})
	miso.IPost("/api/v3",
		func(inb *miso.Inbound, req *PostReq) (*PostRes, error) {
			return api3(inb, req)
		})
	miso.IPost("/api/v4",
		func(inb *miso.Inbound, req api.ApiReq) (*PostRes, error) {
			return api4(inb, req)
		})
	miso.IPost("/api/v5",
		func(inb *miso.Inbound, req *api.ApiReq) (*PostRes, error) {
			return api5(inb, req)
		})
	miso.IPost("/api/v6",
		func(inb *miso.Inbound, req *api.ApiReq) (*PostRes, error) {
			return api6(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v7",
		func(inb *miso.Inbound, req *api.ApiReq) (api.ApiRes, error) {
			return api7(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v8",
		func(inb *miso.Inbound, req *api.ApiReq) (*api.ApiRes, error) {
			return api8(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v9",
		func(inb *miso.Inbound, req *api.ApiReq) (*[]api.ApiRes, error) {
			return api9(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v10",
		func(inb *miso.Inbound, req *api.ApiReq) ([]api.ApiRes, error) {
			return api10(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v11",
		func(inb *miso.Inbound, req *api.ApiReq) ([]PostRes, error) {
			return api11(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v12",
		func(inb *miso.Inbound, req []api.ApiReq) ([]PostRes, error) {
			return api12(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v13",
		func(inb *miso.Inbound, req []api.ApiReq) (any, error) {
			return api13(inb, req, mysql.GetMySQL())
		})
	miso.IPost("/api/v14",
		func(inb *miso.Inbound, req api.ApiReq) ([]PostRes, error) {
			return api14(inb, req, mysql.GetMySQL())
		})
	miso.Get("/api/v15",
		func(inb *miso.Inbound) ([]PostRes, error) {
			return api15(inb, mysql.GetMySQL())
		})
	miso.Get("/api/v16",
		func(inb *miso.Inbound) (miso.PageRes[PostRes], error) {
			return api16(inb, mysql.GetMySQL())
		}).
		Extra(miso.ExtraNgTable, true)

}
