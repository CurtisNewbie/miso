package goauth

import (
	"context"
	"errors"
	"strings"

	"github.com/curtisnewbie/gocommon/client"
	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/sirupsen/logrus"
)

const (
	// Extra Key (Left in common.StrPair) used when registering HTTP routes using methods like server.GET
	EXTRA_PATH_DOC = "PATH_DOC"

	// Property Key for enabling GoAuth Client, by default it's true
	//
	// goauth-client-go doesn't use it internally, it's only useful for the Callers
	PROP_ENABLE_GOAUTH_CLIENT = "goauth.client.enabled"
)

func init() {
	common.SetDefProp(PROP_ENABLE_GOAUTH_CLIENT, true)
}

type PathType string

type PathDoc struct {
	Desc string
	Type PathType
	Code string
}

const (
	PT_PROTECTED PathType = "PROTECTED"
	PT_PUBLIC    PathType = "PUBLIC"
)

type RoleInfoReq struct {
	RoleNo string `json:"roleNo" `
}

type RoleInfoResp struct {
	RoleNo string `json:"roleNo"`
	Name   string `json:"name"`
}

type CreatePathReq struct {
	Type    PathType `json:"type"`
	Url     string   `json:"url"`
	Group   string   `json:"group"`
	Desc    string   `json:"desc"`
	ResCode string   `json:"resCode"`
	Method  string   `json:"method"`
}

type TestResAccessReq struct {
	RoleNo string `json:"roleNo"`
	Url    string `json:"url"`
}

type TestResAccessResp struct {
	Valid bool `json:"valid"`
}

type AddResourceReq struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Test whether this role has access to the url
func TestResourceAccess(ctx context.Context, req TestResAccessReq) (*TestResAccessResp, error) {
	c := common.EmptyExecContext()
	tr := client.NewDynTClient(c, "/remote/path/resource/access-test", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return nil, tr.Err
	}

	r, e := client.ReadGnResp[*TestResAccessResp](tr)
	if e != nil {
		return nil, e
	}

	if r.Error {
		return nil, r.Err()
	}

	if r.Data == nil {
		return nil, errors.New("data is nil, unable to retrieve TestResAccessResp")
	}

	return r.Data, nil
}

// Create resource
func AddResource(ctx context.Context, req AddResourceReq) error {
	c := common.EmptyExecContext()
	tr := client.NewDynTClient(c, "/remote/resource/add", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return tr.Err
	}

	r, e := client.ReadGnResp[any](tr)
	if e != nil {
		return e
	}

	if r.Error {
		return r.Err()
	}

	logrus.Infof("Reported resource, Name: %s, Code: %s", req.Name, req.Code)
	return nil
}

// Report path
func AddPath(ctx context.Context, req CreatePathReq) error {
	c := common.EmptyExecContext()
	tr := client.NewDynTClient(c, "/remote/path/add", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return tr.Err
	}

	r, e := client.ReadGnResp[any](tr)
	if e != nil {
		return e
	}

	if r.Error {
		return r.Err()
	}

	return nil
}

// Retrieve role information
func GetRoleInfo(ctx context.Context, req RoleInfoReq) (*RoleInfoResp, error) {
	c := common.EmptyExecContext()
	tr := client.NewDynTClient(c, "/remote/role/info", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return nil, tr.Err
	}

	r, e := client.ReadGnResp[*RoleInfoResp](tr)
	if e != nil {
		return nil, e
	}

	if r.Error {
		return nil, r.Err()
	}

	if r.Data == nil {
		return nil, errors.New("data is nil, unable to retrieve RoleInfoResp")
	}

	return r.Data, nil
}

func IsEnabled() bool {
	return common.GetPropBool(PROP_ENABLE_GOAUTH_CLIENT)
}

func PathDocExtra(doc PathDoc) common.StrPair {
	return common.StrPair{Left: EXTRA_PATH_DOC, Right: doc}
}

// Register a hook to report paths to GoAuth on server bootstrapped
//
// When using methods like server.Get(...), the extra field should contains a
// common.StrPair where the key is EXTRA_PATH_DOC, so that the PathDoc can be picked
// and reported to GoAuth
//
// For example:
//
//	server.Get(url, handler, gclient.PathDocExtra(pathDoc))
func ReportPathsOnBootstrapped() {
	server.PostServerBootstrapped(func(c common.ExecContext) error {
		app := common.GetPropStr(common.PROP_APP_NAME)
		routes := server.GetHttpRoutes()

		for _, r := range routes {

			v, ok := r.Extra[EXTRA_PATH_DOC]
			if !ok {
				continue
			}

			doc, ok := v.(PathDoc)
			if !ok {
				continue
			}

			url := r.Url
			method := r.Method

			if !strings.HasPrefix(url, "/") {
				url = "/" + url
			}

			if doc.Type == "" {
				doc.Type = PT_PROTECTED
			}

			r := CreatePathReq{
				Method:  method,
				Group:   app,
				Url:     app + url,
				Type:    doc.Type,
				Desc:    doc.Desc,
				ResCode: doc.Code,
			}

			if e := AddPath(context.Background(), r); e != nil {
				return common.TraceErrf(e, "failed to report path to goauth")
			}

			c.Log.Debugf("Reported Path: %-6s %-50s Type: %-10s ResCode: %s Desc: %s", r.Method, r.Url, r.Type, r.ResCode, r.Desc)
		}
		return nil
	})
}
