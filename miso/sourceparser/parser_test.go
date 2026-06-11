package sourceparser

import (
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

// writeTestFile creates a temporary Go file with the given content and returns its path.
func writeTestFile(t *testing.T, name string, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// parseSingle parses a single test file and returns the first endpoint, or fails.
func parseSingle(t *testing.T, content string) *ParsedEndpoint {
	t.Helper()
	path := writeTestFile(t, "test.go", content)
	eps, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(eps) == 0 {
		t.Fatal("expected at least 1 endpoint, got 0")
	}
	return eps[0]
}

// === Handler Type Tests ===

func TestParseFile_AutoHandler(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	))
}`)
	if ep.Method != "POST" {
		t.Errorf("Method = %q, want %q", ep.Method, "POST")
	}
	if ep.URL != "/api/v1" {
		t.Errorf("Url = %q, want %q", ep.URL, "/api/v1")
	}
	if ep.Handler != "AutoHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "AutoHandler")
	}
	if ep.RequestRef == nil {
		t.Fatal("RequestRef is nil")
	}
	if ep.RequestRef.Name != "Req" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "Req")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_ResHandler(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v2", miso.ResHandler(
		func(inb *miso.Inbound) (Res, error) { return nil, nil },
	))
}`)
	if ep.Handler != "ResHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "ResHandler")
	}
	if ep.RequestRef != nil {
		t.Error("RequestRef should be nil for ResHandler (no req param)")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_RawHandler(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v3", miso.RawHandler(
		func(inb *miso.Inbound) { handlerFunc(inb) },
	))
}`)
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
}

func TestParseFile_RawHandler_IdentArg(t *testing.T) {
	// miso.RawHandler(ident) — handler is "RawHandler" (the wrapper call)
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v4", miso.RawHandler(handlerFunc))
}`)
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
}

func TestParseFile_DirectHandlerNoWrapper(t *testing.T) {
	// Direct function reference without any wrapper → "Direct"
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v4", handlerFunc)
}`)
	if ep.Handler != "Direct" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "Direct")
	}
}

func TestParseFile_RawHandler_WithDocJsonResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v5", miso.RawHandler(
		func(inb *miso.Inbound) { handlerFunc(inb) },
	)).DocJsonResp(PostRes{})
}`)
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef should be set via DocJsonResp")
	}
	if ep.ResponseRef.Name != "PostRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "PostRes")
	}
}

// === Chained Method Tests ===

func TestParseFile_ChainedDesc(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Desc("Create resource").Public().Scope("ADMIN")
}`)
	if ep.Desc != "Create resource" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "Create resource")
	}
	if ep.Scope != "ADMIN" {
		t.Errorf("Scope = %q, want %q", ep.Scope, "ADMIN")
	}
}

func TestParseFile_ChainedDocQueryParam(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (Res, error) { return nil, nil },
	)).DocQueryParam("page", "current page").DocHeader("Authorization", "Bearer token")
}`)
	if len(ep.QueryParams) != 1 {
		t.Fatalf("QueryParams len = %d, want 1", len(ep.QueryParams))
	}
	if ep.QueryParams[0].Left != "page" || ep.QueryParams[0].Right != "current page" {
		t.Errorf("QueryParam = %q/%q, want %q/%q",
			ep.QueryParams[0].Left, ep.QueryParams[0].Right, "page", "current page")
	}
	if len(ep.Headers) != 1 {
		t.Fatalf("Headers len = %d, want 1", len(ep.Headers))
	}
	if ep.Headers[0].Left != "Authorization" || ep.Headers[0].Right != "Bearer token" {
		t.Errorf("Header = %q/%q, want %q/%q",
			ep.Headers[0].Left, ep.Headers[0].Right, "Authorization", "Bearer token")
	}
}

func TestParseFile_ChainedExtra(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Extra(miso.ExtraName, "myApi").Extra(miso.ExtraNgTable, true)
}`)
	if len(ep.Extras) != 2 {
		t.Fatalf("Extras len = %d, want 2", len(ep.Extras))
	}
	if ep.FuncName != "myApi" {
		t.Errorf("FuncName = %q, want %q", ep.FuncName, "myApi")
	}
	if ep.Extras[0].Left != "miso.ExtraName" || ep.Extras[0].Right != "myApi" {
		t.Errorf("Extra[0] = %q/%q, want %q/%q", ep.Extras[0].Left, ep.Extras[0].Right, "miso.ExtraName", "myApi")
	}
	if ep.Extras[1].Left != "miso.ExtraNgTable" || ep.Extras[1].Right != "true" {
		t.Errorf("Extra[1] = %q/%q, want %q/%q", ep.Extras[1].Left, ep.Extras[1].Right, "miso.ExtraNgTable", "true")
	}
}

func TestParseFile_DocJsonReq(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.RawHandler(
		func(inb *miso.Inbound) { handlerFunc(inb) },
	)).DocJsonReq(ApiReq{})
}`)
	if ep.RequestRef == nil {
		t.Fatal("RequestRef should be set via DocJsonReq")
	}
	if ep.RequestRef.Name != "ApiReq" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "ApiReq")
	}
}

func TestParseFile_Resource(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Resource("user-service")
}`)
	if ep.Resource != "user-service" {
		t.Errorf("Resource = %q, want %q", ep.Resource, "user-service")
	}
}

// === Type Extraction Tests ===

func TestParseFile_PointerReqAndResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req *Req) (*Res, error) { return nil, nil },
	))
}`)
	if !ep.RequestRef.IsPtr {
		t.Error("RequestRef.IsPtr should be true for *Req")
	}
	if ep.RequestRef.Name != "Req" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "Req")
	}
	if !ep.ResponseRef.IsPtr {
		t.Error("ResponseRef.IsPtr should be true for *Res")
	}
	if ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_SliceReq(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req []Req) (Res, error) { return nil, nil },
	))
}`)
	if !ep.RequestRef.IsSlice {
		t.Error("RequestRef.IsSlice should be true for []Req")
	}
	if ep.RequestRef.IsSlicePtr {
		t.Error("RequestRef.IsSlicePtr should be false for []Req (not []*Req)")
	}
}

func TestParseFile_SliceOfPointerResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) ([]*Res, error) { return nil, nil },
	))
}`)
	if !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef.IsSlice should be true for []*Res")
	}
	if !ep.ResponseRef.IsSlicePtr {
		t.Error("ResponseRef.IsSlicePtr should be true for []*Res")
	}
	if ep.ResponseRef.IsPtr {
		t.Error("ResponseRef.IsPtr should be false (pointer is inside slice []*T)")
	}
	if ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_PointerToSliceReq(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req *[]Req) (Res, error) { return nil, nil },
	))
}`)
	if !ep.RequestRef.IsPtr {
		t.Error("RequestRef.IsPtr should be true for *[]Req")
	}
	if !ep.RequestRef.IsSlice {
		t.Error("RequestRef.IsSlice should be true for *[]Req")
	}
	if ep.RequestRef.IsSlicePtr {
		t.Error("RequestRef.IsSlicePtr should be false for *[]Req (not *[]*Req)")
	}
	if ep.RequestRef.Name != "Req" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "Req")
	}
}

func TestParseFile_PointerToSliceOfPointerResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (*[]*Res, error) { return nil, nil },
	))
}`)
	if !ep.ResponseRef.IsPtr {
		t.Error("ResponseRef.IsPtr should be true for *[]*Res")
	}
	if !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef.IsSlice should be true for *[]*Res")
	}
	if !ep.ResponseRef.IsSlicePtr {
		t.Error("ResponseRef.IsSlicePtr should be true for *[]*Res")
	}
	if ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_SliceOfValuesResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) ([]Res, error) { return nil, nil },
	))
}`)
	if !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef.IsSlice should be true")
	}
	if ep.ResponseRef.IsSlicePtr {
		t.Error("ResponseRef.IsSlicePtr should be false for []Res (not []*Res)")
	}
}

// === Generic Type Tests ===

func TestParseFile_GenericResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (miso.PageRes[Res], error) { return nil, nil },
	))
}`)
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "PageRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "PageRes")
	}
	if ep.ResponseRef.PkgName != "miso" {
		t.Errorf("ResponseRef.PkgName = %q, want %q", ep.ResponseRef.PkgName, "miso")
	}
	if len(ep.ResponseRef.TypeArgs) != 1 {
		t.Fatalf("TypeArgs len = %d, want 1", len(ep.ResponseRef.TypeArgs))
	}
	if ep.ResponseRef.TypeArgs[0].Name != "Res" {
		t.Errorf("TypeArg[0].Name = %q, want %q", ep.ResponseRef.TypeArgs[0].Name, "Res")
	}
}

func TestParseFile_GenericWithPointerArg(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (miso.PageRes[*Res], error) { return nil, nil },
	))
}`)
	if len(ep.ResponseRef.TypeArgs) != 1 {
		t.Fatalf("TypeArgs len = %d, want 1", len(ep.ResponseRef.TypeArgs))
	}
	if !ep.ResponseRef.TypeArgs[0].IsPtr {
		t.Error("TypeArg[0].IsPtr should be true for *Res")
	}
	if ep.ResponseRef.TypeArgs[0].Name != "Res" {
		t.Errorf("TypeArg[0].Name = %q, want %q", ep.ResponseRef.TypeArgs[0].Name, "Res")
	}
}

func TestParseFile_GenericWithSliceArg(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (miso.PageRes[[]Res], error) { return nil, nil },
	))
}`)
	if len(ep.ResponseRef.TypeArgs) != 1 {
		t.Fatalf("TypeArgs len = %d, want 1", len(ep.ResponseRef.TypeArgs))
	}
	if !ep.ResponseRef.TypeArgs[0].IsSlice {
		t.Error("TypeArg[0].IsSlice should be true for []Res")
	}
}

func TestParseFile_PointerToGenericResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (*miso.PageRes[Res], error) { return nil, nil },
	))
}`)
	if !ep.ResponseRef.IsPtr {
		t.Error("ResponseRef.IsPtr should be true for *PageRes[Res]")
	}
	if ep.ResponseRef.Name != "PageRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "PageRes")
	}
	if len(ep.ResponseRef.TypeArgs) != 1 {
		t.Fatalf("TypeArgs len = %d, want 1", len(ep.ResponseRef.TypeArgs))
	}
}

// === Map Type Tests ===

func TestParseFile_MapResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (map[string]int32, error) { return nil, nil },
	))
}`)
	if !ep.ResponseRef.IsMap {
		t.Error("ResponseRef.IsMap should be true")
	}
	if ep.ResponseRef.MapKey == nil || ep.ResponseRef.MapKey.Name != "string" {
		t.Errorf("MapKey.Name = %v, want %q",
			func() string {
				if ep.ResponseRef.MapKey != nil {
					return ep.ResponseRef.MapKey.Name
				}
				return "nil"
			}(), "string")
	}
	if ep.ResponseRef.MapValue == nil || ep.ResponseRef.MapValue.Name != "int32" {
		t.Errorf("MapValue.Name = %v, want %q",
			func() string {
				if ep.ResponseRef.MapValue != nil {
					return ep.ResponseRef.MapValue.Name
				}
				return "nil"
			}(), "int32")
	}
}

// === Cross-Package Type Tests ===

func TestParseFile_CrossPackageReqType(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "github.com/curtisnewbie/miso/demo/api"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req api.PostReq) (api.PostRes, error) { return nil, nil },
	))
}`)
	if ep.RequestRef.PkgName != "api" {
		t.Errorf("RequestRef.PkgName = %q, want %q", ep.RequestRef.PkgName, "api")
	}
	if ep.RequestRef.Name != "PostReq" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "PostReq")
	}
	if ep.ResponseRef.PkgName != "api" {
		t.Errorf("ResponseRef.PkgName = %q, want %q", ep.ResponseRef.PkgName, "api")
	}
	if ep.ResponseRef.Name != "PostRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "PostRes")
	}
}

func TestParseFile_CrossPackagePointerReqType(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/demo/api"
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req *api.PostReq) (*api.PostRes, error) { return nil, nil },
	))
}`)
	if !ep.RequestRef.IsPtr {
		t.Error("RequestRef.IsPtr should be true for *api.PostReq")
	}
	if ep.RequestRef.PkgName != "api" {
		t.Errorf("RequestRef.PkgName = %q, want %q", ep.RequestRef.PkgName, "api")
	}
	if ep.ResponseRef.PkgName != "api" {
		t.Errorf("ResponseRef.PkgName = %q, want %q", ep.ResponseRef.PkgName, "api")
	}
}

// === Skip Param Tests ===

func TestParseFile_SkipInboundParam(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || ep.RequestRef.Name != "Req" {
		t.Error("*miso.Inbound should be skipped, req should be the request param")
	}
}

func TestParseFile_SkipRailParam(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.RawHandler(
		func(inb *miso.Inbound, rail miso.Rail) { handlerFunc(inb, rail) },
	))
}`)
	if ep.RequestRef != nil {
		t.Errorf("RequestRef should be nil when all params are skipped, got Name=%q", ep.RequestRef.Name)
	}
}

func TestParseFile_SkipDBParam(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req, db *gorm.DB) (Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || ep.RequestRef.Name != "Req" {
		t.Error("*gorm.DB should be skipped, req should be the request param")
	}
}

// === BaseRoute.Group Tests ===

func TestParseFile_BaseRouteGroup(t *testing.T) {
	eps, err := ParseFile(writeTestFile(t, "test.go", `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.BaseRoute("/open/api/grouped").Group(
		miso.HttpPost("/post", miso.AutoHandler(
			func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
		)).Desc("Post item"),
		miso.HttpGet("/get", miso.ResHandler(
			func(inb *miso.Inbound) (Res, error) { return nil, nil },
		)),
	)
}`))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints in Group, got %d", len(eps))
	}
	// Check POST endpoint
	if eps[0].Method != "POST" {
		t.Errorf("eps[0].Method = %q, want %q", eps[0].Method, "POST")
	}
	if eps[0].URL != "/open/api/grouped/post" {
		t.Errorf("eps[0].URL = %q, want %q", eps[0].URL, "/open/api/grouped/post")
	}
	if eps[0].Desc != "Post item" {
		t.Errorf("eps[0].Desc = %q, want %q", eps[0].Desc, "Post item")
	}
	// Check GET endpoint
	if eps[1].Method != "GET" {
		t.Errorf("eps[1].Method = %q, want %q", eps[1].Method, "GET")
	}
	if eps[1].URL != "/open/api/grouped/get" {
		t.Errorf("eps[1].URL = %q, want %q", eps[1].URL, "/open/api/grouped/get")
	}
}

// === All HTTP Methods ===

func TestParseFile_AllHTTPMethods(t *testing.T) {
	methods := map[string]string{
		"HttpPost":    "POST",
		"HttpGet":     "GET",
		"HttpPut":     "PUT",
		"HttpDelete":  "DELETE",
		"HttpPatch":   "PATCH",
		"HttpHead":    "HEAD",
		"HttpOptions": "OPTIONS",
		"HttpConnect": "CONNECT",
		"HttpTrace":   "TRACE",
	}
	for callName, wantMethod := range methods {
		t.Run(callName, func(t *testing.T) {
			ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.`+callName+`("/api/v1", miso.RawHandler(handlerFunc))
}`)
			if ep.Method != wantMethod {
				t.Errorf("Method = %q, want %q", ep.Method, wantMethod)
			}
		})
	}
}

// === Error-Only and Void Return Tests ===

func TestParseFile_ErrorOnlyReturn(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) error { return nil },
	))
}`)
	if ep.ResponseRef != nil {
		t.Error("ResponseRef should be nil when only error is returned")
	}
}

func TestParseFile_VoidReturn(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.RawHandler(
		func(inb *miso.Inbound) { handlerFunc(inb) },
	))
}`)
	if ep.ResponseRef != nil {
		t.Error("ResponseRef should be nil when handler has no return")
	}
}

func TestParseFile_AnyReturn(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (any, error) { return nil, nil },
	))
}`)
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "any" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "any")
	}
}

// === DocQueryReq and DocHeaderReq Tests ===

func TestParseFile_DocQueryReq(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (Res, error) { return nil, nil },
	)).DocQueryReq(QueryParam{})
}`)
	if ep.QueryReqType != "QueryParam" {
		t.Errorf("QueryReqType = %q, want %q", ep.QueryReqType, "QueryParam")
	}
}

func TestParseFile_DocHeaderReq(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (Res, error) { return nil, nil },
	)).DocHeaderReq(HeaderStruct{})
}`)
	if ep.HeaderReqType != "HeaderStruct" {
		t.Errorf("HeaderReqType = %q, want %q", ep.HeaderReqType, "HeaderStruct")
	}
}

// === Empty Struct Req ===

func TestParseFile_EmptyStructReq(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req EmptyReq) error { return nil },
	))
}`)
	if ep.RequestRef == nil {
		t.Fatal("RequestRef is nil")
	}
	if ep.RequestRef.Name != "EmptyReq" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "EmptyReq")
	}
}

// === TypeRef String Methods ===

func TestTypeRef_String(t *testing.T) {
	tests := []struct {
		name string
		ref  TypeRef
		want string
	}{
		{"simple", TypeRef{Name: "Req"}, "Req"},
		{"with pkg", TypeRef{PkgName: "api", Name: "Req"}, "api.Req"},
		{"generic", TypeRef{PkgName: "miso", Name: "PageRes", TypeArgs: []TypeRef{{Name: "Res"}}}, "miso.PageRes[Res]"},
		// Note: String() does NOT include */[] prefixes — use FullString() for those
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTypeRef_FullString(t *testing.T) {
	tests := []struct {
		name string
		ref  TypeRef
		want string
	}{
		{"simple", TypeRef{Name: "Req"}, "Req"},
		{"pointer", TypeRef{Name: "Req", IsPtr: true}, "*Req"},
		{"slice", TypeRef{Name: "Req", IsSlice: true}, "[]Req"},
		{"slice ptr", TypeRef{Name: "Req", IsSlice: true, IsSlicePtr: true}, "[]*Req"},
		{"ptr to slice", TypeRef{Name: "Req", IsPtr: true, IsSlice: true}, "*[]Req"},
		{"ptr to slice ptr", TypeRef{Name: "Req", IsPtr: true, IsSlice: true, IsSlicePtr: true}, "*[]*Req"},
		{"map", TypeRef{IsMap: true, MapKey: &TypeRef{Name: "string"}, MapValue: &TypeRef{Name: "int32"}}, "map[string]int32"},
		{"ptr to map", TypeRef{Name: "Req", IsPtr: true, IsMap: true, MapKey: &TypeRef{Name: "string"}, MapValue: &TypeRef{Name: "int32"}}, "*map[string]int32"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ref.FullString()
			if got != tt.want {
				t.Errorf("FullString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// === SkipParamType Tests ===

func TestIsSkipParamType(t *testing.T) {
	path := writeTestFile(t, "test.go", `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/a", miso.AutoHandler(func(inb *miso.Inbound, req Req, db *gorm.DB) (Res, error) { return nil, nil }))
	miso.HttpPost("/b", miso.RawHandler(func(inb *miso.Inbound, rail miso.Rail) { }))
}`)
	eps, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(eps))
	}
	// Endpoint /a: *miso.Inbound skipped, *gorm.DB skipped, req should be Req
	if eps[0].RequestRef == nil || eps[0].RequestRef.Name != "Req" {
		t.Errorf("eps[0] RequestRef should be Req, got %v", eps[0].RequestRef)
	}
	// Endpoint /b: both params skipped (Inbound + Rail)
	if eps[1].RequestRef != nil {
		t.Errorf("eps[1] RequestRef should be nil (all params skipped), got %v", eps[1].RequestRef)
	}
}

// Test ParseFile handles files with no endpoint registrations (e.g., bare
// functions with comments only — ParseFile only picks up AST call chains).
func TestParseFile_NoMisoCalls(t *testing.T) {
	path := writeTestFile(t, "test.go", `package test
import "github.com/curtisnewbie/miso/miso"

// misoapi-http: POST /api/v1
func handlerFunc(inb *miso.Inbound, req Req) (Res, error) { return nil, nil }
`)
	eps, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(eps) != 0 {
		t.Errorf("expected 0 endpoints (comment-only, no AST calls), got %d", len(eps))
	}
}

// Test ParseFile handles file with chained Extra calls with boolean values.
func TestParseFile_ExtraBoolean(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Extra(miso.ExtraNgTable, true)
}`)
	if len(ep.Extras) != 1 {
		t.Fatalf("Extras len = %d, want 1", len(ep.Extras))
	}
	if ep.Extras[0].Left != "miso.ExtraNgTable" {
		t.Errorf("Extra.Left = %q, want %q", ep.Extras[0].Left, "miso.ExtraNgTable")
	}
	if ep.Extras[0].Right != "true" {
		t.Errorf("Extra.Right = %q, want %q", ep.Extras[0].Right, "true")
	}
}

// Test ParseFile with multiple endpoints in one init function.
func TestParseFile_MultipleEndpoints(t *testing.T) {
	eps, err := ParseFile(writeTestFile(t, "test.go", `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	))
	miso.HttpGet("/api/v2", miso.ResHandler(
		func(inb *miso.Inbound) (Res, error) { return nil, nil },
	))
}`))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(eps) != 2 {
		t.Fatalf("expected 2 endpoints, got %d", len(eps))
	}
	if eps[0].Method != "POST" || eps[0].URL != "/api/v1" {
		t.Errorf("eps[0] = %s %s, want POST /api/v1", eps[0].Method, eps[0].URL)
	}
	if eps[1].Method != "GET" || eps[1].URL != "/api/v2" {
		t.Errorf("eps[1] = %s %s, want GET /api/v2", eps[1].Method, eps[1].URL)
	}
}

// Test ParseFile with any handler (HttpAny for catch-all routes).
func TestParseFile_HttpAny(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpAny("/api/v1", miso.RawHandler(handlerFunc))
}`)
	if ep.Method != "ANY" {
		t.Errorf("Method = %q, want %q", ep.Method, "ANY")
	}
}

func TestParseFile_VariableURL(t *testing.T) {
	// URL can be a variable (not just a string literal), e.g.,
	// miso.HttpGet(deregisterURL, miso.ResHandler(...)).
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet(deregisterURL, miso.ResHandler(
		func(inb *miso.Inbound) (any, error) { return nil, nil },
	))
}`)
	if ep.URL != "/${deregisterURL}" {
		t.Errorf("URL = %q, want %q (variable ident should be captured)", ep.URL, "/${deregisterURL}")
	}
	if ep.Handler != "ResHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "ResHandler")
	}
	if ep.ResponseRef == nil || ep.ResponseRef.Name != "any" {
		t.Errorf("ResponseRef should be 'any', got %v", ep.ResponseRef)
	}
}

func TestParseFile_DescVariableArg(t *testing.T) {
	// Undefined variable names fall back to the identifier name.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Desc(deregisterDesc)
}`)
	if ep.Desc != "deregisterDesc" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "deregisterDesc")
	}
}

func TestParseFile_DescBinaryExprArg(t *testing.T) {
	// Desc("prefix " + constName) should concatenate string parts.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
const PropKey = "some.prop.key"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Desc("Endpoint desc. Configurable using '" + PropKey + "'")
}`)
	if ep.Desc != "Endpoint desc. Configurable using 'some.prop.key'" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "Endpoint desc. Configurable using 'some.prop.key'")
	}
}

func TestParseFile_ResourceConstArg(t *testing.T) {
	// Resource(constName) should resolve to the const's string value.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
const ResCodeUpload = "fstore:upload"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Resource(ResCodeUpload)
}`)
	if ep.Resource != "fstore:upload" {
		t.Errorf("Resource = %q, want %q", ep.Resource, "fstore:upload")
	}
}

func TestParseFile_ScopeConstArg(t *testing.T) {
	// Scope(constName) should resolve to the const's string value.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
const ScopeAdmin = "ADMIN"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Scope(ScopeAdmin)
}`)
	if ep.Scope != "ADMIN" {
		t.Errorf("Scope = %q, want %q", ep.Scope, "ADMIN")
	}
}

func TestParseFile_DescConstArg(t *testing.T) {
	// Desc(constName) should resolve to the const's string value.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
const ErrDesc = "something went wrong"
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Desc(ErrDesc)
}`)
	if ep.Desc != "something went wrong" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "something went wrong")
	}
}

func TestParseFile_VarStringArg(t *testing.T) {
	// var with string value should also be resolved.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
var ResCodeDelete = "fstore:delete"
func init() {
	miso.HttpDelete("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Resource(ResCodeDelete)
}`)
	if ep.Resource != "fstore:delete" {
		t.Errorf("Resource = %q, want %q", ep.Resource, "fstore:delete")
	}
}

// === ResHandler type pattern tests ===

func TestParseFile_ResHandler_SliceResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) ([]Res, error) { return nil, nil },
	))
}`)
	if ep.Handler != "ResHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "ResHandler")
	}
	if ep.RequestRef != nil {
		t.Error("RequestRef should be nil for ResHandler")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef.IsSlice should be true for []Res")
	}
	if ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_ResHandler_AnyResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/api/v1", miso.ResHandler(
		func(inb *miso.Inbound) (any, error) { return nil, nil },
	))
}`)
	if ep.Handler != "ResHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "ResHandler")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "any" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "any")
	}
}

// === Combined decorator tests ===

func TestParseFile_DocJsonReqAndResp(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.RawHandler(
		func(inb *miso.Inbound) { handlerFunc(inb) },
	)).DocJsonReq(ApiReq{}).DocJsonResp(PostRes{})
}`)
	if ep.RequestRef == nil {
		t.Fatal("RequestRef should be set via DocJsonReq")
	}
	if ep.RequestRef.Name != "ApiReq" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "ApiReq")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef should be set via DocJsonResp")
	}
	if ep.ResponseRef.Name != "PostRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "PostRes")
	}
}

func TestParseFile_RawHandler_IdentDocJsonResp(t *testing.T) {
	// RawHandler with ident arg + DocJsonResp (no DocJsonReq). Matches /api/v23 pattern.
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpPost("/api/v1", miso.RawHandler(handlerFunc)).DocJsonResp(PostRes{})
}`)
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
	if ep.RequestRef != nil {
		t.Error("RequestRef should be nil (no DocJsonReq)")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef should be set via DocJsonResp")
	}
	if ep.ResponseRef.Name != "PostRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "PostRes")
	}
}

// === AutoHandler + inject *gorm.DB combination patterns ===
// Each individual feature (pointer, slice, inject skip, any) is tested
// in isolation above. These tests verify they compose correctly.

func TestParseFile_AutoHandler_PtrReq_PtrResp_Inject(t *testing.T) {
	// Matches /api/v6, /api/v8: *req + *resp + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v6", miso.AutoHandler(
		func(inb *miso.Inbound, req *Req, db *gorm.DB) (*Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || !ep.RequestRef.IsPtr || ep.RequestRef.Name != "Req" {
		t.Errorf("RequestRef: ptr=%v name=%q, want ptr=true name=Req",
			ep.RequestRef != nil && ep.RequestRef.IsPtr,
			func() string {
				if ep.RequestRef != nil {
					return ep.RequestRef.Name
				}
				return ""
			}())
	}
	if ep.ResponseRef == nil || !ep.ResponseRef.IsPtr || ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef: ptr=%v name=%q, want ptr=true name=Res",
			ep.ResponseRef != nil && ep.ResponseRef.IsPtr,
			func() string {
				if ep.ResponseRef != nil {
					return ep.ResponseRef.Name
				}
				return ""
			}())
	}
}

func TestParseFile_AutoHandler_PtrReq_ValueResp_Inject(t *testing.T) {
	// Matches /api/v7: *req + value resp + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v7", miso.AutoHandler(
		func(inb *miso.Inbound, req *Req, db *gorm.DB) (Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || !ep.RequestRef.IsPtr {
		t.Error("RequestRef should be pointer-to-Req")
	}
	if ep.ResponseRef == nil || ep.ResponseRef.IsPtr {
		t.Error("ResponseRef should be value Res, not pointer")
	}
	if ep.ResponseRef != nil && ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

func TestParseFile_AutoHandler_PtrReq_SlicePtrResp_Inject(t *testing.T) {
	// Matches /api/v9: *req + []*resp + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v9", miso.AutoHandler(
		func(inb *miso.Inbound, req *Req, db *gorm.DB) ([]*Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || !ep.RequestRef.IsPtr {
		t.Error("RequestRef should be pointer-to-Req")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef.IsSlice should be true")
	}
	if !ep.ResponseRef.IsSlicePtr {
		t.Error("ResponseRef.IsSlicePtr should be true for []*Res")
	}
	if ep.ResponseRef.IsPtr {
		t.Error("ResponseRef.IsPtr should be false (pointer inside slice)")
	}
}

func TestParseFile_AutoHandler_PtrReq_SliceValResp_Inject(t *testing.T) {
	// Matches /api/v10, /api/v11: *req + []resp + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v10", miso.AutoHandler(
		func(inb *miso.Inbound, req *Req, db *gorm.DB) ([]Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || !ep.RequestRef.IsPtr {
		t.Error("RequestRef should be pointer-to-Req")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef.IsSlice should be true")
	}
	if ep.ResponseRef.IsSlicePtr {
		t.Error("ResponseRef.IsSlicePtr should be false for []Res (not []*Res)")
	}
}

func TestParseFile_AutoHandler_SliceReq_SliceValResp_Inject(t *testing.T) {
	// Matches /api/v12: []req + []resp + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v12", miso.AutoHandler(
		func(inb *miso.Inbound, req []Req, db *gorm.DB) ([]Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || !ep.RequestRef.IsSlice {
		t.Error("RequestRef should be slice of Req")
	}
	if ep.ResponseRef == nil || !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef should be slice of Res")
	}
}

func TestParseFile_AutoHandler_SliceReq_AnyResp_Inject(t *testing.T) {
	// Matches /api/v13: []req + any + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v13", miso.AutoHandler(
		func(inb *miso.Inbound, req []Req, db *gorm.DB) (any, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || !ep.RequestRef.IsSlice {
		t.Error("RequestRef should be slice of Req")
	}
	if ep.ResponseRef == nil || ep.ResponseRef.Name != "any" {
		t.Error("ResponseRef should be 'any'")
	}
}

func TestParseFile_AutoHandler_ValueReq_SliceValResp_Inject(t *testing.T) {
	// Matches /api/v14: value req + []resp + *gorm.DB
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
import "gorm.io/gorm"
func init() {
	miso.HttpPost("/api/v14", miso.AutoHandler(
		func(inb *miso.Inbound, req Req, db *gorm.DB) ([]Res, error) { return nil, nil },
	))
}`)
	if ep.RequestRef == nil || ep.RequestRef.IsPtr {
		t.Error("RequestRef should be value Req (not pointer)")
	}
	if ep.ResponseRef == nil || !ep.ResponseRef.IsSlice {
		t.Error("ResponseRef should be slice of Res")
	}
	if ep.ResponseRef != nil && ep.ResponseRef.Name != "Res" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "Res")
	}
}

// === resolveConstStr unit tests ===

func TestResolveConstStr(t *testing.T) {
	t.Run("nil object", func(t *testing.T) {
		if got := resolveConstStr(nil); got != "" {
			t.Errorf("nil → %q, want %q", got, "")
		}
	})
	t.Run("string const", func(t *testing.T) {
		pkg := types.NewPackage("testpkg", "testpkg")
		c := types.NewConst(token.NoPos, pkg, "StrConst", types.Typ[types.String], constant.MakeString("hello"))
		if got := resolveConstStr(c); got != "hello" {
			t.Errorf("string const → %q, want %q", got, "hello")
		}
	})
	t.Run("non-string const", func(t *testing.T) {
		pkg := types.NewPackage("testpkg", "testpkg")
		c := types.NewConst(token.NoPos, pkg, "IntConst", types.Typ[types.Int], constant.MakeInt64(42))
		if got := resolveConstStr(c); got != "" {
			t.Errorf("int const → %q, want %q", got, "")
		}
	})
	t.Run("non-const object", func(t *testing.T) {
		pkg := types.NewPackage("testpkg", "testpkg")
		v := types.NewVar(token.NoPos, pkg, "v", types.Typ[types.String])
		if got := resolveConstStr(v); got != "" {
			t.Errorf("var → %q, want %q", got, "")
		}
	})
}

// === extractStringArg with *types.Package tests ===

func TestExtractStringArg_WithPackage_LocalIdent(t *testing.T) {
	pkg := types.NewPackage("testpkg", "testpkg")
	c := types.NewConst(token.NoPos, pkg, "MyConst", types.Typ[types.String], constant.MakeString("resolved"))
	pkg.Scope().Insert(c)

	// Construct dst.Ident{Name: "MyConst"}
	ident := &dst.Ident{Name: "MyConst"}
	args := []dst.Expr{ident}

	got := extractStringArg(args, 0, nil, pkg) // constVars=nil, pkg has it
	if got != "resolved" {
		t.Errorf("MyConst via pkg.Scope → %q, want %q", got, "resolved")
	}
}

func TestExtractStringArg_WithPackage_ConstVarsTakesPriority(t *testing.T) {
	pkg := types.NewPackage("testpkg", "testpkg")
	c := types.NewConst(token.NoPos, pkg, "MyConst", types.Typ[types.String], constant.MakeString("fromPkg"))
	pkg.Scope().Insert(c)

	// constVars has a different value — should take priority
	constVars := map[string]string{"MyConst": "fromConstVars"}
	ident := &dst.Ident{Name: "MyConst"}
	args := []dst.Expr{ident}

	got := extractStringArg(args, 0, constVars, pkg)
	if got != "fromConstVars" {
		t.Errorf("constVars should take priority → %q, want %q", got, "fromConstVars")
	}
}

func TestExtractStringArg_WithPackage_BasicLitPriority(t *testing.T) {
	pkg := types.NewPackage("testpkg", "testpkg")
	c := types.NewConst(token.NoPos, pkg, "MyConst", types.Typ[types.String], constant.MakeString("fromPkg"))
	pkg.Scope().Insert(c)

	// BasicLit should take priority over any const resolution
	lit := &dst.BasicLit{Kind: token.STRING, Value: `"literal"`}
	args := []dst.Expr{lit}

	got := extractStringArg(args, 0, map[string]string{"MyConst": "fromConstVars"}, pkg)
	if got != "literal" {
		t.Errorf("BasicLit should take priority → %q, want %q", got, "literal")
	}
}

func TestExtractStringArg_NilPackage_ConstVarsFallback(t *testing.T) {
	// nil pkg + constVars = old behavior should still work
	constVars := map[string]string{"MyVar": "fromVars"}
	ident := &dst.Ident{Name: "MyVar"}
	args := []dst.Expr{ident}

	got := extractStringArg(args, 0, constVars, nil)
	if got != "fromVars" {
		t.Errorf("constVars fallback → %q, want %q", got, "fromVars")
	}
}

func TestExtractStringArg_NilPackage_SelectorExprFallthrough(t *testing.T) {
	// nil pkg + SelectorExpr = falls through to exprToString
	sel := &dst.SelectorExpr{
		X:   &dst.Ident{Name: "pkg"},
		Sel: &dst.Ident{Name: "Const"},
	}
	args := []dst.Expr{sel}

	got := extractStringArg(args, 0, nil, nil)
	// exprToString for SelectorExpr returns "pkg.Const"
	if got != "pkg.Const" {
		t.Errorf("SelectorExpr fallthrough → %q, want %q", got, "pkg.Const")
	}
}

// === ParseFileDst with *types.Package integration tests ===

func TestParseFileDst_WithPackage_LocalConst(t *testing.T) {
	// Same-package const resolution: ParseFileDst with *types.Package
	// where pkg.Scope() contains the const values.
	path := writeTestFile(t, "test.go", `package test
import "github.com/curtisnewbie/miso/miso"
const LocalURL = "/api/local"
const LocalDesc = "test description"
func init() {
	miso.HttpGet(LocalURL, miso.RawHandler(myHandler)).
		Desc(LocalDesc)
}
func myHandler(inb *miso.Inbound) {}
`)

	// Construct a *types.Package with the same consts in scope
	pkg := types.NewPackage("testpkg", "testpkg")
	pkg.Scope().Insert(types.NewConst(
		token.NoPos, pkg, "LocalURL", types.Typ[types.String],
		constant.MakeString("/api/local"),
	))
	pkg.Scope().Insert(types.NewConst(
		token.NoPos, pkg, "LocalDesc", types.Typ[types.String],
		constant.MakeString("test description"),
	))

	f, err := decorator.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	eps := ParseFileDst(f, pkg)
	if len(eps) < 1 {
		t.Fatalf("expected at least 1 endpoint, got %d", len(eps))
	}

	ep := eps[0]
	if ep.URL != "/api/local" {
		t.Errorf("URL: got %q, want %q", ep.URL, "/api/local")
	}
	if ep.Desc != "test description" {
		t.Errorf("Desc: got %q, want %q", ep.Desc, "test description")
	}
}

func TestParseFileDst_NilPackage_ConstVarsStillWorks(t *testing.T) {
	// nil pkg + constVars: AST-based fallback should still resolve bare Ident consts.
	path := writeTestFile(t, "test.go", `package test
import "github.com/curtisnewbie/miso/miso"
const LocalURL = "/api/local"
func init() {
	miso.HttpGet(LocalURL, miso.RawHandler(myHandler))
}
func myHandler(inb *miso.Inbound) {}
`)

	constVars := map[string]string{
		"LocalURL": "/from-constvars",
	}

	f, err := decorator.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	eps := ParseFileDst(f, nil, constVars)
	if len(eps) < 1 {
		t.Fatalf("expected at least 1 endpoint, got %d", len(eps))
	}

	// collectConstVars captures LocalURL="/api/local" from the file source,
	// and it takes priority over extraConsts. So the URL comes from the file,
	// not from the extra constVars.
	ep := eps[0]
	if ep.URL != "/api/local" {
		t.Errorf("URL (nil pkg, file-local takes priority): got %q, want %q", ep.URL, "/api/local")
	}
}

// === BinaryExpr (string concatenation) tests ===

func TestExtractStringArg_BinaryExpr_Literals(t *testing.T) {
	// "a" + "b" → "ab"
	bin := &dst.BinaryExpr{
		X:  &dst.BasicLit{Kind: token.STRING, Value: `"hello "`},
		Op: token.ADD,
		Y:  &dst.BasicLit{Kind: token.STRING, Value: `"world"`},
	}
	args := []dst.Expr{bin}
	got := extractStringArg(args, 0, nil, nil)
	if got != "hello world" {
		t.Errorf("BinaryExpr literals: got %q, want %q", got, "hello world")
	}
}

func TestExtractStringArg_BinaryExpr_LiteralPlusConst(t *testing.T) {
	// "prefix" + LocalConst → "prefixresolved"
	constVars := map[string]string{"LocalConst": "resolved"}
	bin := &dst.BinaryExpr{
		X:  &dst.BasicLit{Kind: token.STRING, Value: `"prefix"`},
		Op: token.ADD,
		Y:  &dst.Ident{Name: "LocalConst"},
	}
	args := []dst.Expr{bin}
	got := extractStringArg(args, 0, constVars, nil)
	if got != "prefixresolved" {
		t.Errorf("BinaryExpr literal + const: got %q, want %q", got, "prefixresolved")
	}
}

func TestExtractStringArg_BinaryExpr_ImportedConst(t *testing.T) {
	// "prefix '" + pkg.Const + "'" → "prefix 'hello'"
	impPkg := types.NewPackage("imported/pkg", "imp")
	impPkg.Scope().Insert(types.NewConst(
		token.NoPos, impPkg, "PropKey", types.Typ[types.String],
		constant.MakeString("some.prop.key"),
	))

	pkg := types.NewPackage("testpkg", "testpkg")
	pkg.SetImports([]*types.Package{impPkg})

	bin := &dst.BinaryExpr{
		X:  &dst.BasicLit{Kind: token.STRING, Value: `"Configurable using '"`},
		Op: token.ADD,
		Y: &dst.SelectorExpr{
			X:   &dst.Ident{Name: "imp"},
			Sel: &dst.Ident{Name: "PropKey"},
		},
	}
	args := []dst.Expr{bin}
	got := extractStringArg(args, 0, nil, pkg)
	if got != "Configurable using 'some.prop.key" {
		t.Errorf("BinaryExpr with imported const: got %q, want %q", got, "Configurable using 'some.prop.key")
	}
}

func TestExtractStringArg_BinaryExpr_NestedConcat(t *testing.T) {
	// "a" + "b" + "c" → "abc" (left-associative BinaryExpr chain)
	// AST: ((a + b) + c)
	inner := &dst.BinaryExpr{
		X:  &dst.BasicLit{Kind: token.STRING, Value: `"a"`},
		Op: token.ADD,
		Y:  &dst.BasicLit{Kind: token.STRING, Value: `"b"`},
	}
	outer := &dst.BinaryExpr{
		X:  inner,
		Op: token.ADD,
		Y:  &dst.BasicLit{Kind: token.STRING, Value: `"c"`},
	}
	args := []dst.Expr{outer}
	got := extractStringArg(args, 0, nil, nil)
	if got != "abc" {
		t.Errorf("Nested BinaryExpr: got %q, want %q", got, "abc")
	}
}

// === ParseFileDst: BinaryExpr with imported const ===

func TestParseFileDst_BinaryExprWithImportedConst(t *testing.T) {
	path := writeTestFile(t, "test.go", `package test
import (
	"github.com/curtisnewbie/miso/miso"
	"imported/pkg"
)
func init() {
	miso.HttpPost("/api/v1", miso.AutoHandler(
		func(inb *miso.Inbound, req Req) (Res, error) { return nil, nil },
	)).Desc("Configurable using '" + imp.PropKey + "'")
}
`)

	impPkg := types.NewPackage("imported/pkg", "imp")
	impPkg.Scope().Insert(types.NewConst(
		token.NoPos, impPkg, "PropKey", types.Typ[types.String],
		constant.MakeString("some.prop.key"),
	))

	pkg := types.NewPackage("testpkg", "testpkg")
	pkg.SetImports([]*types.Package{impPkg})

	f, err := decorator.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	eps := ParseFileDst(f, pkg)
	if len(eps) < 1 {
		t.Fatalf("expected at least 1 endpoint, got %d", len(eps))
	}

	ep := eps[0]
	want := "Configurable using 'some.prop.key'"
	if ep.Desc != want {
		t.Errorf("Desc: got %q, want %q", ep.Desc, want)
	}
}

// === Bare (Unqualified) Endpoint Registration Tests ===
// These test endpoints declared inside `package miso` where HttpGet, RawHandler,
// etc. are called without the `miso.` qualifier.

func TestParseFile_BareHttpGet(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpGet("/metrics",
		RawHandler(func(inb *Inbound) { handler.ServeHTTP(inb.Unwrap()) })).
		Desc("Collect prometheus metrics information").
		DocHeader("Authorization", "Basic authorization if enabled")
}`)
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.URL != "/metrics" {
		t.Errorf("URL = %q, want %q", ep.URL, "/metrics")
	}
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
	if ep.Desc != "Collect prometheus metrics information" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "Collect prometheus metrics information")
	}
	if len(ep.Headers) != 1 {
		t.Fatalf("Headers len = %d, want 1", len(ep.Headers))
	}
	if ep.Headers[0].Left != "Authorization" {
		t.Errorf("Header[0].Left = %q, want %q", ep.Headers[0].Left, "Authorization")
	}
}

func TestParseFile_BareHttpPost(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpPost("/api/users", AutoHandler(
		func(inb *Inbound, req CreateUserReq) (CreateUserRes, error) { return CreateUserRes{}, nil },
	)).Desc("Create user")
}`)
	if ep.Method != "POST" {
		t.Errorf("Method = %q, want %q", ep.Method, "POST")
	}
	if ep.URL != "/api/users" {
		t.Errorf("URL = %q, want %q", ep.URL, "/api/users")
	}
	if ep.Handler != "AutoHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "AutoHandler")
	}
	if ep.RequestRef == nil {
		t.Fatal("RequestRef is nil")
	}
	if ep.RequestRef.Name != "CreateUserReq" {
		t.Errorf("RequestRef.Name = %q, want %q", ep.RequestRef.Name, "CreateUserReq")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "CreateUserRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "CreateUserRes")
	}
}

func TestParseFile_BareHttpGet_ResHandler(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpGet("/api/health", ResHandler(
		func(inb *Inbound) (HealthRes, error) { return HealthRes{}, nil },
	))
}`)
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.URL != "/api/health" {
		t.Errorf("URL = %q, want %q", ep.URL, "/api/health")
	}
	if ep.Handler != "ResHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "ResHandler")
	}
	if ep.RequestRef != nil {
		t.Error("RequestRef should be nil for ResHandler")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef is nil")
	}
	if ep.ResponseRef.Name != "HealthRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "HealthRes")
	}
}

func TestParseFile_BareHttpPut_Chained(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpPut("/api/user/:id", AutoHandler(
		func(inb *Inbound, req UpdateUserReq) (any, error) { return nil, nil },
	)).Desc("Update user").Protected().Resource("user:update")
}`)
	if ep.Method != "PUT" {
		t.Errorf("Method = %q, want %q", ep.Method, "PUT")
	}
	if ep.URL != "/api/user/:id" {
		t.Errorf("URL = %q, want %q", ep.URL, "/api/user/:id")
	}
	if ep.Handler != "AutoHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "AutoHandler")
	}
	if ep.Desc != "Update user" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "Update user")
	}
	if ep.Scope != "PROTECTED" {
		t.Errorf("Scope = %q, want %q", ep.Scope, "PROTECTED")
	}
	if ep.Resource != "user:update" {
		t.Errorf("Resource = %q, want %q", ep.Resource, "user:update")
	}
}

func TestParseFile_BareHttpDelete_Direct(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpDelete("/api/user/:id", deleteUserHandler)
}`)
	if ep.Method != "DELETE" {
		t.Errorf("Method = %q, want %q", ep.Method, "DELETE")
	}
	if ep.URL != "/api/user/:id" {
		t.Errorf("URL = %q, want %q", ep.URL, "/api/user/:id")
	}
	if ep.Handler != "Direct" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "Direct")
	}
}

func TestParseFile_BareHttpGet_VariableURL(t *testing.T) {
	ep := parseSingle(t, `package miso
const MetricsRoute = "/metrics"
func init() {
	HttpGet(MetricsRoute,
		RawHandler(func(inb *Inbound) {}),
	)
}`)
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.URL != "/metrics" {
		t.Errorf("URL = %q, want %q", ep.URL, "/metrics")
	}
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
}

func TestParseFile_BareHttpAny(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpAny("/api/catchall", RawHandler(func(inb *Inbound) {}))
}`)
	if ep.Method != "ANY" {
		t.Errorf("Method = %q, want %q", ep.Method, "ANY")
	}
	if ep.URL != "/api/catchall" {
		t.Errorf("URL = %q, want %q", ep.URL, "/api/catchall")
	}
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
}

func TestParseFile_BareHttpGet_DocJsonResp(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpGet("/api/data", RawHandler(loadDataFunc)).DocJsonResp(DataRes{})
}`)
	if ep.Method != "GET" {
		t.Errorf("Method = %q, want %q", ep.Method, "GET")
	}
	if ep.URL != "/api/data" {
		t.Errorf("URL = %q, want %q", ep.URL, "/api/data")
	}
	if ep.Handler != "RawHandler" {
		t.Errorf("Handler = %q, want %q", ep.Handler, "RawHandler")
	}
	if ep.ResponseRef == nil {
		t.Fatal("ResponseRef should be set via DocJsonResp")
	}
	if ep.ResponseRef.Name != "DataRes" {
		t.Errorf("ResponseRef.Name = %q, want %q", ep.ResponseRef.Name, "DataRes")
	}
}

func TestParseFile_NoDoc(t *testing.T) {
	ep := parseSingle(t, `package test
import "github.com/curtisnewbie/miso/miso"
func init() {
	miso.HttpGet("/health", miso.RawHandler(func(inb *miso.Inbound) {})).
		Desc("Health check").
		NoDoc()
}`)
	if !ep.NoDoc {
		t.Error("Expected NoDoc to be true")
	}
	if ep.Desc != "Health check" {
		t.Errorf("Desc = %q, want %q", ep.Desc, "Health check")
	}
}

func TestParseFile_BareNoDoc(t *testing.T) {
	ep := parseSingle(t, `package miso
func init() {
	HttpGet("/health", RawHandler(func(inb *Inbound) {})).
		NoDoc()
}`)
	if !ep.NoDoc {
		t.Error("Expected NoDoc to be true")
	}
}
