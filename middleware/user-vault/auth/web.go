package auth

import (
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/miso"
)

const (
	ScopeProtected string = "PROTECTED"
	ScopePublic    string = "PUBLIC"
)

var (
	registeredResources = []Resource{}
	callbackRegistered  = false
)

var (
	loadResourcePathOnce sync.Once
	loadedResources      = []Resource{}
	loadedPaths          = []Endpoint{}
)

type Endpoint struct {
	Type    string `json:"type" desc:"access scope type: PROTECTED/PUBLIC"`
	Url     string `json:"url" desc:"endpoint url"`
	Group   string `json:"group" desc:"app name"`
	Desc    string `json:"desc" desc:"description of the endpoint"`
	ResCode string `json:"resCode" desc:"resource code"`
	Method  string `json:"method" desc:"http method"`
}

type Resource struct {
	Name string `json:"name" desc:"resource name"`
	Code string `json:"code" desc:"resource code, unique identifier"`
}

type ResourceInfoRes struct {
	Resources []Resource
	Paths     []Endpoint
}

// Create endpoint to expose resources and endpoint paths to be collected by user-vault.
func ExposeResourceInfo(res []Resource, extraEndpoints ...Endpoint) {

	registeredResources = append(registeredResources, res...)

	if callbackRegistered {
		return
	}

	miso.PreServerBootstrap(func(rail miso.Rail) error {

		// resources and paths are polled by uservault
		miso.Get("/auth/resource", ServeResourceInfo(extraEndpoints...)).
			Desc("Expose resource and endpoint information to other backend service for authorization.").
			Protected().
			DocJsonResp(ResourceInfoRes{})

		return nil
	})

	callbackRegistered = true
}

func ServeResourceInfo(extraEndpoints ...Endpoint) func(inb *miso.Inbound) (any, error) {
	return func(inb *miso.Inbound) (any, error) {

		// resources and paths are lazily loaded
		loadResourcePathOnce.Do(func() {
			app := miso.GetPropStr(miso.PropAppName)
			for _, res := range registeredResources {
				if res.Code == "" || res.Name == "" {
					continue
				}
				loadedResources = append(loadedResources, res)
			}

			routes := miso.GetHttpRoutes()
			for _, route := range routes {
				if route.Url == "" {
					continue
				}
				var routeType = ScopeProtected
				if route.Scope == miso.ScopePublic {
					routeType = ScopePublic
				}

				url := route.Url
				if !strings.HasPrefix(url, "/") {
					url = "/" + url
				}

				r := Endpoint{
					Method:  route.Method,
					Group:   app,
					Url:     "/" + app + url,
					Type:    routeType,
					Desc:    route.Desc,
					ResCode: route.Resource,
				}
				loadedPaths = append(loadedPaths, r)
			}

			if len(extraEndpoints) > 0 {
				loadedPaths = append(loadedPaths, extraEndpoints...)
			}
		})

		return ResourceInfoRes{
			Resources: loadedResources,
			Paths:     loadedPaths,
		}, nil
	}
}
