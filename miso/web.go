package miso

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/trace"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/osutil"
	"github.com/curtisnewbie/miso/util/pair"
	"github.com/curtisnewbie/miso/util/rfutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

const (
	ExtraName         = "miso-Name"
	ExtraDesc         = "miso-Desc"
	ExtraScope        = "miso-Scope"
	ExtraResource     = "miso-Resource"
	ExtraQueryParam   = "miso-QueryParam"
	ExtraHeaderParam  = "miso-HeaderParam"
	ExtraJsonRequest  = "miso-JsonRequest"
	ExtraJsonResponse = "miso-JsonResponse"
	ExtraNgTable      = "miso-NgTable"

	ScopePublic    = "PUBLIC"
	ScopeProtected = "PROTECTED"

	TagQueryParam  = "form"
	TagHeaderParam = "header"
)

var (
	flightRecorderOnce = sync.OnceValue(func() *flightRecorder { return newFlightRecorder("trace.out") })

	beforeRouteRegister = slutil.NewSyncSlice[func(Rail) error](2)

	interceptors []func(c *gin.Context, next func())

	lazyRouteRegistars []*LazyRouteDecl
	routeRegistars     []routesRegistar

	serverHttpRoutes []HttpRoute
	ginPreProcessors []GinPreProcessor

	// all http methods
	anyHttpMethods = []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect,
		http.MethodTrace,
	}

	resultBodyBuilder = ResultBodyBuilder{
		ErrJsonBuilder:     func(rail Rail, url string, err error) any { return WrapResp(rail, nil, err, url) },
		PayloadJsonBuilder: func(payload any) any { return OkRespWData(payload) },
		OkJsonBuilder:      func() any { return OkResp() },
	}

	endpointResultHandler = func(c *gin.Context, rail Rail, payload any, err error) {
		if err != nil {
			dispatchJson(c, resultBodyBuilder.ErrJsonBuilder(rail, c.Request.RequestURI, err))
			return
		}
		if payload != nil {
			dispatchJson(c, resultBodyBuilder.PayloadJsonBuilder(payload))
			return
		}
		dispatchJson(c, resultBodyBuilder.OkJsonBuilder())
	}

	// pprof / trace endpoint register disabled
	pprofRegisterDisabled = false

	noRouteHandler = func(ctx *gin.Context, rail Rail) {
		ctx.AbortWithStatus(404)
	}

	// default health check handler disabled
	defaultHealthCheckHandlerDisabled = false
)

type ParamDoc struct {
	Name string
	Desc string
}

type HttpRoute struct {
	Url         string           // http request url.
	Method      string           // http method.
	Extra       map[string][]any // extra metadata kv store.
	Desc        string           // description of the route (metadata).
	Scope       string           // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource    string           // the documented resource that the route should be bound to (metadata).
	Headers     []ParamDoc       // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams []ParamDoc       // the documented query parameters that will used by the endpoint (metadata).
}

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap HTTP Server",
		Bootstrap: webServerBootstrap,
		Condition: webServerBootstrapCondition,
		Order:     BootstrapOrderL3,
	})
	BeforeWebRouteRegister(func(rail Rail) error {
		prepAuthInterceptors(rail)
		prepDebugRoutes(rail)
		prepHealthcheckRoutes()
		prepApiDocRoutes(rail)
		return nil
	})
}

type ResultBodyBuilder struct {
	// wrap error in json, the returned object will be serialized to json.
	ErrJsonBuilder func(rail Rail, url string, err error) any

	// wrap payload object, the returned object will be serialized to json.
	PayloadJsonBuilder func(payload any) any

	// build empty ok response object, the returned object will be serialized to json.
	OkJsonBuilder func() any
}

// Replace the default ResultBodyBuilder
func SetResultBodyBuilder(rbb ResultBodyBuilder) error {
	resultBodyBuilder = rbb
	return nil
}

// Record server route
func recordHttpServerRoute(url string, method string, extra ...pair.Pair[string, any]) {
	extras := pair.MergeStrPairs(extra...)
	r := HttpRoute{
		Url:    url,
		Method: method,
		Extra:  extras,
	}
	if l, ok := extras[ExtraResource]; ok && len(l) > 0 {
		if v, ok := l[0].(string); ok {
			r.Resource = v
		}
	}
	if l, ok := extras[ExtraScope]; ok && len(l) > 0 {
		if v, ok := l[0].(string); ok {
			r.Scope = v
		}
	}
	if l, ok := extras[ExtraDesc]; ok && len(l) > 0 {
		if v, ok := l[0].(string); ok {
			r.Desc = v
		}
	}
	if l, ok := extras[ExtraQueryParam]; ok && len(l) > 0 {
		for _, p := range l {
			if v, ok := p.(ParamDoc); ok {
				r.QueryParams = append(r.QueryParams, v)
			}
		}
	}
	if l, ok := extras[ExtraHeaderParam]; ok && len(l) > 0 {
		for _, p := range l {
			if v, ok := p.(ParamDoc); ok {
				r.Headers = append(r.Headers, v)
			}
		}
	}
	serverHttpRoutes = append(serverHttpRoutes, r)
}

// Get recorded http server routes
func GetHttpRoutes() []HttpRoute {
	return serverHttpRoutes
}

type routesRegistar func(*gin.Engine)

func addRoutesRegistar(reg routesRegistar) {
	routeRegistars = append(routeRegistars, reg)
}

// Register GIN route for consul healthcheck
func prepHealthcheckRoutes() {
	if defaultHealthCheckHandlerDisabled {
		return
	}

	url := GetPropStr(PropHealthCheckUrl)
	if !strutil.IsBlankStr(url) {
		HttpGet(url, RawHandler(DefaultHealthCheckInbound))
	}
}

func startHttpServer(rail Rail, router http.Handler) error {
	addr := fmt.Sprintf("%s:%s", GetPropStr(PropServerHost), GetPropStr(PropServerPort))
	return startNetHttpServer(rail, addr, router)
}

func startNetHttpServer(rail Rail, addr string, router http.Handler) error {
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	la := ln.Addr().(*net.TCPAddr)
	rail.Infof("Serving HTTP on %s (actual port: %d)", server.Addr, la.Port)
	SetProp(PropServerActualPort, la.Port)

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			panic(fmt.Errorf("http.Server Serve: %s", err))
		}
	}()

	AddAsyncShutdownHook(func() { shutdownHttpServer(server) })
	return nil
}

// Register http routes on gin.Engine
func registerServerRoutes(rail Rail, engine *gin.Engine) error {
	if err := beforeRouteRegister.ForEachErr(func(t func(Rail) error) (stop bool, err error) {
		return false, t(rail)
	}); err != nil {
		return err
	}

	// no route
	engine.NoRoute(func(ctx *gin.Context) {
		rail := BuildRail(ctx)
		rail.Warnf("NoRoute for %s '%s'", ctx.Request.Method, ctx.Request.RequestURI)
		noRouteHandler(ctx, rail)
	})

	// register custom routes
	for _, registerRoute := range routeRegistars {
		registerRoute(engine)
	}
	routeRegistars = nil

	for _, lrr := range lazyRouteRegistars {
		lrr.build(engine)
	}
	lazyRouteRegistars = nil

	logRoutes := GetPropBool(PropServerLogRoutes)
	if IsDebugLevel() || logRoutes {
		for _, r := range engine.Routes() {
			if logRoutes {
				rail.Infof("%-6s %s", r.Method, r.Path)
			} else {
				rail.Debugf("%-6s %s", r.Method, r.Path)
			}
		}
	}
	return nil
}

func shutdownHttpServer(server *http.Server) {
	Info("Shutting down http server")
	defer Infof("Http server exited")

	timeout := GetPropInt(PropServerGracefulShutdownTimeSec)
	if timeout > 0 {
		dur := (time.Duration(timeout) / 2)
		if dur > 0 {
			// http server also has a timeout to avoid blocking the graceful shutdown period the whole time.
			c, cancel := context.WithTimeout(context.Background(), dur*time.Second)
			defer cancel()
			server.Shutdown(c)
			return
		}
	}

	server.Shutdown(context.Background())
}

// Default Recovery func
func DefaultRecovery(c *gin.Context, e interface{}) {
	rail := BuildRail(c)
	rail.Errorf("%v '%v' Recovered from panic, %v", c.Request.Method, c.Request.RequestURI, e)

	// response already written, avoid writting it again.
	if c.Writer.Written() {
		if me, ok := e.(*MisoErr); ok {
			rail.Infof("Miso error, code: '%v', msg: '%v', internalMsg: '%v'", me.Code(), me.Msg(), me.InternalMsg())
			return
		}
		rail.Errorf("Unknown error, %v", e)
		return
	}

	if err, ok := e.(error); ok {
		endpointResultHandler(c, rail, nil, err)
		return
	}

	endpointResultHandler(c, rail, nil, errs.NewErrf("Unknown error, please try again later"))
}

// Tracing Middleware
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// propagate tracing key/value pairs with context
		ctx := c.Request.Context()

		UsePropagationKeys(func(k string) {
			if h := c.GetHeader(k); h != "" {
				ctx = context.WithValue(ctx, k, h) //lint:ignore SA1029 keys must be exposed to retrieve the values
			}
		})

		// replace the context
		c.Request = c.Request.WithContext(ctx)

		// follow the chain
		c.Next()
	}
}

// Build Rail from gin.Context.
//
// This func creates new Rail for the first time by setting up proper traceId and spanId.
//
// It can also recognize that a traceId (and spanId) was previously created, and do attempt to reuse these tracing values,
// such that the Rail acts as if it's the previous one, this is especially useful when we are recovering from a panic.
// In most cases, we should only call BuildRail for once.
//
// However, if the Rail has attempted to overwrite it's spanId (i.e., creating new span), this newly created spanId will not
// be reflected on the Rail created here. But this should be fine, because new span is usually created for async operation.
func BuildRail(c *gin.Context) Rail {
	if !GetPropBool(PropServerPropagateInboundTrace) {
		return EmptyRail()
	}

	if c.Keys == nil {
		c.Keys = map[string]any{}
	}

	ctx := c.Request.Context()

	// it's possible that the spanId and traceId have been created already
	// if we call BuildRail() for the second time, we should read from the *gin.Context
	// instead of creating new ones.
	// for the most of the time, we are using one single Rail throughout the method calls
	contextModified := false
	UsePropagationKeys(func(k string) {
		if v, ok := c.Keys[k]; ok && v != "" {
			// c.Keys -> c.Request.Context()
			ctx = context.WithValue(ctx, k, v) //lint:ignore SA1029 keys must be exposed for client to use
			contextModified = true             // the trace is not newly created, we are loading it from *gin.Context
		}
	})

	// create a new Rail
	rail := NewRail(ctx)

	// this is mainly used for panic recovery
	if !contextModified {
		UsePropagationKeys(func(k string) {
			// c.Request.Context() -> c.Keys
			// copy the newly created keys back to the gin.Context
			if v, ok := c.Keys[k]; !ok || v == "" {
				c.Keys[k] = rail.CtxValue(k)
			}
		})
	}

	return rail
}

// Traced and parameters mapped route handler.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param) or 'header' tag.
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
type MappedTRouteHandler[Req any, Res any] func(inb *Inbound, req Req) (Res, error)

// Build route handler with the mapped payload object, context, and logger.
//
// value and error returned by handler are automically wrapped in a Resp object
func newMappedTRouteHandler[Req any, Res any](handler MappedTRouteHandler[Req, Res]) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := BuildRail(c)

		// bind to payload boject
		var req Req
		mustBind(rail, c, &req)

		wtcbCnt := 0
		if GetPropBool(PropServerRequestValidateEnabled) {
			wtcbCnt += 2
		}
		if GetPropBool(PropServerRequestAutoMapHeader) {
			wtcbCnt += 1
		}
		if wtcbCnt > 0 {
			wtcb := make([]rfutil.WalkTagCallback, 0, wtcbCnt)

			// validate request
			if GetPropBool(PropServerRequestValidateEnabled) {
				wtcb = append(wtcb, ValidateWalkTagCallback, ValidateWalkTagCallbackDeprecated)
			}

			// for setting headers
			if GetPropBool(PropServerRequestAutoMapHeader) {
				wtcb = append(wtcb, reflectSetHeaderCallback(c))
			}

			if err := rfutil.WalkTagShallow(&req, wtcb...); err != nil {
				endpointResultHandler(c, rail, nil, err)
				return
			}
		}

		if GetPropBool(PropServerRequestLogEnabled) {
			rail.Infof("%v %v, req: %+v", c.Request.Method, c.Request.RequestURI, req)
		}

		// handle the requests
		res, err := handler(newInbound(c), req)

		// wrap result and error
		endpointResultHandler(c, rail, res, err)
	}
}

// Raw version of traced route handler.
type RawTRouteHandler func(inb *Inbound)

// Build route handler with context, and logger
func newRawTRouteHandler(handler RawTRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		handler(newInbound(c))
	}
}

// Traced route handler.
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With Res type declared, miso will automatically parse the Res type using reflect and generate an API documentation
// describing the endpoint.
type TRouteHandler[Res any] func(inb *Inbound) (Res, error)

// Build route handler with context, and logger
//
// value and error returned by handler are automically wrapped in a Resp object
func newTRouteHandler[Res any](handler TRouteHandler[Res]) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := BuildRail(c)
		r, e := handler(newInbound(c))
		endpointResultHandler(c, rail, r, e)
	}
}

// Handle endpoint's result using the configured EndpointResultHandler.
func HandleEndpointResult(inb Inbound, rail Rail, result any, err error) {
	c := inb.Engine().(*gin.Context)
	endpointResultHandler(c, rail, result, err)
}

// Must bind request payload to the given pointer, else panic
func mustBind(rail Rail, c *gin.Context, ptr any) {
	onFailed := func(err error) {
		rail.Warnf("Bind payload failed, %v", err)
		panic(errs.NewErrf("Illegal Arguments"))
	}

	// we now use jsoniter
	if c.Request.Method != http.MethodGet && c.ContentType() == gin.MIMEJSON {
		if err := json.DecodeJson(c.Request.Body, ptr); err != nil {
			onFailed(err)
		}
		return
	}

	// other mime types
	if err := c.ShouldBind(ptr); err != nil {
		onFailed(err)
	}
}

// Dispatch a json response
func dispatchJsonCode(c *gin.Context, code int, body interface{}) {
	c.Status(code)
	c.Header("Content-Type", applicationJson)

	err := json.EncodeJson(c.Writer, body)
	if err != nil {
		panic(err)
	}
}

// Dispatch a json response
func dispatchJson(c *gin.Context, body interface{}) {
	dispatchJsonCode(c, http.StatusOK, body)
}

func webServerBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropServerEnabled) && !GetPropBool(PropAppTestEnv), nil
}

func webServerBootstrap(rail Rail) error {
	rail.Info("Starting HTTP server")

	// always set to releaseMode
	gin.SetMode(gin.ReleaseMode)
	if GetPropBool(PropServerGinValidationDisabled) {
		rail.Debug("Disabled Gin's builtin validation")
		gin.DisableBindValidation()
	}

	// gin engine
	engine := gin.New()
	engine.Use(TraceMiddleware())

	if !IsProdMode() && IsDebugLevel() {
		engine.Use(gin.Logger()) // gin's default logger for debugging
	}

	if GetPropBool(PropServerPerfEnabled) {
		engine.Use(PerfMiddleware())
	}

	for _, p := range ginPreProcessors {
		p(rail, engine)
	}
	ginPreProcessors = nil

	// register customer recovery func
	engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

	// register http routes
	if err := registerServerRoutes(rail, engine); err != nil {
		return err
	}

	// start the http server
	return startHttpServer(rail, engine)
}

type TreePath interface {
	Prepend(baseUrl string)
}

// Lazy route declaration
type LazyRouteDecl struct {
	Url     string
	Method  string
	Handler func(c *gin.Context)

	RegisterFunc func(extra ...pair.Pair[string, any])
	Extras       []pair.Pair[string, any]
}

// Build endpoint.
func (g *LazyRouteDecl) build(engine *gin.Engine) {
	recordHttpServerRoute(g.Url, g.Method, g.Extras...)
	engine.Handle(g.Method, g.Url, g.Handler)
}

func (g *LazyRouteDecl) Prepend(baseUrl string) {
	g.Url = baseUrl + g.Url
}

// Add endpoint description (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) Desc(desc string) *LazyRouteDecl {
	// v := strings.TrimSpace(regexp.MustCompile(`[\n]+`).ReplaceAllString(desc, "\n"))
	// v = strings.TrimSpace(regexp.MustCompile(`[\t ]+`).ReplaceAllString(v, " "))
	// return g.Extra(ExtraDesc, v)
	return g.Extra(ExtraDesc, strings.TrimSpace(regexp.MustCompile(`[\n\t ]+`).ReplaceAllString(desc, " ")))
}

// Mark endpoint publicly accessible (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) Public() *LazyRouteDecl {
	return g.Extra(ExtraScope, ScopePublic)
}

// Document the access scope of the endpoint (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) Scope(scope string) *LazyRouteDecl {
	return g.Extra(ExtraScope, scope)
}

// Documents that the endpoint requires protection (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) Protected() *LazyRouteDecl {
	return g.Extra(ExtraScope, ScopeProtected)
}

// Record the resource that the endppoint should be bound to (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) Resource(resource string) *LazyRouteDecl {
	return g.Extra(ExtraResource, strings.TrimSpace(resource))
}

// Add extra info to endpoint's metadata.
func (g *LazyRouteDecl) Extra(key string, value any) *LazyRouteDecl {
	return g.extra(key, value, nil)
}

type extraMatchCond = func(key string, val any, ex pair.Pair[string, any]) (overwrite bool, breakLoop bool)

func (g *LazyRouteDecl) extra(key string, value any, cond extraMatchCond) *LazyRouteDecl {
	if cond == nil {
		g.Extras = append(g.Extras, pair.New(key, value))
	} else {
		for i, ex := range g.Extras {
			overwrite, breakLoop := cond(key, value, ex)

			if overwrite {
				ex.Right = value
				g.Extras[i] = ex
			}

			if breakLoop {
				return g
			}
		}
		g.Extras = append(g.Extras, pair.New(key, value))
	}
	return g
}

// Document query parameter that the endpoint will use (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocQueryParam(queryName string, desc string) *LazyRouteDecl {
	return g.extra(ExtraQueryParam, ParamDoc{queryName, desc}, extraFilterOneParamDocByName())
}

// Document header parameter that the endpoint will use (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocHeader(headerName string, desc string) *LazyRouteDecl {
	return g.extra(ExtraHeaderParam, ParamDoc{headerName, desc}, extraFilterOneParamDocByName())
}

// Document header parameters that the endpoint expects (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocHeaderReq(v any) *LazyRouteDecl {
	t := reflect.TypeOf(v)
	for _, pd := range parseHeaderDoc(t) {
		g.extra(ExtraHeaderParam, pd, extraFilterOneParamDocByName())
	}
	return g
}

// Document query parameters that the endpoint expects (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocQueryReq(v any) *LazyRouteDecl {
	t := reflect.TypeOf(v)
	for _, pd := range parseQueryDoc(t) {
		g.extra(ExtraQueryParam, pd, extraFilterOneParamDocByName())
	}
	return g
}

// Document json request that the endpoint expects (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocJsonReq(v any) *LazyRouteDecl {
	// json request could contain fields that are mapped using header or query param
	return g.DocQueryReq(v).DocHeaderReq(v).extra(ExtraJsonRequest, v, extraFilterOneByKey())
}

// Document json response that the endpoint returns (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocJsonResp(v any) *LazyRouteDecl {
	return g.extra(ExtraJsonResponse, v, extraFilterOneByKey())
}

func extraFilterOneByKey() extraMatchCond {
	return func(key string, val any, ex pair.Pair[string, any]) (overwrite bool, breakLoop bool) {
		if key == ex.Left {
			return true, true
		}
		return false, false
	}
}

func extraFilterOneParamDocByName() extraMatchCond {
	return func(key string, val any, ex pair.Pair[string, any]) (overwrite bool, breakLoop bool) {
		vd := val.(ParamDoc)

		if key != ex.Left {
			return false, false
		}

		// unique ParamDoc.Name
		if pd, ok := ex.Right.(ParamDoc); ok && pd.Name == vd.Name {
			if vd.Desc != "" { // always pick the one with desc
				return true, true
			}
			return false, true
		}

		return false, false
	}
}

func newLazyRouteDecl(url string, method string, handler func(c *gin.Context)) *LazyRouteDecl {
	dec := &LazyRouteDecl{
		Url:     url,
		Method:  method,
		Handler: interceptedHandler(handler),
		Extras:  []pair.Pair[string, any]{},
	}
	lazyRouteRegistars = append(lazyRouteRegistars, dec)
	return dec
}

type RoutingGroup struct {
	Base  string
	Paths []TreePath
}

// Group routes, routes are immediately registered
func (rg *RoutingGroup) Group(grouped ...TreePath) *RoutingGroup {
	if rg.Paths == nil {
		rg.Paths = make([]TreePath, 0, len(grouped))
	}
	for _, r := range grouped {
		rg.Paths = append(rg.Paths, r)
		r.Prepend(rg.Base)
	}
	return rg
}

func (rg *RoutingGroup) Prepend(baseUrl string) {
	for _, r := range rg.Paths {
		r.Prepend(baseUrl)
	}
}

// Group routes together to share the same base url.
func GroupRoute(baseUrl string, grouped ...TreePath) *RoutingGroup {
	return BaseRoute(baseUrl).Group(grouped...)
}

// Group routes together to share the same base url.
func BaseRoute(baseUrl string) *RoutingGroup {
	return &RoutingGroup{Base: baseUrl}
}

// Disable pprof debug endpoint handler.
func DisablePProfEndpointRegister() {
	pprofRegisterDisabled = true
}

// Preprocessor of *gin.Engine.
type GinPreProcessor func(rail Rail, engine *gin.Engine)

// Process *gin.Engine before the web server starts, particularly useful when trying to add middleware.
func PreProcessGin(preProcessor GinPreProcessor) {
	ginPreProcessors = append(ginPreProcessors, preProcessor)
}

func reflectSetHeaderCallback(c *gin.Context) rfutil.WalkTagCallback {
	return walkHeaderTagCallback(func(k string) string { return c.GetHeader(k) })
}

func reflectSetHeaderValue(fieldVal reflect.Value, fieldType reflect.Type, header string) {
	switch fieldType.Kind() {
	case reflect.String:
		fieldVal.SetString(header)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		vv := cast.ToInt64(header)
		fieldVal.SetInt(vv)
	case reflect.Float32, reflect.Float64:
		vv := cast.ToFloat64(header)
		fieldVal.SetFloat(vv)
	case reflect.Bool:
		vv := cast.ToBool(header)
		fieldVal.SetBool(vv)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		vv := cast.ToUint64(header)
		fieldVal.SetUint(vv)
	case reflect.Pointer:
		ptrType := fieldType.Elem()
		reflectSetHeaderValue(fieldVal, ptrType, header)
	}
}

func walkHeaderTagCallback(getHeader func(k string) string) rfutil.WalkTagCallback {
	return rfutil.WalkTagCallback{
		Tag: TagHeaderParam,
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			hv := getHeader(tagVal)
			if hv == "" {
				return nil
			}
			reflectSetHeaderValue(fieldVal, fieldType.Type, hv)
			return nil
		},
	}
}

// type alias of Rail, just to avoid name conflict, *Inbound has Rail(), I don't want to break the code :(
type erail = Rail

// Inbound request context.
//
// Inbound hides the underlying engine (e.g., *gin.Context) using .Engine() method.
// In most cases, you should not attempt to cast the engine explictly, it's possible that
// miso will replace the engine in future release.
//
// However, you should be able to satisfy most of your need by calling .Unwrap(),
// that returns the underlying http.ResponseWriter, *http.Request.
//
// Use miso.Rail for tracing (not just logs), pass it around your application code and the code
// calling miso's methods course.
type Inbound struct {
	erail
	engine  *gin.Context
	w       http.ResponseWriter
	r       *http.Request
	queries url.Values
}

func newInbound(c *gin.Context) *Inbound {
	return &Inbound{
		erail:  BuildRail(c),
		engine: c,
		w:      c.Writer,
		r:      c.Request,
	}
}

func (i *Inbound) Engine() any {
	return i.engine
}

func (i *Inbound) Unwrap() (http.ResponseWriter, *http.Request) {
	return i.w, i.r
}

func (i *Inbound) Writer() http.ResponseWriter {
	return i.w
}

func (i *Inbound) Request() *http.Request {
	return i.r
}

func (i *Inbound) WriteSSE(name string, message any) {
	if tmp, err := json.SWriteJson(message); err == nil {
		message = tmp // we serialize json ourselves for consistency
	}
	i.engine.SSEvent(name, message)
}

func (i *Inbound) Rail() Rail {
	return i.erail
}

/*
Handle the result using universally configured endpoint result handler.

The result or error is written back to the client. In most cases, caller must exit the handler
after calling this method.

E.g.,

	miso.HttpGet("/dir/info", miso.RawHandler(func(inb *miso.Inbound) {
		// ... do something

		if err != nil {
			inb.HandleResult(nil, err) // something goes wrong
			return
		}

		// return result back to the client
		inb.HandleResult(result, err)
	}))
*/
func (i *Inbound) HandleResult(result any, err error) {
	HandleEndpointResult(*i, i.Rail(), result, err)
}

func (i *Inbound) Status(status int) {
	i.w.WriteHeader(status)
}

func (i *Inbound) Query(k string) string {
	if i.queries != nil {
		return i.queries.Get(k)
	}
	i.queries = i.r.URL.Query()
	return i.queries.Get(k)
}

func (i *Inbound) Queries() url.Values {
	if i.queries != nil {
		return i.queries
	}
	i.queries = i.r.URL.Query()
	return i.queries
}

func (i *Inbound) Header(k string) string {
	return i.r.Header.Get(k)
}

func (i *Inbound) Headers() http.Header {
	return i.r.Header
}

func (i *Inbound) SetHeader(k string, v string) {
	i.r.Header.Set(k, v)
}

func (i *Inbound) AddHeader(k string, v string) {
	i.r.Header.Add(k, v)
}

func (i *Inbound) MustBind(ptr any) {
	mustBind(i.Rail(), i.engine, ptr)
}

func (i *Inbound) ReadRawBytes() ([]byte, error) {
	_, r := i.Unwrap()
	by, err := io.ReadAll(r.Body)
	return by, errs.Wrap(err)
}

func (i *Inbound) WriteJson(v any) {
	i.SetHeader("Content-Type", applicationJson)
	w, _ := i.Unwrap()
	if err := json.EncodeJson(w, v); err != nil {
		panic(err)
	}
}

func (i *Inbound) WriteString(v string) {
	i.SetHeader("Content-Type", textPlain)
	w, _ := i.Unwrap()
	if _, err := w.Write([]byte(v)); err != nil {
		panic(err)
	}
}

func (i *Inbound) WriteJsonStatus(v any, httpStatus int) {
	i.Status(httpStatus)
	i.WriteJson(v)
}

func (i *Inbound) LogRequest() {
	rail := i.Rail()
	_, r := i.Unwrap()
	rail.Infof("Receive '%v %v' request from %v", r.Method, r.RequestURI, r.RemoteAddr)
	rail.Infof("Content-Length: %v", r.ContentLength)

	var bodystr string
	if r.Body != nil {
		body, e := io.ReadAll(r.Body)
		if e != nil {
			rail.Errorf("Failed to read request body, %v", e)
			i.Status(http.StatusInternalServerError)
			return
		}
		bodystr = string(body)
	}
	rail.Info("Headers: ")
	for k, v := range r.Header {
		if strutil.ContainsAnyStrIgnoreCase(k, "authorization", "cookie", "token") {
			v = []string{"***"}
		}
		rail.Infof("  %-30s: %v", k, v)
	}

	rail.Info("")
	rail.Info("Body: ")
	rail.Infof("  %s", bodystr)
	rail.Info("")
}

func setNoRouteHandler(f func(ctx *gin.Context, rail Rail)) {
	noRouteHandler = f
}

// Enable Basic authorization globally for all registered endpoints.
func EnableBasicAuth(f func(username string, password string, url string, method string) bool) {
	PreProcessGin(func(rail Rail, engine *gin.Engine) {
		engine.Use(func(ctx *gin.Context) {
			url := ctx.Request.RequestURI
			method := ctx.Request.Method
			u, p, _ := ctx.Request.BasicAuth()
			if f(u, p, url, method) {
				ctx.Next()
			} else {
				rail.Warnf("Rejected request '%s %s' from %v (remote_addr), basic auth invalid", method, url, ctx.Request.RemoteAddr)
				ctx.Writer.Header().Add("WWW-Authenticate", "Basic realm=\"Username and Password\"")
				ctx.AbortWithStatus(http.StatusUnauthorized)
			}
		})
	})
}

// Disable the default health check endpoint handler.
func DisableDefaultHealthCheckHandler() {
	defaultHealthCheckHandlerDisabled = true
}

// Add request interceptor.
//
// For requests to be processed, next func must be called, otherwise the request is dropped (i.e., rejected).
//
// For requests that are rejected, interceptor should set appropriate http status.
func AddInterceptor(f func(inb *gin.Context, next func())) {
	if f == nil {
		panic("interceptor is nil")
	}
	interceptors = append(interceptors, f)
}

type interceptor struct {
	idx          int
	c            *gin.Context
	interceptors []func(c *gin.Context, next func())
}

func (it *interceptor) next() {
	it.idx++
	if it.idx < len(it.interceptors) {
		it.interceptors[it.idx](it.c, it.next)
	}
}

func newInterceptor(c *gin.Context, handler func(c *gin.Context)) *interceptor {
	copy := slutil.SliceCopy(interceptors)
	return &interceptor{
		idx:          -1,
		c:            c,
		interceptors: append(copy, func(c *gin.Context, next func()) { handler(c) }),
	}
}

func interceptedHandler(f func(c *gin.Context)) func(c *gin.Context) {
	return func(c *gin.Context) {
		interceptors := newInterceptor(c, f)
		interceptors.next()
	}
}

func MatchPathPatternFunc(patterns ...string) func(method string, url string) bool {
	return func(method string, url string) bool {
		return strutil.MatchPathAny(patterns, url)
	}
}

// Deprecated: use [AddBearerAuthInterceptor] instead.
func AddBearerInterceptor(doIntercept func(method string, url string) bool, bearerToken func() string) {
	AddInterceptor(func(inb *gin.Context, next func()) {
		url := inb.Request.RequestURI
		method := inb.Request.Method

		if doIntercept(method, url) {
			bearer := bearerToken()
			r := inb.Request
			w := inb.Writer
			token, ok := ParseBearer(inb.GetHeader("Authorization"))
			if !ok || token != bearer {
				Debugf("Bearer authorization failed, missing bearer token or token mismatch, %v %v", r.Method, r.RequestURI)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		next()
	})
}

func AddBearerAuthInterceptor(doIntercept func(method string, url string) bool, validateBearerToken func(provided string) bool) {
	AddInterceptor(func(inb *gin.Context, next func()) {
		url := inb.Request.RequestURI
		method := inb.Request.Method

		if doIntercept(method, url) {
			r := inb.Request
			w := inb.Writer
			token, ok := ParseBearer(inb.GetHeader("Authorization"))
			if !ok || !validateBearerToken(token) {
				Debugf("Bearer authorization failed, missing bearer token or token mismatch, %v %v", r.Method, r.RequestURI)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		next()
	})
}

func AddCorsAny() {
	PreProcessGin(func(rail Rail, engine *gin.Engine) {
		engine.Use(func(c *gin.Context) {
			h := c.Writer.Header()
			h.Set("Access-Control-Allow-Origin", "*")
			h.Set("Access-Control-Allow-Credentials", "false")
			h.Set("Access-Control-Allow-Headers", "*")
			h.Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, HEAD")
			h.Set("Access-Control-Max-Age", "3600")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})
	})
}

type rawHandler struct {
	handleFunc func(c *gin.Context)
}

func (a *rawHandler) handle(c *gin.Context) {
	a.handleFunc(c)
}

type resAutoHandler struct {
	handleFunc func(c *gin.Context)
	resVar     any
}

func (a *resAutoHandler) handle(c *gin.Context) {
	a.handleFunc(c)
}

func (a *resAutoHandler) res() any {
	return a.resVar
}

type autoHandler struct {
	handleFunc func(c *gin.Context)
	reqVar     any
	resVar     any
}

func (a *autoHandler) handle(c *gin.Context) {
	a.handleFunc(c)
}

func (a *autoHandler) req() any {
	return a.reqVar
}

func (a *autoHandler) res() any {
	return a.resVar
}

type httpHandler = interface {
	handle(c *gin.Context)
}

type reqAwareHandler = interface {
	req() any
}

type resAwareHandler = interface {
	res() any
}

func HttpAny(url string, handler RawTRouteHandler, extra ...pair.Pair[string, any]) {
	for _, method := range anyHttpMethods {
		recordHttpServerRoute(url, method, extra...)
	}
	addRoutesRegistar(func(e *gin.Engine) { e.Any(url, newRawTRouteHandler(handler)) })
}

func HttpGet(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodGet, url, handler)
}

func HttpHead(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodHead, url, handler)
}

func HttpPost(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodPost, url, handler)
}

func HttpPut(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodPut, url, handler)
}

func HttpPatch(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodPatch, url, handler)
}

func HttpDelete(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodDelete, url, handler)
}

func HttpConnect(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodConnect, url, handler)
}

func HttpOptions(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodOptions, url, handler)
}

func HttpTrace(url string, handler httpHandler) *LazyRouteDecl {
	return handleHttp(http.MethodTrace, url, handler)
}

func handleHttp(method string, url string, handler httpHandler) *LazyRouteDecl {
	decl := newLazyRouteDecl(url, method, handler.handle)
	return setAwareHandler(decl, handler)
}

func setAwareHandler(decl *LazyRouteDecl, handler any) *LazyRouteDecl {
	if v, ok := handler.(reqAwareHandler); ok {
		r := v.req()
		switch decl.Method {
		case http.MethodPut, http.MethodPost:
			decl = decl.DocJsonReq(r)
		default:
			decl = decl.DocQueryReq(r).
				DocHeaderReq(r)
		}

	}
	if v, ok := handler.(resAwareHandler); ok {
		decl = decl.DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(v.res()))
	}
	return decl
}

// Create HTTP handler that automatically resolve Request and Response data.
//
// Req type should be a struct (or a pointer to a struct), where all fields are automatically mapped from the request
// using 'json' tag, 'form' tag (for form-data or query param) or 'header' tag.
//
// Both Res value and error (if not nil) are be wrapped inside miso.Resp and serialized as json.
// This behaviour can be custmized using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func AutoHandler[Req any, Res any](handler MappedTRouteHandler[Req, Res]) httpHandler {
	req := rfutil.NewVar[Req]()
	res := rfutil.NewVar[Res]()
	return &autoHandler{
		handleFunc: newMappedTRouteHandler(handler),
		reqVar:     req,
		resVar:     res,
	}
}

// Create raw HTTP Handler.
//
// Request and Response are handled by the handler itself.
func RawHandler(handler RawTRouteHandler) httpHandler {
	return &rawHandler{
		handleFunc: newRawTRouteHandler(handler),
	}
}

// Create HTTP handler that automatically resolve Res.
//
// Both Res value and error (if not nil) are be wrapped inside miso.Resp and serialized as json.
// This behaviour can be custmized using miso.SetResultBodyBuilder func.
//
// With Res type declared, miso will automatically parse these the type using reflect
// and generate an API documentation describing the endpoint.
func ResHandler[Res any](handler TRouteHandler[Res]) httpHandler {
	res := rfutil.NewVar[Res]()
	return &resAutoHandler{
		handleFunc: newTRouteHandler(handler),
		resVar:     res,
	}
}

func BeforeWebRouteRegister(f ...func(Rail) error) {
	beforeRouteRegister.Append(f...)
}

type flightRecorder struct {
	fr  trace.FlightRecorder
	mu  *sync.Mutex
	out string
}

func (f *flightRecorder) Start(dur time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.fr.Enabled() {
		return ErrIllegalArgument.WithMsg("FlightRecorder is currently running")
	}

	if err := f.fr.Start(); err != nil {
		return err
	}

	Infof("FlightRecorder started, dur: %v", dur)

	go func() {
		<-time.After(dur)
		f.Stop()
	}()

	return nil
}

func (f *flightRecorder) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.fr.Enabled() {
		return nil
	}

	fi, err := osutil.OpenRWFile(f.out, true)
	if err != nil {
		return err
	}

	_ = fi.Truncate(0)

	if _, err := f.fr.WriteTo(fi); err != nil {
		return err
	}

	if err := f.fr.Stop(); err != nil {
		return err
	}

	Infof("FlightRecorder stopped, output written to %v", f.out)
	return nil
}

func newFlightRecorder(out string) *flightRecorder {
	return &flightRecorder{
		fr:  *trace.NewFlightRecorder(),
		mu:  &sync.Mutex{},
		out: out,
	}
}

func prepDebugRoutes(rail Rail) {
	if !pprofRegisterDisabled && (!IsProdMode() || GetPropBool(PropServerPprofEnabled)) {
		GroupRoute("/debug/pprof",
			HttpGet("", RawHandler(func(inb *Inbound) { pprof.Index(inb.Unwrap()) })),
			HttpGet("/:name", RawHandler(func(inb *Inbound) { pprof.Index(inb.Unwrap()) })),
			HttpGet("/cmdline", RawHandler(func(inb *Inbound) { pprof.Cmdline(inb.Unwrap()) })),
			HttpGet("/profile", RawHandler(func(inb *Inbound) { pprof.Profile(inb.Unwrap()) })),
			HttpGet("/symbol", RawHandler(func(inb *Inbound) { pprof.Symbol(inb.Unwrap()) })),
			HttpGet("/trace", RawHandler(func(inb *Inbound) { pprof.Trace(inb.Unwrap()) })),
		)
		rail.Infof("Registered /debug/pprof APIs for debugging")

		HttpGet("/debug/trace/recorder/run", RawHandler(HandleFlightRecorderRun)).
			DocQueryParam("duration", "Duration of the flight recording. Required. Duration cannot exceed 30 min.").
			Desc("Start FlightRecorder. Recorded result is written to trace.out when it's finished or stopped.")

		HttpGet("/debug/trace/recorder/stop", RawHandler(HandleFlightRecorderStop)).
			Desc("Stop existing FlightRecorder session.")

		rail.Infof("Registered /debug/trace APIs for debugging")

		if GetPropStrTrimmed(PropServerAuthBearer) != "" { // server.auth.bearer is already set for all apis
			rail.Infof("Using configuration '%v' in authentication interceptor for pprof & trace APIs", PropServerAuthBearer)
		} else {
			// we have set auth bearer for pprof apis specifically
			if GetPropStrTrimmed(PropServerPprofAuthBearer) != "" {
				AddBearerAuthInterceptor(
					MatchPathPatternFunc("/debug/pprof/**", "/debug/trace/**"),
					func(tok string) bool {
						v := GetPropStrTrimmed(PropServerPprofAuthBearer) // prop value may change while it's runs
						return v == "" || v == tok
					},
				)
				rail.Infof("Using configuration '%v' in authentication interceptor for pprof & trace APIs", PropServerPprofAuthBearer)

			} else if IsProdMode() { // in prod mode, print warning
				rail.Warnf("pprof authentication is not enabled in production mode, pprof & trace APIs are not protected")
			} else {
				// pprof apis not protected, but we are not in prod mode either
			}
		}
	}
}

func prepApiDocRoutes(rail Rail) {
	if err := serveApiDocTmpl(rail); err != nil {
		rail.Errorf("failed to server apidoc, %v", err)
	}
}

func prepAuthInterceptors(rail Rail) {
	if GetPropStrTrimmed(PropServerAuthBearer) != "" {
		AddBearerAuthInterceptor(
			func(method, url string) bool { return true },
			func(tok string) bool {
				v := GetPropStrTrimmed(PropServerAuthBearer) // prop value may change while it's runs
				return v == "" || v == tok
			},
		)
		rail.Infof("Registered bearer authentication interceptor for all APIs")
	}
}

func HandleFlightRecorderRun(inb *Inbound) {
	fr := flightRecorderOnce()
	dur, err := time.ParseDuration(strings.TrimSpace(inb.Query("duration")))
	if err != nil {
		inb.HandleResult(nil, errs.Wrapf(err, "Invalid duration expression"))
		return
	}
	if dur >= 30*time.Minute { // just in case
		inb.HandleResult(nil, ErrIllegalArgument.WithMsg("Flight recording cannot proceed for over 30 min"))
		return
	}
	if err := fr.Start(dur); err != nil {
		inb.HandleResult(nil, err)
		return
	}
}

func HandleFlightRecorderStop(inb *Inbound) {
	fr := flightRecorderOnce()
	err := fr.Stop()
	inb.HandleResult(nil, err)
}
