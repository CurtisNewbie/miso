package goauth

import (
	"context"
	"errors"
	"strings"

	"github.com/curtisnewbie/miso/bus"
	"github.com/curtisnewbie/miso/client"
	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/server"
	"github.com/sirupsen/logrus"
)

const (
	// Extra Key (Left in core.StrPair) used when registering HTTP routes using methods like server.GET
	EXTRA_PATH_DOC = "PATH_DOC"

	// Property Key for enabling GoAuth Client, by default it's true
	//
	// goauth-client-go doesn't use it internally, it's only useful for the Callers
	PROP_ENABLE_GOAUTH_CLIENT = "goauth.client.enabled"

	// event bus name for adding paths
	addPathEventBus = "goauth.add-path"

	// event bus name for adding resources
	addResourceEventBus = "goauth.add-resource"
)

func init() {
	core.SetDefProp(PROP_ENABLE_GOAUTH_CLIENT, true)
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
	c := core.EmptyRail()
	tr := client.NewDynTClient(c, "/remote/path/resource/access-test", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return nil, tr.Err
	}

	if err := tr.Require2xx(); err != nil {
		return nil, err
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
	c := core.EmptyRail()
	tr := client.NewDynTClient(c, "/remote/resource/add", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return tr.Err
	}

	if err := tr.Require2xx(); err != nil {
		return err
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
	c := core.EmptyRail()
	tr := client.NewDynTClient(c, "/remote/path/add", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return tr.Err
	}

	if err := tr.Require2xx(); err != nil {
		return err
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
	c := core.EmptyRail()
	tr := client.NewDynTClient(c, "/remote/role/info", "goauth").
		EnableTracing().
		PostJson(req)

	if tr.Err != nil {
		return nil, tr.Err
	}

	if err := tr.Require2xx(); err != nil {
		return nil, err
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

// Check whether goauth client is enabled
//
//	"goauth.client.enabled"
func IsEnabled() bool {
	return core.GetPropBool(PROP_ENABLE_GOAUTH_CLIENT)
}

func PathDocExtra(doc PathDoc) core.StrPair {
	return core.StrPair{Left: EXTRA_PATH_DOC, Right: doc}
}

// Register a hook to report paths to GoAuth on server bootstrapped
//
// When using methods like server.Get(...), the extra field should contains a
// core.StrPair where the key is EXTRA_PATH_DOC, so that the PathDoc can be picked
// and reported to GoAuth
//
// For example:
//
//	server.Get(url, handler, gclient.PathDocExtra(pathDoc))
//
// This method checks if the goauth client is enabled, nothing will happen if the client is disabled.
func ReportPathsOnBootstrapped(rail core.Rail) {
	if !IsEnabled() {
		rail.Debug("GoAuth client disabled, will not report paths")
		return
	}

	bus.DeclareEventBus(addPathEventBus)

	server.PostServerBootstrapped(func(rail core.Rail) error {
		app := core.GetPropStr(core.PROP_APP_NAME)
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

			// if e := AddPath(context.Background(), r); e != nil {
			// 	return core.TraceErrf(e, "failed to report path to goauth")
			// }

			// report the path asynchronously
			if err := AddPathAsync(rail, r); err != nil {
				return err
			}

			// rail.Debugf("Reported Path: %-6s %-50s Type: %-10s ResCode: %s Desc: %s", r.Method, r.Url, r.Type, r.ResCode, r.Desc)
		}
		return nil
	})
}

// Report path asynchronously
func AddPathAsync(rail core.Rail, req CreatePathReq) error {
	return bus.SendToEventBus(rail, req, addPathEventBus)
}

// Report resource asynchronously
func AddResourceAsync(rail core.Rail, req AddResourceReq) error {
	return bus.SendToEventBus(rail, req, addResourceEventBus)
}

// Register a hook to report resources to GoAuth on server bootstrapped
//
// This method checks if the goauth client is enabled, nothing will happen if the client is disabled.
func ReportResourcesOnBootstrapped(rail core.Rail, reqs []AddResourceReq) {
	if !IsEnabled() {
		rail.Debug("GoAuth client disabled, will not report resources")
		return
	}

	bus.DeclareEventBus(addResourceEventBus)

	server.PostServerBootstrapped(func(rail core.Rail) error {
		for _, req := range reqs {
			if e := AddResourceAsync(rail, req); e != nil {
				rail.Errorf("Failed to report resource, %v", e)
				return e
			}
		}
		return nil
	})
}
