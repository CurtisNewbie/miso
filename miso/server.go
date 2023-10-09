package miso

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
)

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
	Url         string
	Method      string
	Extra       map[string]any
	HandlerName string
}

type ComponentBootstrap struct {
	Name      string
	Bootstrap func(rail Rail) error
}

const (
	OPEN_API_PREFIX = "/open/api" // merely a const value, doesn't have special meaning
)

var (
	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	routesRegiatarList []routesRegistar = []routesRegistar{}
	serverHttpRoutes   []HttpRoute      = []HttpRoute{}

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

	// handler of endpoint results (response object or error)
	serverResultHandler ServerResultHandler = func(c *gin.Context, rail Rail, r any, e error) {
		defaultHandleResult(c, rail, r, e)
	}

	requestValidationEnabled = false // whether request validation is enabled, read-only
)

type ServerResultHandler func(c *gin.Context, rail Rail, r any, e error)

func init() {
	SetDefProp(PropServerEnabled, true)
	SetDefProp(PropServerHost, "0.0.0.0")
	SetDefProp(PropServerPort, 8080)
	SetDefProp(PropServerGracefulShutdownTimeSec, 5)
	SetDefProp(PropServerPerfEnabled, false)
	SetDefProp(PropServerPropagateInboundTrace, true)
	SetDefProp(PropServerRequestValidateEnabled, true)
	SetDefProp(PropServerJsonNamingLowercase, true)

	SetDefProp(PropLoggingRollingFileMaxAge, 0)
	SetDefProp(PropLoggingRollingFileMaxSize, 50)
	SetDefProp(PropLoggingRollingFileMaxBackups, 0)

	// bootstrap callbacks
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap MySQL",
		Bootstrap: MySQLBootstrap,
	})
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Redis",
		Bootstrap: RedisBootstrap,
	})
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap RabbitMQ",
		Bootstrap: RabbitBootstrap,
	})
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Prometheus",
		Bootstrap: PrometheusBootstrap,
	})
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap HTTP Server",
		Bootstrap: WebServerBootstrap,
	})
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Boostrap Consul",
		Bootstrap: ConsulBootstrap,
	})
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Cron/Task Scheduler",
		Bootstrap: SchedulerBootstrap,
	})

	PreServerBootstrap(func(rail Rail) error {
		requestValidationEnabled = GetPropBool(PropServerRequestValidateEnabled)
		rail.Infof("Server request parameter validation enabled: %v", requestValidationEnabled)
		return nil
	})
}

// Replace the default ServerResultHandler
func SetServerResultHanlder(srh ServerResultHandler) error {
	if srh == nil {
		return NewErr("ServerResultHandler is nil")
	}
	serverResultHandler = srh
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

	logrus.Info("Triggering shutdown hook")
	for _, hook := range shutdownHook {
		hook()
	}
}

// Record server route
func recordHttpServerRoute(url string, method string, handlerName string, extra ...StrPair) {
	serverHttpRoutes = append(serverHttpRoutes, HttpRoute{
		Url:         url,
		Method:      method,
		HandlerName: handlerName,
		Extra:       MergeStrPairs(extra...),
	})
}

// Get recorded server routes (deprecated, use GetHttpRoutes() instead)
func GetRecordedHttpServerRoutes() []string {
	urls := []string{}
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
func RawGet(url string, handler RawTRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodGet, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewRawTRouteHandler(handler)) })
}

// Register POST request route (raw version)
func RawPost(url string, handler RawTRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodPost, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewRawTRouteHandler(handler)) })
}

// Register PUT request route (raw version)
func RawPut(url string, handler RawTRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodPut, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewRawTRouteHandler(handler)) })
}

// Register DELETE request route (raw version)
func RawDelete(url string, handler RawTRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodDelete, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewRawTRouteHandler(handler)) })
}

// Add RoutesRegistar for GET request.
//
// The result or error is wrapped in Resp automatically.
func Get(url string, handler TRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodGet, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for POST request.
//
// The result or error is wrapped in Resp automatically.
func Post(url string, handler TRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodPost, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func Put(url string, handler TRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodPut, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for DELETE request.
//
// The result and error are wrapped in Resp automatically as json.
func Delete(url string, handler TRouteHandler, extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodDelete, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for POST request with automatic payload binding.
//
// The result or error is wrapped in Resp automatically.
func IPost[Req any](url string, handler MappedTRouteHandler[Req], extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodPost, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewMappedTRouteHandler(handler)) })
}

// Add RoutesRegistar for GET request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json.
func IGet[Req any](url string, handler MappedTRouteHandler[Req], extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodGet, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewMappedTRouteHandler(handler)) })
}

// Add RoutesRegistar for DELETE request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json
func IDelete[Req any](url string, handler MappedTRouteHandler[Req], extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodDelete, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewMappedTRouteHandler(handler)) })
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func IPut[Req any](url string, handler MappedTRouteHandler[Req], extra ...StrPair) {
	recordHttpServerRoute(url, http.MethodPut, FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewMappedTRouteHandler(handler)) })
}

func addRoutesRegistar(reg routesRegistar) {
	routesRegiatarList = append(routesRegiatarList, reg)
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
func ConfigureLogging(rail Rail) {

	// determine the writer that we will use for logging (loggerOut and loggerErrOut)
	if ContainsProp(PropLoggingRollingFile) {
		loggerOut = BuildRollingLogFileWriter(NewRollingLogFileParam{
			Filename:   GetPropStr(PropLoggingRollingFile),
			MaxSize:    GetPropInt(PropLoggingRollingFileMaxSize), // megabytes
			MaxAge:     GetPropInt(PropLoggingRollingFileMaxAge),  //days
			MaxBackups: GetPropInt(PropLoggingRollingFileMaxBackups),
		})
		loggerErrOut = loggerOut
	}

	logrus.SetOutput(loggerOut)

	if HasProp(PropLoggingFile) {
		if level, ok := ParseLogLevel(GetPropStr(PropLoggingFile)); ok {
			logrus.SetLevel(level)
		}
	}
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
	AddShutdownHook(func() { MarkServerShuttingDown() })

	rail, cancel := rail.WithCancel()
	AddShutdownHook(func() { cancel() })

	// default way to load configuration
	DefaultReadConfig(args, rail)

	// configure logging
	ConfigureLogging(rail)

	appName := GetPropStr(PropAppName)
	if appName == "" {
		rail.Fatalf("Propertity '%s' is required", PropAppName)
	}

	rail.Infof("\n\n---------------------------------------------- starting %s -------------------------------------------------------\n", appName)
	rail.Infof("Miso Version: %s", MisoVersion)

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	if e := callPreServerBootstrapListeners(rail); e != nil {
		rail.Errorf("Error occurred while invoking pre server bootstrap callbacks, %v", e)
		return
	}

	// bootstrap components
	for _, sbc := range serverBootrapCallbacks {
		start := time.Now()
		if e := sbc.Bootstrap(rail); e != nil {
			rail.Errorf("Failed to bootstrap server component, %v", e)
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
	for _, registerRoute := range routesRegiatarList {
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
	logrus.Info("Shutting down http server gracefully")

	// set timeout for graceful shutdown
	timeout := GetPropInt(PropServerGracefulShutdownTimeSec)
	if timeout <= 0 {
		timeout = 30
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// shutdown web server with the timeout
	server.Shutdown(ctx)
	logrus.Infof("Http server exited")
}

// Resolve handler path for open api (it doesn't really affect anything, just a path prefix)
func OpenApiPath(relPath string) string {
	return ResolvePath(relPath, true)
}

// Resolve handler path for internal endpoints, (it doesn't really affect anything, just a path prefix)
func InternalApiPath(relPath string) string {
	return ResolvePath(relPath, false)
}

// Resolve handler path.
//
// deprecated.
func ResolvePath(relPath string, isOpenApi bool) string {
	if !strings.HasPrefix(relPath, "/") {
		relPath = "/" + relPath
	}

	if isOpenApi {
		return OPEN_API_PREFIX + relPath
	}

	return "/remote" + relPath
}

// Default Recovery func
func DefaultRecovery(c *gin.Context, e interface{}) {
	rail := BuildRail(c)
	rail.Errorf("Recovered from panic, %v", e)

	if err, ok := e.(error); ok {
		serverResultHandler(c, rail, nil, err)
		return
	}

	serverResultHandler(c, rail, nil, NewErr("Unknown error, please try again later"))
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
		propagatedKeys := append(GetPropagationKeys(), X_SPANID, X_TRACEID)

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

		if requestValidationEnabled {
			// validate request
			if e := Validate(req); e != nil {
				serverResultHandler(c, rail, nil, e)
				return
			}
		}

		// handle the requests
		res, err := handler(c, rail, req)

		// wrap result and error
		serverResultHandler(c, rail, res, err)
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
		serverResultHandler(c, rail, r, e)
	}
}

// Handle route's result
func ServerHandleResult(c *gin.Context, rail Rail, result any, err error) {
	serverResultHandler(c, rail, result, err)
}

func defaultHandleResult(c *gin.Context, rail Rail, r any, e error) {
	if e != nil {
		DispatchErrJson(c, rail, e)
		return
	}

	if r != nil {
		DispatchOkWData(c, r)
		return
	}
	DispatchOk(c)
}

// Must bind request payload to the given pointer, else panic
func MustBind(rail Rail, c *gin.Context, ptr any) {
	onFailed := func(err error) {
		rail.Errorf("Bind payload failed, %v", err)
		panic("Illegal Arguments")
	}

	// we now use jsoniter
	if c.ContentType() == gin.MIMEJSON {
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
func DispatchJson(c *gin.Context, body interface{}) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", applicationJson)

	err := jsoniter.NewEncoder(c.Writer).Encode(body)
	if err != nil {
		panic(err)
	}
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	DispatchJson(c, OkRespWData(data))
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, rail Rail, err error) {
	DispatchJson(c, WrapResp(nil, err, rail))
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	DispatchJson(c, ErrorResp(msg))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	DispatchJson(c, OkResp())
}

func MySQLBootstrap(rail Rail) error {
	if !IsMySqlEnabled() {
		return nil
	}

	if e := InitMySQLFromProp(); e != nil {
		return TraceErrf(e, "Failed to establish connection to MySQL")
	}
	return nil
}

func WebServerBootstrap(rail Rail) error {
	if !GetPropBool(PropServerEnabled) {
		return nil
	}
	rail.Info("Starting HTTP server")

	if GetPropBool(PropServerJsonNamingLowercase) {
		rail.Debug("HTTP Server using lowercase naming strategy for JSON processing.")
		extra.SetNamingStrategy(LowercaseNamingStrategy)
	}

	// Load propagation keys for tracing
	LoadPropagationKeyProp(rail)

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

	// register customer recovery func
	engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

	// register consul health check
	if IsConsulEnabled() && GetPropBool(PropConsulRegisterDefaultHealthcheck) {
		registerRouteForConsulHealthcheck(engine)
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

func PrometheusBootstrap(rail Rail) error {
	if !GetPropBool(PropMetricsEnabled) || !GetPropBool(PropServerEnabled) {
		return nil
	}

	handler := PrometheusHandler()
	RawGet(GetPropStr(PropPromRoute), func(c *gin.Context, rail Rail) {
		handler.ServeHTTP(c.Writer, c.Request)
	})
	return nil
}

func RabbitBootstrap(rail Rail) error {
	if !RabbitMQEnabled() {
		return nil
	}
	if e := StartRabbitMqClient(rail); e != nil {
		return TraceErrf(e, "Failed to establish connection to RabbitMQ")
	}
	return nil
}

func ConsulBootstrap(rail Rail) error {
	if !IsConsulEnabled() {
		return nil
	}

	// create consul client
	if _, e := GetConsulClient(); e != nil {
		return TraceErrf(e, "Failed to establish connection to Consul")
	}

	// deregister on shutdown
	AddShutdownHook(func() {
		if e := DeregisterService(); e != nil {
			rail.Errorf("Failed to deregister on Consul, %v", e)
		}
	})

	if e := RegisterService(); e != nil {
		return TraceErrf(e, "Failed to register on Consul")
	}
	return nil
}

func RedisBootstrap(rail Rail) error {
	if !IsRedisEnabled() {
		return nil
	}
	if _, e := InitRedisFromProp(rail); e != nil {
		return TraceErrf(e, "Failed to establish connection to Redis")
	}
	return nil
}

func SchedulerBootstrap(rail Rail) error {
	// distributed task scheduler has pending tasks and is enabled
	if IsTaskSchedulerPending() && !IsTaskSchedulingDisabled() {
		StartTaskSchedulerAsync(rail)
		rail.Info("Distributed Task Scheduler started")
		AddShutdownHook(func() { StopTaskScheduler() })
	} else if HasScheduler() {
		// cron scheduler, note that task scheduler internally wraps cron scheduler, we only starts one of them
		StartSchedulerAsync()
		rail.Info("Cron Scheduler started")
		AddShutdownHook(func() { StopScheduler() })
	}
	return nil
}

// Change first rune to lower case
func LowercaseNamingStrategy(name string) string {
	ru := []rune(name)
	if len(ru) < 1 {
		return name
	}
	ru[0] = unicode.ToLower(ru[0])
	return string(ru)
}

type GroupedRouteRegistar struct{ registerRoute func(baseUrl string) }
type RoutingGroup struct{ Base string }

// Group routes together to share the same base url.
func BaseRoute(baseUrl string) RoutingGroup {
	return RoutingGroup{Base: baseUrl}
}

func (rg RoutingGroup) Group(grouped ...GroupedRouteRegistar) {
	for i := range grouped {
		grouped[i].registerRoute(rg.Base)
	}
}

func GrpRawGet(url string, handler RawTRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			RawGet(baseUrl+url, handler, extra...)
		},
	}
}

func GrpRawPost(url string, handler RawTRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			RawPost(baseUrl+url, handler, extra...)
		},
	}

}

func GrpRawPut(url string, handler RawTRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			RawPut(baseUrl+url, handler, extra...)
		},
	}
}

func GrpRawDelete(url string, handler RawTRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			RawDelete(baseUrl+url, handler, extra...)
		},
	}
}

func GrpGet(url string, handler TRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			Get(baseUrl+url, handler, extra...)
		},
	}
}

func GrpPost(url string, handler TRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			Post(baseUrl+url, handler, extra...)
		},
	}
}

func GrpPut(url string, handler TRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			Put(baseUrl+url, handler, extra...)
		},
	}
}

func GrpDelete(url string, handler TRouteHandler, extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			Delete(baseUrl+url, handler, extra...)
		},
	}
}

func GrpIGet[T any](url string, handler MappedTRouteHandler[T], extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			IGet(baseUrl+url, handler, extra...)
		},
	}
}

func GrpIPost[T any](url string, handler MappedTRouteHandler[T], extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			IPost(baseUrl+url, handler, extra...)
		},
	}
}

func GrpIDelete[T any](url string, handler MappedTRouteHandler[T], extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			IDelete(baseUrl+url, handler, extra...)
		},
	}
}

func GrpIPut[T any](url string, handler MappedTRouteHandler[T], extra ...StrPair) GroupedRouteRegistar {
	return GroupedRouteRegistar{
		registerRoute: func(baseUrl string) {
			IPut(baseUrl+url, handler, extra...)
		},
	}
}
