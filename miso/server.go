package miso

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

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

	ScopePublic    = "PUBLIC"
	ScopeProtected = "PROTECTED"

	defShutdownOrder = 5

	TagQueryParam = "form"
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

	endpointResultHandler EndpointResultHandler = func(c *gin.Context, rail Rail, payload any, err error) {
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

	// enable pprof manually
	manualRegisterPprof = false
)

type OrderedShutdownHook struct {
	Hook  func()
	Order int
}

// Preprocessor of *gin.Engine.
type GinPreProcessor func(rail Rail, engine *gin.Engine)

// Raw version of traced route handler.
type RawTRouteHandler func(c *gin.Context, rail Rail)

// Traced route handler.
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With Res type declared, miso will automatically parse the Res type using reflect and generate an API documentation
// describing the endpoint.
type TRouteHandler[Res any] func(c *gin.Context, rail Rail) (Res, error)

// Traced and parameters mapped route handler.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
type MappedTRouteHandler[Req any, Res any] func(c *gin.Context, rail Rail, req Req) (Res, error)

type routesRegistar func(*gin.Engine)

type HttpRoute struct {
	Url             string
	Method          string
	Extra           map[string][]any
	Desc            string     // description of the route (metadata).
	Scope           string     // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource        string     // the documented resource that the route should be bound to (metadata).
	Headers         []ParamDoc // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams     []ParamDoc // the documented query parameters that will used by the endpoint (metadata).
	JsonRequestVal  any        // the documented json request value that is expected by the endpoint (metadata).
	JsonResponseVal any        // the documented json response value that will be returned by the endpoint (metadata).
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

type ResultBodyBuilder struct {
	// wrap error in json, the returned object will be serialized to json.
	ErrJsonBuilder func(rail Rail, url string, err error) any

	// wrap payload object, the returned object will be serialized to json.
	PayloadJsonBuilder func(payload any) any

	// build empty ok response object, the returned object will be serialized to json.
	OkJsonBuilder func() any
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

	SetDefProp(PropLoggingRollingFileMaxAge, 0)
	SetDefProp(PropLoggingRollingFileMaxSize, 50)
	SetDefProp(PropLoggingRollingFileMaxBackups, 0)
	SetDefProp(PropLoggingRollingFileRotateDaily, true)

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap HTTP Server",
		Bootstrap: WebServerBootstrap,
		Condition: WebServerBootstrapCondition,
		Order:     BootstrapOrderL3,
	})
}

type EndpointResultHandler func(c *gin.Context, rail Rail, payload any, err error)

// Replace the default ResultBodyBuilder
func SetResultBodyBuilder(rbb ResultBodyBuilder) error {
	resultBodyBuilder = rbb
	return nil
}

// Register shutdown hook, hook should never panic
func AddShutdownHook(hook func()) {
	addOrderedShutdownHook(defShutdownOrder, hook)
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
func recordHttpServerRoute(url string, method string, extra ...StrPair) {
	extras := MergeStrPairs(extra...)
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
	if l, ok := extras[ExtraJsonRequest]; ok && len(l) > 0 {
		r.JsonRequestVal = l[0]
	}
	if l, ok := extras[ExtraJsonResponse]; ok && len(l) > 0 {
		r.JsonResponseVal = l[0]
	}
	serverHttpRoutes = append(serverHttpRoutes, r)
}

// Get recorded http server routes
func GetHttpRoutes() []HttpRoute {
	return serverHttpRoutes
}

// Register ANY request route (raw version)
func RawAny(url string, handler RawTRouteHandler, extra ...StrPair) {
	for i := range anyHttpMethods {
		recordHttpServerRoute(url, anyHttpMethods[i], extra...)
	}
	addRoutesRegistar(func(e *gin.Engine) { e.Any(url, NewRawTRouteHandler(handler)) })
}

// Register GET request route (raw version)
func RawGet(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodGet, NewRawTRouteHandler(handler))
}

// Register POST request route (raw version)
func RawPost(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPost, NewRawTRouteHandler(handler))
}

// Register PUT request route (raw version)
func RawPut(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPut, NewRawTRouteHandler(handler))
}

// Register DELETE request route (raw version)
func RawDelete(url string, handler RawTRouteHandler) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodDelete, NewRawTRouteHandler(handler))
}

// Register GET request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Get[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodGet, NewTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register POST request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Post[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPost, NewTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register PUT request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Put[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPut, NewTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register DELETE request.
//
// The result and error are automatically wrapped to miso.Resp (see miso.SetResultBodyBuilder func)
// and serialized to json.
func Delete[Res any](url string, handler TRouteHandler[Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodDelete, NewTRouteHandler(handler)).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register POST request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IPost[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPost, NewMappedTRouteHandler(handler)).
		DocJsonReq(New[Req]()).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register GET request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'form' tag (for form-data, query param).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IGet[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodGet, NewMappedTRouteHandler(handler)).
		DocQueryReq(New[Req]()).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register DELETE request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IDelete[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodDelete, NewMappedTRouteHandler(handler)).
		DocQueryReq(New[Req]()).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

// Register PUT request.
//
// Req type should be a struct, where all fields are automatically mapped from the request
// using 'json' tag or 'form' tag (for form-data, query param).
//
// Res type should be a struct. By default both Res value and error (if not nil) will be wrapped inside
// miso.Resp and serialized to json. Wrapping to miso.Resp is customizable using miso.SetResultBodyBuilder func.
//
// With both Req and Res type declared, miso will automatically parse these two types using reflect
// and generate an API documentation describing the endpoint.
func IPut[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res]) *LazyRouteDecl {
	return NewLazyRouteDecl(url, http.MethodPut, NewMappedTRouteHandler(handler)).
		DocJsonReq(New[Req]()).
		DocJsonResp(resultBodyBuilder.PayloadJsonBuilder(New[Res]()))
}

func addRoutesRegistar(reg routesRegistar) {
	routeRegistars = append(routeRegistars, reg)
}

// Register GIN route for consul healthcheck
func registerRouteForConsulHealthcheck(router *gin.Engine) {
	router.GET(GetPropStr(PropConsulHealthcheckUrl), DefaultHealthCheck)
}

func startHttpServer(rail Rail, server *http.Server) {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		rail.Fatalf("http.Server ListenAndServe: %s", err)
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

	if HasProp(PropLoggingFile) {
		if level, ok := ParseLogLevel(GetPropStr(PropLoggingFile)); ok {
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

	server.PreServerBootstrap(func(c Rail) error {
		// do something right after configuration being loaded, but server hasn't been bootstraped yet
	});

	server.PostServerBootstrapped(func(c Rail) error {
		// do something after the server bootstrap
	});

	// start the server
	server.BootstrapServer(os.Args)
*/
func BootstrapServer(args []string) {
	addOrderedShutdownHook(0, MarkServerShuttingDown) // the first hook to be called
	var rail Rail = EmptyRail()

	start := time.Now().UnixMilli()
	defer triggerShutdownHook()

	rail, cancel := rail.WithCancel()
	AddShutdownHook(cancel)

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
	rail.Infof("Miso Version: %s", Version)
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
		rail.Debugf("Callback %-30s - took %v", sbc.Name, time.Since(start))
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
	osSigQuit := make(chan os.Signal, 2)
	signal.Notify(osSigQuit, os.Interrupt, syscall.SIGTERM)

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
		c := BuildRail(ctx)
		c.Warnf("NoRoute for %s '%s', returning 404", ctx.Request.Method, ctx.Request.RequestURI)
		ctx.AbortWithStatus(404)
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
	rail.Errorf("Recovered from panic, %v", e)

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
		HandleEndpointResult(c, rail, nil, err)
		return
	}

	HandleEndpointResult(c, rail, nil, NewErrf("Unknown error, please try again later"))
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

// Build route handler with the mapped payload object, context, and logger.
//
// value and error returned by handler are automically wrapped in a Resp object
func NewMappedTRouteHandler[Req any, Res any](handler MappedTRouteHandler[Req, Res]) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := BuildRail(c)

		// bind to payload boject
		var req Req
		MustBind(rail, c, &req)

		if GetPropBool(PropServerRequestValidateEnabled) {
			// validate request
			if e := Validate(req); e != nil {
				HandleEndpointResult(c, rail, nil, e)
				return
			}
		}

		if GetPropBool(PropServerRequestLogEnabled) {
			rail.Infof("%v %v, req: %+v", c.Request.Method, c.Request.RequestURI, req)
		}

		// handle the requests
		res, err := handler(c, rail, req)

		// wrap result and error
		HandleEndpointResult(c, rail, res, err)
	}
}

// Build route handler with context, and logger
func NewRawTRouteHandler(handler RawTRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		handler(c, BuildRail(c))
	}
}

// Build route handler with context, and logger
//
// value and error returned by handler are automically wrapped in a Resp object
func NewTRouteHandler[Res any](handler TRouteHandler[Res]) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := BuildRail(c)
		r, e := handler(c, rail)
		HandleEndpointResult(c, rail, r, e)
	}
}

// Handle endpoint's result using the configured EndpointResultHandler.
func HandleEndpointResult(c *gin.Context, rail Rail, result any, err error) {
	endpointResultHandler(c, rail, result, err)
}

// Must bind request payload to the given pointer, else panic
func MustBind(rail Rail, c *gin.Context, ptr any) {
	onFailed := func(err error) {
		rail.Errorf("Bind payload failed, %v", err)
		panic("Illegal Arguments")
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

	if !manualRegisterPprof && (!IsProdMode() || GetPropBool(PropServerPprofEnabled)) {
		GroupRoute("/debug/pprof",
			RawGet("", func(c *gin.Context, rail Rail) { pprof.Index(c.Writer, c.Request) }),
			RawGet("/:name", func(c *gin.Context, rail Rail) { pprof.Index(c.Writer, c.Request) }),
			RawGet("/cmdline", func(c *gin.Context, rail Rail) { pprof.Cmdline(c.Writer, c.Request) }),
			RawGet("/profile", func(c *gin.Context, rail Rail) { pprof.Profile(c.Writer, c.Request) }),
			RawGet("/symbol", func(c *gin.Context, rail Rail) { pprof.Symbol(c.Writer, c.Request) }),
			RawGet("/trace", func(c *gin.Context, rail Rail) { pprof.Trace(c.Writer, c.Request) }),
		)
	}

	// register customer recovery func
	engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

	// register consul health check
	if GetPropBool(PropConsulEnabled) && GetPropBool(PropConsulRegisterDefaultHealthcheck) {
		registerRouteForConsulHealthcheck(engine)
	}

	if !IsProdMode() && GetPropBool(PropServerGenerateEndpointDocEnabled) {
		if err := serveApiDocTmpl(rail); err != nil {
			rail.Errorf("failed to buildEndpointDocTmpl, %v", err)
		}
	}

	// register http routes
	registerServerRoutes(rail, engine)

	// start the http server
	server := createHttpServer(engine)
	rail.Infof("Serving HTTP on %s", server.Addr)
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

	RegisterFunc func(extra ...StrPair)
	Extras       []StrPair
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

type extraMatchCond = func(key string, val any, ex StrPair) (overwrite bool, breakLoop bool)

func (g *LazyRouteDecl) extra(key string, value any, cond extraMatchCond) *LazyRouteDecl {
	if cond == nil {
		g.Extras = append(g.Extras, StrPair{key, value})
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
		g.Extras = append(g.Extras, StrPair{key, value})
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
	return g.extra(ExtraJsonRequest, v, extraFilterOneByKey())
}

// Document json response that the endpoint returns (only serves as metadata that maybe used by some plugins).
func (g *LazyRouteDecl) DocJsonResp(v any) *LazyRouteDecl {
	return g.extra(ExtraJsonResponse, v, extraFilterOneByKey())
}

func extraFilterOneByKey() extraMatchCond {
	return func(key string, val any, ex StrPair) (overwrite bool, breakLoop bool) {
		if key == ex.Left {
			return true, true
		}
		return false, false
	}
}

func extraFilterOneParamDocByName() extraMatchCond {
	return func(key string, val any, ex StrPair) (overwrite bool, breakLoop bool) {
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
		Extras:  []StrPair{},
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

// Registrer pprof debug endpoint manually.
func ManualPprofRegister() {
	manualRegisterPprof = true
}

// Process *gin.Engine before the web server starts, particularly useful when trying to add middleware.
func PreProcessGin(preProcessor GinPreProcessor) {
	ginPreProcessors = append(ginPreProcessors, preProcessor)
}
