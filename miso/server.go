package miso

import (
	"context"
	"errors"
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
	ScopePublic       = "PUBLIC"

	TagApiDocDesc = "desc"
)

var (
	ApiDocTypeAlias = map[string]string{
		"ETime":       "int64",
		"*ETime":      "int64",
		"*miso.ETime": "int64",
	}

	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	routeRegistars   = []routesRegistar{}
	serverHttpRoutes = []HttpRoute{}
	ginPreProcessors = []GinPreProcessor{}

	shuttingDown   bool         = false
	shutingDownRwm sync.RWMutex // rwmutex for shuttingDown

	shutdownHook []func()
	shmu         sync.Mutex // mutex for shutdownHook

	// server component bootstrap callbacks
	serverBootrapCallbacks []ComponentBootstrap = []ComponentBootstrap{}

	// listener for events trigger before server components being bootstrapped
	preServerBootstrapListener []func(r Rail) error = []func(r Rail) error{}
	// listener for events trigger after server components bootstrapped
	postServerBootstrapListener []func(r Rail) error = []func(r Rail) error{}

	// all http methods
	anyHttpMethods = []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect,
		http.MethodTrace,
	}

	// channel for signaling server shutdown
	manualSigQuit = make(chan int, 1)

	defaultResultBodyBuilder = ResultBodyBuilder{
		ErrJsonBuilder:     func(rail Rail, err error) any { return WrapResp(nil, err, rail) },
		PayloadJsonBuilder: func(payload any) any { return OkRespWData(payload) },
		OkJsonBuilder:      func() any { return OkResp() },
	}

	endpointResultHandler EndpointResultHandler = func(c *gin.Context, rail Rail, payload any, err error) {
		BuildEndpointResultHandler(c, rail, payload, err, defaultResultBodyBuilder)
	}

	manualRegisterPprof = false
)

// Preprocessor of *gin.Engine.
type GinPreProcessor func(rail Rail, engine *gin.Engine)

// Raw version of traced route handler.
type RawTRouteHandler func(c *gin.Context, rail Rail)

// Traced route handler.
type TRouteHandler func(c *gin.Context, rail Rail) (any, error)

/*
Traced and parameters mapped route handler.

T should be a struct, where all fields are automatically mapped from the request using different tags.

  - json
  - xml
  - form (supports: form-data, query param)

For binding, go read https://gin-gonic.com/docs/
*/
type MappedTRouteHandler[Req any] func(c *gin.Context, rail Rail, req Req) (any, error)

type routesRegistar func(*gin.Engine)

type HttpRoute struct {
	Url              string
	Method           string
	Extra            map[string][]any
	HandlerName      string
	Desc             string        // description of the route (metadata).
	Scope            string        // the documented access scope of the route, it maybe "PUBLIC" or something else (metadata).
	Resource         string        // the documented resource that the route should be bound to (metadata).
	Headers          []ParamDoc    // the documented header parameters that will be used by the endpoint (metadata).
	QueryParams      []ParamDoc    // the documented query parameters that will used by the endpoint (metadata).
	JsonRequestType  *reflect.Type // the documented json request type that is expected by the endpoint (metadata).
	JsonResponseType *reflect.Type // the documented json response type that will be returned by the endpoint (metadata).
}

type ParamDoc struct {
	Name string
	Desc string
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
	ErrJsonBuilder     func(rail Rail, err error) any // wrap error in json, return json object that will be serialized.
	PayloadJsonBuilder func(payload any) any          // wrap payload in json.
	OkJsonBuilder      func() any                     // build empty ok json.
}

func init() {
	AddShutdownHook(MarkServerShuttingDown)

	SetDefProp(PropServerEnabled, true)
	SetDefProp(PropServerHost, "0.0.0.0")
	SetDefProp(PropServerPort, 8080)
	SetDefProp(PropServerGracefulShutdownTimeSec, 5)
	SetDefProp(PropServerPerfEnabled, false)
	SetDefProp(PropServerPropagateInboundTrace, true)
	SetDefProp(PropServerRequestValidateEnabled, true)
	SetDefProp(PropServerPprofEnabled, false)
	SetDefProp(PropServerGenerateEndpointDocEnabled, false)

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

// Replace the default EndpointResultHandler
func SetEndpointResultHandler(erh EndpointResultHandler) error {
	if erh == nil {
		return errors.New("EndpointResultHandler provided is nil")
	}
	endpointResultHandler = erh
	return nil
}

// Register shutdown hook, hook should never panic
func AddShutdownHook(hook func()) {
	shmu.Lock()
	defer shmu.Unlock()
	shutdownHook = append(shutdownHook, hook)
}

// Trigger shutdown hook
func triggerShutdownHook() {
	shmu.Lock()
	defer shmu.Unlock()

	Info("Triggering shutdown hook")
	for _, hook := range shutdownHook {
		hook()
	}
}

// Record server route
func recordHttpServerRoute(url string, method string, handlerName string, extra ...StrPair) {
	extras := MergeStrPairs(extra...)
	r := HttpRoute{
		Url:         url,
		Method:      method,
		HandlerName: handlerName,
		Extra:       extras,
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
		if v, ok := l[0].(*reflect.Type); ok {
			r.JsonRequestType = v
		}
	}
	if l, ok := extras[ExtraJsonResponse]; ok && len(l) > 0 {
		if v, ok := l[0].(*reflect.Type); ok {
			r.JsonResponseType = v
		}
	}
	serverHttpRoutes = append(serverHttpRoutes, r)
}

// Get recorded server routes (deprecated, use GetHttpRoutes() instead)
func GetRecordedHttpServerRoutes() []string {
	urls := make([]string, len(serverHttpRoutes))
	for _, r := range serverHttpRoutes {
		urls = append(urls, r.Url)
	}
	return urls
}

// Get recorded http server routes
func GetHttpRoutes() []HttpRoute {
	return serverHttpRoutes
}

// Register ANY request route (raw version
func RawAny(url string, handler RawTRouteHandler, extra ...StrPair) {
	for i := range anyHttpMethods {
		recordHttpServerRoute(url, anyHttpMethods[i], FuncName(handler), extra...)
	}
	addRoutesRegistar(func(e *gin.Engine) { e.Any(url, NewRawTRouteHandler(handler)) })
}

// Register GET request route (raw version)
func RawGet(url string, handler RawTRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodGet, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewRawTRouteHandler(handler)) })
	})
}

// Register POST request route (raw version)
func RawPost(url string, handler RawTRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodPost, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewRawTRouteHandler(handler)) })
	})
}

// Register PUT request route (raw version)
func RawPut(url string, handler RawTRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodPut, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewRawTRouteHandler(handler)) })
	})
}

// Register DELETE request route (raw version)
func RawDelete(url string, handler RawTRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodDelete, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewRawTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for GET request.
//
// The result or error is wrapped in Resp automatically.
func Get(url string, handler TRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodGet, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for POST request.
//
// The result or error is wrapped in Resp automatically.
func Post(url string, handler TRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodPost, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func Put(url string, handler TRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodPut, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for DELETE request.
//
// The result and error are wrapped in Resp automatically as json.
func Delete(url string, handler TRouteHandler) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodDelete, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for POST request with automatic payload binding.
//
// The result or error is wrapped in Resp automatically.
func IPost[Req any](url string, handler MappedTRouteHandler[Req]) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodPost, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewMappedTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for GET request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json.
func IGet[Req any](url string, handler MappedTRouteHandler[Req]) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodGet, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewMappedTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for DELETE request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json
func IDelete[Req any](url string, handler MappedTRouteHandler[Req]) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodDelete, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewMappedTRouteHandler(handler)) })
	})
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func IPut[Req any](url string, handler MappedTRouteHandler[Req]) GroupedRouteRegistar {
	return NewGroupedRouteRegistar(func(baseUrl string, extra ...StrPair) {
		url := baseUrl + url
		recordHttpServerRoute(url, http.MethodPut, FuncName(handler), extra...)
		addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewMappedTRouteHandler(handler)) })
	})
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
	rail.Infof("Miso Version: %s", MisoVersion)

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
		propagatedKeys := GetPropagationKeys()

		for _, k := range propagatedKeys {
			if h := c.GetHeader(k); h != "" {
				ctx = context.WithValue(ctx, k, h) //lint:ignore SA1029 keys must be exposed to retrieve the values
			}
		}

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
// be reflected on the Rail created here. But this should be find, because new span is usually created for async operation.
func BuildRail(c *gin.Context) Rail {
	if !GetPropBool(PropServerPropagateInboundTrace) {
		return EmptyRail()
	}

	if c.Keys == nil {
		c.Keys = map[string]any{}
	}

	tracked := GetPropagationKeys()
	ctx := c.Request.Context()

	// it's possible that the spanId and traceId have been set to the context
	// if we calling BuildRail() for the second time, we should read from the context
	// instead of creating new ones.
	// for the most of the time, we are using one single Rail throughout the method calls
	contextModified := false
	for i := range tracked {
		t := tracked[i]
		if v, ok := c.Keys[t]; ok && v != "" {
			ctx = context.WithValue(ctx, t, v) //lint:ignore SA1029 keys must be exposed for client to use
			contextModified = true
		}
	}

	// create a new Rail
	rail := NewRail(ctx)

	if !contextModified {
		for i := range tracked { // copy the newly created keys back to the gin.Context
			t := tracked[i]
			if v, ok := c.Keys[t]; !ok || v == "" {
				c.Keys[t] = rail.CtxValue(t)
			}
		}
	}

	return rail
}

// Build route handler with the mapped payload object, context, and logger.
//
// value and error returned by handler are automically wrapped in a Resp object
func NewMappedTRouteHandler[Req any](handler MappedTRouteHandler[Req]) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := BuildRail(c)

		// bind to payload boject
		var req Req
		MustBind(rail, c, &req)

		if GetPropBool(PropServerRequestValidateEnabled) {
			// validate request
			if e := Validate(req); e != nil {
				endpointResultHandler(c, rail, nil, e)
				return
			}
		}

		if GetPropBool(PropServerRequestLogEnabled) {
			rail.Infof("%v %v, req: %+v", c.Request.Method, c.Request.RequestURI, req)
		}

		// handle the requests
		res, err := handler(c, rail, req)

		// wrap result and error
		endpointResultHandler(c, rail, res, err)
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
func NewTRouteHandler(handler TRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := BuildRail(c)
		r, e := handler(c, rail)
		endpointResultHandler(c, rail, r, e)
	}
}

// Handle endpoint's result using the configured EndpointResultHandler.
func HandleEndpointResult(c *gin.Context, rail Rail, result any, err error) {
	endpointResultHandler(c, rail, result, err)
}

func BuildEndpointResultHandler(c *gin.Context, rail Rail, payload any, err error, builder ResultBodyBuilder) {
	if err != nil {
		DispatchJson(c, builder.ErrJsonBuilder(rail, err))
		return
	}
	if payload != nil {
		DispatchJson(c, builder.PayloadJsonBuilder(payload))
		return
	}
	DispatchJson(c, builder.OkJsonBuilder())
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

	if GetPropBool(PropServerPprofEnabled) && !manualRegisterPprof {
		path := "/debug/pprof"
		BaseRoute(path).Group(
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

	// register http routes
	registerServerRoutes(rail, engine)

	// start the http server
	server := createHttpServer(engine)
	rail.Infof("Serving HTTP on %s", server.Addr)
	go startHttpServer(rail, server)

	if GetPropBool(PropServerGenerateEndpointDocEnabled) {
		genEndpointDoc(rail)
	}

	AddShutdownHook(func() { shutdownHttpServer(server) })
	return nil
}

func genEndpointDoc(rail Rail) {
	b := strings.Builder{}
	b.WriteString("# API Endpoints\n")

	hr := GetHttpRoutes()
	for _, r := range hr {
		b.WriteString("\n- ")
		b.WriteString(r.Method)
		b.WriteString(" ")
		b.WriteString(r.Url)
		if r.Desc != "" {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- Description: ")
			b.WriteString(r.Desc)
		}
		// if r.Scope != "" {
		// 	b.WriteRune('\n')
		// 	b.WriteString(Spaces(2))
		// 	b.WriteString("- Access Scope: ")
		// 	b.WriteString(r.Scope)
		// }
		// if r.Resource != "" {
		// 	b.WriteRune('\n')
		// 	b.WriteString(Spaces(2))
		// 	b.WriteString("- Resource: \"")
		// 	b.WriteString(r.Resource)
		// 	b.WriteRune('"')
		// }
		if len(r.Headers) > 0 {
			for _, h := range r.Headers {
				b.WriteRune('\n')
				b.WriteString(Spaces(2))
				b.WriteString("- Header Parameter: \"")
				b.WriteString(h.Name)
				b.WriteString("\"\n")
				b.WriteString(Spaces(4))
				b.WriteString("- Description: ")
				b.WriteString(h.Desc)
			}
		}
		if len(r.QueryParams) > 0 {
			for _, q := range r.QueryParams {
				b.WriteRune('\n')
				b.WriteString(Spaces(2))
				b.WriteString("- Query Parameter: \"")
				b.WriteString(q.Name)
				b.WriteString("\"\n")
				b.WriteString(Spaces(4))
				b.WriteString("- Description: ")
				b.WriteString(q.Desc)
			}
		}
		if r.JsonRequestType != nil {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- JSON Request: ")
			buildJsonPayloadDoc(&b, *r.JsonRequestType, 2)
		}
		if r.JsonResponseType != nil {
			b.WriteRune('\n')
			b.WriteString(Spaces(2))
			b.WriteString("- JSON Response: ")
			buildJsonPayloadDoc(&b, *r.JsonResponseType, 2)
		}
	}
	rail.Infof("Generated API Endpoints Documentation:\n\n%s\n", b.String())
}

func buildJsonPayloadDoc(b *strings.Builder, t reflect.Type, indent int) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if IsVoid(f.Type) {
			continue
		}
		var name string
		if v := f.Tag.Get("json"); v != "" {
			name = v
		} else {
			name = LowercaseNamingStrategy(f.Name)
		}

		var typeName string
		if f.Type.Name() != "" {
			typeName = f.Type.Name()
		} else {
			typeName = f.Type.String()
		}
		typeAlias, typeAliasMatched := ApiDocTypeAlias[typeName]
		if typeAliasMatched {
			typeName = typeAlias
		}
		b.WriteString(fmt.Sprintf("\n%s- \"%s\": (%s) %s", Spaces(indent+2), name, typeName, f.Tag.Get(TagApiDocDesc)))

		if !typeAliasMatched {
			if f.Type.Kind() == reflect.Struct {
				buildJsonPayloadDoc(b, f.Type, indent+2)
			} else if f.Type.Kind() == reflect.Slice {
				et := f.Type.Elem()
				if et.Kind() == reflect.Struct {
					buildJsonPayloadDoc(b, et, indent+2)
				}
			}
		}
	}
}

type GroupedRouteRegistar struct {
	RegisterFunc func(baseUrl string, extra ...StrPair)
	Extras       []StrPair
}

// Build endpoint.
func (g GroupedRouteRegistar) Build() {
	g.RegisterFunc("", g.Extras...)
}

// Add endpoint description (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) Desc(desc string) GroupedRouteRegistar {
	return g.Extra(ExtraDesc, strings.TrimSpace(regexp.MustCompile(`[\n\t ]+`).ReplaceAllString(desc, " ")))
}

// Mark endpoint publicly accessible (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) Public() GroupedRouteRegistar {
	return g.Extra(ExtraScope, ScopePublic)
}

// Record the resource that the endppoint should be bound to (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) Resource(resource string) GroupedRouteRegistar {
	return g.Extra(ExtraResource, strings.TrimSpace(resource))
}

// Add extra info to endpoint's metadata.
func (g GroupedRouteRegistar) Extra(key string, value any) GroupedRouteRegistar {
	g.Extras = append(g.Extras, StrPair{key, value})
	return g
}

// Document query parameter that the endpoint will use (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) DocQueryParam(queryName string, desc string) GroupedRouteRegistar {
	return g.Extra(ExtraQueryParam, ParamDoc{queryName, desc})
}

// Document header parameter that the endpoint will use (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) DocHeader(headerName string, desc string) GroupedRouteRegistar {
	return g.Extra(ExtraHeaderParam, ParamDoc{headerName, desc})
}

// Document json request that the endpoint expects (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) DocJsonReq(t reflect.Type) GroupedRouteRegistar {
	return g.Extra(ExtraJsonRequest, &t)
}

// Document json response that the endpoint returns (only serves as metadata that maybe used by some plugins).
func (g GroupedRouteRegistar) DocJsonResp(t reflect.Type) GroupedRouteRegistar {
	return g.Extra(ExtraJsonResponse, &t)
}

// Create new GroupedRouteRegistar.
func NewGroupedRouteRegistar(f func(baseUrl string, extra ...StrPair)) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		RegisterFunc: f,
		Extras:       []StrPair{},
	}
}

type RoutingGroup struct {
	Base string
}

// Group routes, routes are immediately registered
func (rg *RoutingGroup) Group(grouped ...GroupedRouteRegistar) {
	for _, r := range grouped {
		r.RegisterFunc(rg.Base, r.Extras...)
	}
}

// Group routes under the sub paths, routes are immediately registered.
func (rg *RoutingGroup) With(subpaths ...*RoutingSubPath) {
	for _, s := range subpaths {
		for _, r := range s.delayedRegisters {
			r.RegisterFunc(rg.Base+s.path, r.Extras...)
		}
	}
}

// Group routes together to share the same base url.
func BaseRoute(baseUrl string) *RoutingGroup {
	return &RoutingGroup{Base: baseUrl}
}

// RoutingSubPath, each sub path belongs to a specific RoutingGroup (the base path)
type RoutingSubPath struct {
	path             string
	delayedRegisters []GroupedRouteRegistar
}

// Create sub path for routing requests
func SubPath(path string) *RoutingSubPath {
	return &RoutingSubPath{
		path:             path,
		delayedRegisters: []GroupedRouteRegistar{},
	}
}

// Group routes under current sub path.
func (s *RoutingSubPath) Group(grouped ...GroupedRouteRegistar) *RoutingSubPath {
	s.delayedRegisters = append(s.delayedRegisters, grouped...)
	return s
}

// Registrer pprof debug endpoint manually.
func ManualPprofRegister() {
	manualRegisterPprof = true
}

// Process *gin.Engine before the web server starts, particularly useful when trying to add middleware.
func PreProcessGin(preProcessor GinPreProcessor) {
	ginPreProcessors = append(ginPreProcessors, preProcessor)
}
