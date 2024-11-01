package miso

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/version"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

const (
	// Components like database that are essential and must be ready before anything else.
	BootstrapOrderL1 = -20

	// Components that are bootstraped before the web server, such as metrics stuff.
	BootstrapOrderL2 = -15

	// The web server or anything similar, bootstraping web server doesn't really mean that we will receive inbound requests.
	BootstrapOrderL3 = -10

	// Components that introduce inbound requests or job scheduling.
	//
	// When these components bootstrap, the server is considered truly running.
	// For example, service registration (for service discovery), MQ broker connection and so on.
	BootstrapOrderL4 = -5

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

	defShutdownOrder = 5

	TagQueryParam  = "form"
	TagHeaderParam = "header"
)

var (
	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	lazyRouteRegistars []*LazyRouteDecl
	routeRegistars     []routesRegistar

	serverHttpRoutes []HttpRoute
	ginPreProcessors []GinPreProcessor

	shuttingDown   bool         = false
	shutingDownRwm sync.RWMutex // rwmutex for shuttingDown

	shutdownHook []OrderedShutdownHook
	shmu         sync.Mutex // mutex for shutdownHook

	serverBootrapCallbacks      []ComponentBootstrap
	preServerBootstrapListener  []func(r Rail) error
	postServerBootstrapListener []func(r Rail) error

	// all http methods
	anyHttpMethods = []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect,
		http.MethodTrace,
	}

	// channel for signaling server shutdown
	manualSigQuit = make(chan int, 1)

	resultBodyBuilder = ResultBodyBuilder{
		ErrJsonBuilder:     func(rail Rail, url string, err error) any { return WrapResp(rail, nil, err, url) },
		PayloadJsonBuilder: func(payload any) any { return OkRespWData(payload) },
		OkJsonBuilder:      func() any { return OkResp() },
	}

	endpointResultHandler = func(c *gin.Context, rail Rail, payload any, err error) {
		if err != nil {
			DispatchJson(c, resultBodyBuilder.ErrJsonBuilder(rail, c.Request.RequestURI, err))
			return
		}
		if payload != nil {
			DispatchJson(c, resultBodyBuilder.PayloadJsonBuilder(payload))
			return
		}
		DispatchJson(c, resultBodyBuilder.OkJsonBuilder())
	}

	// pprof endpoint register disabled
	pprofRegisterDisabled = false
	noRouteHandler        = func(ctx *gin.Context, rail Rail) {
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

type ComponentBootstrap struct {
	// name of the component.
	Name string
	// the actual bootstrap function.
	Bootstrap func(rail Rail) error
	// check whether component should be bootstraped
	Condition func(rail Rail) (bool, error)
	// order of which the components are bootstraped, natural order, it's by default 15.
	Order int
}

func init() {
	SetDefProp(PropServerEnabled, true)
	SetDefProp(PropServerHost, "0.0.0.0")
	SetDefProp(PropServerPort, 8080)
	SetDefProp(PropServerGracefulShutdownTimeSec, 5)
	SetDefProp(PropServerPerfEnabled, false)
	SetDefProp(PropServerPropagateInboundTrace, true)
	SetDefProp(PropServerRequestValidateEnabled, true)
	SetDefProp(PropServerPprofEnabled, false)
	SetDefProp(PropServerRequestAutoMapHeader, true)
	SetDefProp(PropServerGinValidationDisabled, true)

	SetDefProp(PropLoggingRollingFileMaxAge, 0)
	SetDefProp(PropLoggingRollingFileMaxSize, 50)
	SetDefProp(PropLoggingRollingFileMaxBackups, 10)
	SetDefProp(PropLoggingRollingFileRotateDaily, true)

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap HTTP Server",
		Bootstrap: WebServerBootstrap,
		Condition: WebServerBootstrapCondition,
		Order:     BootstrapOrderL3,
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

// Register shutdown hook, hook should never panic
func AddShutdownHook(hook func()) {
	addOrderedShutdownHook(defShutdownOrder, hook)
}

type OrderedShutdownHook struct {
	Hook  func()
	Order int
}

func addOrderedShutdownHook(order int, hook func()) {
	shmu.Lock()
	defer shmu.Unlock()
	shutdownHook = append(shutdownHook, OrderedShutdownHook{
		Order: order,
		Hook:  hook,
	})
}

// Trigger shutdown hook
func triggerShutdownHook() {
	shmu.Lock()
	defer shmu.Unlock()

	sort.Slice(shutdownHook, func(i, j int) bool { return shutdownHook[i].Order < shutdownHook[j].Order })
	for _, hook := range shutdownHook {
		hook.Hook()
	}
}

// Record server route
func recordHttpServerRoute(url string, method string, extra ...util.StrPair) {
	extras := util.MergeStrPairs(extra...)
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

// Register ANY request route (raw version)
func RawAny(url string, handler RawTRouteHandler, extra ...util.StrPair) {
	for i := range anyHttpMethods {
		recordHttpServerRoute(url, anyHttpMethods[i], extra...)
	}
	addRoutesRegistar(func(e *gin.Engine) { e.Any(url, newRawTRouteHandler(handler)) })
}

// Register GET request route (raw version)
func RawGet(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodGet, newRawTRouteHandler(handler))
}

// Register POST request route (raw version)
func RawPost(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPost, newRawTRouteHandler(handler))
}

// Register PUT request route (raw version)
func RawPut(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPut, newRawTRouteHandler(handler))
}

// Register DELETE request route (raw version)
func RawDelete(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodDelete, newRawTRouteHandler(handler))
}

// Register GET request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Get[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodGet, newTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register POST request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Post[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPost, newTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register PUT request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Put[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPut, newTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register DELETE request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Delete[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodDelete, newTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register POST request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param) or 'header' tag (only supports string/*string).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IPost[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPost, newMappedTRouteHandler(handler)).
		DocJsonReq(util.NewVar[Req]()).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register GET request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'form' tag (for form-data, query param) or 'header' tag (only supports string/*string).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IGet[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	var r Req
	return NewLazyRouteDecl(url, http.MethodGet, newMappedTRouteHandler(handler)).
		DocQueryReq(r).
		DocHeaderReq(r).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register DELETE request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param) or 'header' tag (only supports string/*string).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IDelete[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	var r Req
	return NewLazyRouteDecl(url, http.MethodDelete, newMappedTRouteHandler(handler)).
		DocQueryReq(r).
		DocHeaderReq(r).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

// Register PUT request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param) or 'header' tag (only supports string/*string).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IPut[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPut, newMappedTRouteHandler(handler)).
		DocJsonReq(util.NewVar[Req]()).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(util.NewVar[Res]()))
}

type routesRegistar func(*gin.Engine)

func addRoutesRegistar(reg routesRegistar) {
	routeRegistars = append(routeRegistars, reg)
}

// Register GIN route for consul healthcheck
func registerRouteForConsulHealthcheck(router *gin.Engine) {
	router.GET(GetPropStr(PropConsulHealthcheckUrl), DefaultHealthCheck)
}

func startHttpServer(rail Rail, server *http.Server) {
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		panic(fmt.Errorf("http.Server Serve: %s", err))
	}
	la := ln.Addr().(*net.TCPAddr)
	rail.Infof("Serving HTTP on %s (actual port: %d)", server.Addr, la.Port)
	SetProp(PropServerActualPort, la.Port)
	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		panic(fmt.Errorf("http.Server Serve: %s", err))
	}
}

func createHttpServer(router http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%s", GetPropStr(PropServerHost), GetPropStr(PropServerPort))
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	return server
}

// Configure logging level and output target based on loaded configuration.
func ConfigureLogging(rail Rail) error {

	// determine the writer that we will use for logging (loggerOut and loggerErrOut)
	if ContainsProp(PropLoggingRollingFile) {
		logFile := GetPropStr(PropLoggingRollingFile)
		log := BuildRollingLogFileWriter(NewRollingLogFileParam{
			Filename:   logFile,
			MaxSize:    GetPropInt(PropLoggingRollingFileMaxSize), // megabytes
			MaxAge:     GetPropInt(PropLoggingRollingFileMaxAge),  //days
			MaxBackups: GetPropInt(PropLoggingRollingFileMaxBackups),
		})
		loggerOut = log
		loggerErrOut = log

		if GetPropBool(PropLoggingRollingFileRotateDaily) {
			// schedule a job to rotate the log at 00:00:00
			if err := ScheduleCron(Job{
				Name:            "RotateLogJob",
				Cron:            "0 0 0 * * ?",
				CronWithSeconds: true,
				Run:             func(r Rail) error { return log.Rotate() },
			}); err != nil {
				return fmt.Errorf("failed to register RotateLogJob, %v", err)
			}
		}
	}

	logrus.SetOutput(loggerOut)

	if HasProp(PropLoggingLevel) {
		if level, ok := ParseLogLevel(GetPropStr(PropLoggingLevel)); ok {
			logrus.SetLevel(level)
		}
	}
	return nil
}

func callPostServerBootstrapListeners(rail Rail) error {
	i := 0
	for i < len(postServerBootstrapListener) {
		if e := postServerBootstrapListener[i](rail); e != nil {
			return e
		}
		i++
	}
	postServerBootstrapListener = nil
	return nil
}

// Add listener that is invoked when server is finally bootstrapped
//
// This usually means all server components are started, such as MySQL connection, Redis Connection and so on.
//
// Caller is free to call PostServerBootstrapped inside another PostServerBootstrapped callback.
func PostServerBootstrapped(callback func(rail Rail) error) {
	if callback == nil {
		return
	}
	postServerBootstrapListener = append(postServerBootstrapListener, callback)
}

// Add listener that is invoked before the server is fully bootstrapped
//
// This usually means that the configuration is loaded, and the logging is configured, but the server components are not yet initialized.
//
// Caller is free to call PostServerBootstrapped or PreServerBootstrap inside another PreServerBootstrap callback.
func PreServerBootstrap(callback func(rail Rail) error) {
	if callback == nil {
		return
	}
	preServerBootstrapListener = append(preServerBootstrapListener, callback)
}

func callPreServerBootstrapListeners(rail Rail) error {
	i := 0
	for i < len(preServerBootstrapListener) {
		if e := preServerBootstrapListener[i](rail); e != nil {
			return e
		}
		i++
	}
	preServerBootstrapListener = nil
	return nil
}

// Register server component bootstrap callback
//
// When such callback is invoked, configuration should be fully loaded, the callback is free to read the loaded configuration
// and decide whether or not the server component should be initialized, e.g., by checking if the enable flag is true.
func RegisterBootstrapCallback(bootstrapComponent ComponentBootstrap) {
	serverBootrapCallbacks = append(serverBootrapCallbacks, bootstrapComponent)
}

/*
Bootstrap server

This func will attempt to create http server, connect to MySQL, Redis or Consul based on the configuration loaded.

It also handles service registration/de-registration on Consul before Gin bootstraped and after
SIGTERM/INTERRUPT signals are received.

Graceful shutdown for the http server is also enabled and can be configured through props.

To configure server, MySQL, Redis, Consul and so on, see PROPS_* in prop.go.

It's also possible to register callbacks that are triggered before/after server bootstrap

	miso.PreServerBootstrap(func(c Rail) error {
		// do something right after configuration being loaded, but server hasn't been bootstraped yet
	});

	miso.PostServerBootstrapped(func(c Rail) error {
		// do something after the server bootstrap
	});

	// start the server
	miso.BootstrapServer(os.Args)
*/
func BootstrapServer(args []string) {
	osSigQuit := make(chan os.Signal, 2)
	signal.Notify(osSigQuit, os.Interrupt, syscall.SIGTERM)

	addOrderedShutdownHook(0, MarkServerShuttingDown) // the first hook to be called
	var rail Rail = EmptyRail()

	start := time.Now().UnixMilli()
	defer triggerShutdownHook()

	// default way to load configuration
	DefaultReadConfig(args, rail)

	// configure logging
	if err := ConfigureLogging(rail); err != nil {
		rail.Errorf("Configure logging failed, %v", err)
		return
	}

	appName := GetPropStr(PropAppName)
	if appName == "" {
		rail.Fatalf("Property '%s' is required", PropAppName)
	}

	rail.Infof("\n\n---------------------------------------------- starting %s -------------------------------------------------------\n", appName)
	rail.Infof("Miso Version: %s", version.Version)
	rail.Infof("Production Mode: %v", GetPropBool(PropProdMode))

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	if e := callPreServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while invoking pre server bootstrap callbacks, %v", e)
		return
	}

	// bootstrap components, these are sorted by their orders
	sort.Slice(serverBootrapCallbacks, func(i, j int) bool { return serverBootrapCallbacks[i].Order < serverBootrapCallbacks[j].Order })
	Debugf("serverBootrapCallbacks: %+v", serverBootrapCallbacks)
	for _, sbc := range serverBootrapCallbacks {
		if sbc.Condition != nil {
			ok, ce := sbc.Condition(rail)
			if ce != nil {
				rail.Errorf("Failed to bootstrap server component: %v, failed on condition check, %v", sbc.Name, ce)
				return
			}
			if !ok {
				continue
			}
		}

		start := time.Now()
		if e := sbc.Bootstrap(rail); e != nil {
			rail.Errorf("Failed to bootstrap server component: %v, %v", sbc.Name, e)
			return
		}
		took := time.Since(start)
		rail.Debugf("Callback %-30s - took %v", sbc.Name, took)
		if took >= 5*time.Second {
			rail.Warnf("Component '%s' might be too slow to bootstrap, took: %v", sbc.Name, took)
		}
	}
	serverBootrapCallbacks = nil

	end := time.Now().UnixMilli()
	rail.Infof("\n\n---------------------------------------------- %s started (took: %dms) --------------------------------------------\n", appName, end-start)

	// invoke listener for serverBootstraped event
	if e := callPostServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while invoking post server bootstrap callbacks, %v", e)
		return
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	select {
	case sig := <-osSigQuit:
		rail.Infof("Received OS signal: %v, exiting", sig)
	case <-manualSigQuit: // or wait for maunal shutdown signal
		rail.Infof("Received manual shutdown signal, exiting")
	}
}

// Shutdown server
func Shutdown() {
	manualSigQuit <- 1
}

// Register http routes on gin.Engine
func registerServerRoutes(c Rail, engine *gin.Engine) {
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

	for _, r := range GetHttpRoutes() {
		c.Debugf("%-6s %s", r.Method, r.Url)
	}
}

/*
shutdown http server, including gracefull shutdown within certain duration of time

This func looks for following prop:

	"server.gracefulShutdownTimeSec"
*/
func shutdownHttpServer(server *http.Server) {
	Info("Shutting down http server gracefully")

	// set timeout for graceful shutdown
	timeout := GetPropInt(PropServerGracefulShutdownTimeSec)
	if timeout <= 0 {
		timeout = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// shutdown web server with the timeout
	server.Shutdown(ctx)
	Infof("Http server exited")
}

// Default Recovery func
func DefaultRecovery(c *gin.Context, e interface{}) {
	rail := BuildRail(c)
	rail.Errorf("%v '%v' Recovered from panic, %v", c.Request.Method, c.Request.RequestURI, e)

	// response already written, avoid writting it again.
	if c.Writer.Written() {
		if me, ok := e.(*MisoErr); ok {
			rail.Infof("Miso error, code: '%v', msg: '%v', internalMsg: '%v'", me.Code, me.Msg, me.InternalMsg)
			return
		}
		rail.Errorf("Unknown error, %v", e)
		return
	}

	if err, ok := e.(error); ok {
		endpointResultHandler(c, rail, nil, err)
		return
	}

	endpointResultHandler(c, rail, nil, NewErrf("Unknown error, please try again later"))
}

// check if the server is shutting down
func IsShuttingDown() bool {
	shutingDownRwm.RLock()
	defer shutingDownRwm.RUnlock()
	return shuttingDown
}

// mark that the server is shutting down
func MarkServerShuttingDown() {
	shutingDownRwm.Lock()
	defer shutingDownRwm.Unlock()
	shuttingDown = true
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
// using 'json' tag or 'form' tag (for form-data, query param) or 'header' tag (only supports string/*string).
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
		MustBind(rail, c, &req)

		wtcbCnt := 0
		if GetPropBool(PropServerRequestValidateEnabled) {
			wtcbCnt += 2
		}
		if GetPropBool(PropServerRequestAutoMapHeader) {
			wtcbCnt += 1
		}
		if wtcbCnt > 0 {
			wtcb := make([]util.WalkTagCallback, 0, wtcbCnt)

			// validate request
			if GetPropBool(PropServerRequestValidateEnabled) {
				wtcb = append(wtcb, ValidateWalkTagCallback, ValidateWalkTagCallbackDeprecated)
			}

			// for setting headers
			if GetPropBool(PropServerRequestAutoMapHeader) {
				wtcb = append(wtcb, reflectSetHeaderCallback(c))
			}

			if err := util.WalkTagShallow(&req, wtcb...); err != nil {
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
func MustBind(rail Rail, c *gin.Context, ptr any) {
	onFailed := func(err error) {
		rail.Errorf("Bind payload failed, %v", err)
		panic(NewErrf("Illegal Arguments"))
	}

	// we now use jsoniter
	if c.Request.Method != http.MethodGet && c.ContentType() == gin.MIMEJSON {
		if err := jsoniter.NewDecoder(c.Request.Body).Decode(ptr); err != nil {
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
func DispatchJsonCode(c *gin.Context, code int, body interface{}) {
	c.Status(code)
	c.Header("Content-Type", applicationJson)

	err := jsoniter.NewEncoder(c.Writer).Encode(body)
	if err != nil {
		panic(err)
	}
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	DispatchJson(c, ErrorResp(msg))
}

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	DispatchJsonCode(c, http.StatusOK, body)
}

func WebServerBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropServerEnabled), nil
}

func WebServerBootstrap(rail Rail) error {
	rail.Info("Starting HTTP server")

	// Load propagation keys for tracing
	LoadPropagationKeys(rail)

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

	if !pprofRegisterDisabled && (!IsProdMode() || GetPropBool(PropServerPprofEnabled)) {
		GroupRoute("/debug/pprof",
			RawGet("", func(inb *Inbound) { pprof.Index(inb.Unwrap()) }),
			RawGet("/:name", func(inb *Inbound) { pprof.Index(inb.Unwrap()) }),
			RawGet("/cmdline", func(inb *Inbound) { pprof.Cmdline(inb.Unwrap()) }),
			RawGet("/profile", func(inb *Inbound) { pprof.Profile(inb.Unwrap()) }),
			RawGet("/symbol", func(inb *Inbound) { pprof.Symbol(inb.Unwrap()) }),
			RawGet("/trace", func(inb *Inbound) { pprof.Trace(inb.Unwrap()) }),
		)
	}

	// register customer recovery func
	engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

	// register consul health check
	if GetPropBool(PropConsulEnabled) && GetPropBool(PropConsulRegisterDefaultHealthcheck) && !defaultHealthCheckHandlerDisabled {
		registerRouteForConsulHealthcheck(engine)
	}

	if !IsProdMode() {
		if err := serveApiDocTmpl(rail); err != nil {
			rail.Errorf("failed to buildEndpointDocTmpl, %v", err)
		}
	}

	// register http routes
	registerServerRoutes(rail, engine)

	// start the http server
	server := createHttpServer(engine)
	go startHttpServer(rail, server)

	AddShutdownHook(func() { shutdownHttpServer(server) })
	return nil
}

type TreePath interface {
	Prepend(baseUrl string)
}

// Lazy route declaration
type LazyRouteDecl struct {
	Url     string
	Method  string
	Handler func(c *gin.Context)

	RegisterFunc func(extra ...util.StrPair)
	Extras       []util.StrPair
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

type extraMatchCond = func(key string, val any, ex util.StrPair) (overwrite bool, breakLoop bool)

func (g *LazyRouteDecl) extra(key string, value any, cond extraMatchCond) *LazyRouteDecl {
	if cond == nil {
		g.Extras = append(g.Extras, util.StrPair{Left: key, Right: value})
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
		g.Extras = append(g.Extras, util.StrPair{Left: key, Right: value})
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
	return func(key string, val any, ex util.StrPair) (overwrite bool, breakLoop bool) {
		if key == ex.Left {
			return true, true
		}
		return false, false
	}
}

func extraFilterOneParamDocByName() extraMatchCond {
	return func(key string, val any, ex util.StrPair) (overwrite bool, breakLoop bool) {
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

func NewLazyRouteDecl(url string, method string, handler func(c *gin.Context)) *LazyRouteDecl {
	dec := &LazyRouteDecl{
		Url:     url,
		Method:  method,
		Handler: handler,
		Extras:  []util.StrPair{},
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

func reflectSetHeaderCallback(c *gin.Context) util.WalkTagCallback {
	return walkHeaderTagCallback(func(k string) string { return c.GetHeader(k) })
}

func walkHeaderTagCallback(getHeader func(k string) string) util.WalkTagCallback {
	return util.WalkTagCallback{
		Tag: TagHeaderParam,
		OnWalked: func(tagVal string, fieldVal reflect.Value, fieldType reflect.StructField) error {
			hv := getHeader(tagVal)
			if hv == "" {
				return nil
			}
			switch fieldType.Type.Kind() {
			case reflect.String:
				fieldVal.SetString(hv)
			case reflect.Pointer:
				ptrType := fieldType.Type.Elem()
				if ptrType.Kind() == reflect.String {
					fieldVal.Set(reflect.ValueOf(&hv))
				}
			}
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
	engine  any
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

func (i *Inbound) Rail() Rail {
	return i.erail
}

/*
Handle the result using universally configured handler.

The result or error is written back to the client. In most cases, caller must exit the handler
after calling this method. Theoritically, this method is only useful for RawGet, RawPut, RawPost, RawDelete.
Other methods, such as IGet, IPost, Post, Put or Delete, handle the results automatically in exactly the same way.

E.g.,

	miso.RawGet("/dir/info", func(inb *miso.Inbound) {
		// ... do something

		if err != nil {
			inb.HandleResult(nil, err) // something goes wrong
			return
		}

		// return result back to the client
		inb.HandleResult(result, err)
	})
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

func (i *Inbound) Header(k string) string {
	return i.r.Header.Get(k)
}

func (i *Inbound) SetHeader(k string, v string) {
	i.r.Header.Set(k, v)
}

func (i *Inbound) AddHeader(k string, v string) {
	i.r.Header.Add(k, v)
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
