package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/curtisnewbie/miso/consul"
	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/metrics"
	"github.com/curtisnewbie/miso/mysql"
	"github.com/curtisnewbie/miso/rabbitmq"
	"github.com/curtisnewbie/miso/redis"
	"github.com/curtisnewbie/miso/task"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Raw version of traced route handler.
type RawTRouteHandler func(c *gin.Context, rail core.Rail)

// Traced route handler.
type TRouteHandler func(c *gin.Context, rail core.Rail) (any, error)

/*
Traced and parameters mapped route handler.

T should be a struct, where all fields are automatically mapped from the request using different tags.

  - json
  - xml
  - form
*/
type MappedTRouteHandler[Req any, Res any] func(c *gin.Context, rail core.Rail, req Req) (Res, error)

type routesRegistar func(*gin.Engine)

type HttpRoute struct {
	Url         string
	Method      string
	Extra       map[string]any
	HandlerName string
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
	serverBootrapCallbacks []func(ctx context.Context, r core.Rail) error = []func(context.Context, core.Rail) error{}

	// listener for events trigger before server components being bootstrapped
	preServerBootstrapListener []func(r core.Rail) error = []func(r core.Rail) error{}
	// listener for events trigger after server components bootstrapped
	postServerBootstrapListener []func(r core.Rail) error = []func(r core.Rail) error{}

	// all http methods
	anyHttpMethods = []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect,
		http.MethodTrace,
	}

	// channel for signaling server shutdown
	manualSigQuit = make(chan int, 1)
)

func init() {
	core.SetDefProp(core.PROP_SERVER_ENABLED, true)
	core.SetDefProp(core.PROP_SERVER_HOST, "0.0.0.0")
	core.SetDefProp(core.PROP_SERVER_PORT, 8080)
	core.SetDefProp(core.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC, 5)
	core.SetDefProp(core.PROP_SERVER_PERF_ENABLED, false)
	core.SetDefProp(core.PROP_SERVER_PROPAGATE_INBOUND_TRACE, true)

	// mysql
	RegisterBootstrapCallback(func(_ context.Context, rail core.Rail) error {
		if !mysql.IsMySqlEnabled() {
			return nil
		}

		defer core.DebugTimeOp(rail, time.Now(), "Connect MySQL")
		if e := mysql.InitMySqlFromProp(); e != nil {
			return core.TraceErrf(e, "Failed to establish connection to MySQL")
		}
		return nil
	})

	// redis
	RegisterBootstrapCallback(func(_ context.Context, rail core.Rail) error {
		if !redis.IsRedisEnabled() {
			return nil
		}
		defer core.DebugTimeOp(rail, time.Now(), "Connect Redis")
		if _, e := redis.InitRedisFromProp(); e != nil {
			return core.TraceErrf(e, "Failed to establish connection to Redis")
		}
		return nil
	})

	// rabbitmq
	RegisterBootstrapCallback(func(ctx context.Context, rail core.Rail) error {
		if !rabbitmq.IsEnabled() {
			return nil
		}
		defer core.DebugTimeOp(rail, time.Now(), "Connect RabbitMQ")
		if e := rabbitmq.StartRabbitMqClient(ctx); e != nil {
			return core.TraceErrf(e, "Failed to establish connection to RabbitMQ")
		}
		return nil
	})

	// prometheus
	RegisterBootstrapCallback(func(ctx context.Context, rail core.Rail) error {
		if !core.GetPropBool(core.PROP_METRICS_ENABLED) || !core.GetPropBool(core.PROP_SERVER_ENABLED) {
			return nil
		}

		defer core.DebugTimeOp(rail, time.Now(), "Prepare Prometheus metrics endpoint")
		handler := metrics.PrometheusHandler()
		RawGet(core.GetPropStr(core.PROP_PROM_ROUTE), func(c *gin.Context, rail core.Rail) {
			handler.ServeHTTP(c.Writer, c.Request)
		})
		return nil
	})

	// web server
	RegisterBootstrapCallback(func(ctx context.Context, rail core.Rail) error {
		if !core.GetPropBool(core.PROP_SERVER_ENABLED) {
			return nil
		}
		defer core.DebugTimeOp(rail, time.Now(), "Prepare HTTP server")
		rail.Info("Starting HTTP server")

		// Load propagation keys for tracing
		core.LoadPropagationKeyProp(rail)

		// always set to releaseMode
		gin.SetMode(gin.ReleaseMode)

		// gin engine
		engine := gin.New()
		engine.Use(TraceMiddleware())

		if !core.IsProdMode() && core.IsDebugLevel() {
			engine.Use(gin.Logger()) // gin's default logger for debugging
		}

		if core.GetPropBool(core.PROP_SERVER_PERF_ENABLED) {
			engine.Use(PerfMiddleware())
		}

		// register customer recovery func
		engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

		// register consul health check
		if consul.IsConsulEnabled() && core.GetPropBool(core.PROP_CONSUL_REGISTER_DEFAULT_HEALTHCHECK) {
			registerRouteForConsulHealthcheck(engine)
		}

		// register http routes
		registerServerRoutes(rail, engine)

		// start the http server
		server := createHttpServer(engine)
		rail.Infof("Serving HTTP on %s", server.Addr)
		go startHttpServer(ctx, server)

		AddShutdownHook(func() { shutdownHttpServer(server) })
		return nil
	})

	// consul
	RegisterBootstrapCallback(func(_ context.Context, rail core.Rail) error {
		if !consul.IsConsulEnabled() {
			return nil
		}
		defer core.DebugTimeOp(rail, time.Now(), "Connect Consul")

		// create consul client
		if _, e := consul.GetConsulClient(); e != nil {
			return core.TraceErrf(e, "Failed to establish connection to Consul")
		}

		// deregister on shutdown
		AddShutdownHook(func() {
			if e := consul.DeregisterService(); e != nil {
				rail.Errorf("Failed to deregister on Consul, %v", e)
			}
		})

		if e := consul.RegisterService(); e != nil {
			return core.TraceErrf(e, "Failed to register on Consul")
		}
		return nil
	})

	// cron schedulers and distributed task scheduler
	RegisterBootstrapCallback(func(ctx context.Context, rail core.Rail) error {
		defer core.DebugTimeOp(rail, time.Now(), "Prepare cron scheduler and distributed task scheduler")

		// distributed task scheduler has pending tasks and is enabled
		if task.IsTaskSchedulerPending() && !task.IsTaskSchedulingDisabled() {
			task.StartTaskSchedulerAsync()
			rail.Info("Distributed Task Scheduler started")
			AddShutdownHook(func() { task.StopTaskScheduler() })
		} else if core.HasScheduler() {
			// cron scheduler, note that task scheduler internally wraps cron scheduler, we only starts one of them
			core.StartSchedulerAsync()
			rail.Info("Scheduler started")
			AddShutdownHook(func() { core.StopScheduler() })
		}
		return nil
	})
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
func recordHttpServerRoute(url string, method string, handlerName string, extra ...core.StrPair) {
	serverHttpRoutes = append(serverHttpRoutes, HttpRoute{
		Url:         url,
		Method:      method,
		HandlerName: handlerName,
		Extra:       core.MergeStrPairs(extra...),
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
func RawAny(url string, handler RawTRouteHandler, extra ...core.StrPair) {
	for i := range anyHttpMethods {
		recordHttpServerRoute(url, anyHttpMethods[i], core.FuncName(handler), extra...)
	}
	addRoutesRegistar(func(e *gin.Engine) { e.Any(url, NewRawTRouteHandler(handler)) })
}

// Register GET request route (raw version)
func RawGet(url string, handler RawTRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodGet, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewRawTRouteHandler(handler)) })
}

// Register POST request route (raw version)
func RawPost(url string, handler RawTRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodPost, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewRawTRouteHandler(handler)) })
}

// Register PUT request route (raw version)
func RawPut(url string, handler RawTRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodPut, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewRawTRouteHandler(handler)) })
}

// Register DELETE request route (raw version)
func RawDelete(url string, handler RawTRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodDelete, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewRawTRouteHandler(handler)) })
}

// Add RoutesRegistar for GET request.
//
// The result or error is wrapped in Resp automatically.
func Get(url string, handler TRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodGet, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for POST request.
//
// The result or error is wrapped in Resp automatically.
func Post(url string, handler TRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodPost, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func Put(url string, handler TRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodPut, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for DELETE request.
//
// The result and error are wrapped in Resp automatically as json.
func Delete(url string, handler TRouteHandler, extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodDelete, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for POST request with automatic payload binding.
//
// The result or error is wrapped in Resp automatically.
func IPost[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res], extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodPost, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewMappedTRouteHandler(handler)) })
}

// Add RoutesRegistar for GET request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json.
func IGet[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res], extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodGet, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewMappedTRouteHandler(handler)) })
}

// Add RoutesRegistar for DELETE request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json
func IDelete[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res], extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodDelete, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewMappedTRouteHandler(handler)) })
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func IPut[Req any, Res any](url string, handler MappedTRouteHandler[Req, Res], extra ...core.StrPair) {
	recordHttpServerRoute(url, http.MethodPut, core.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewMappedTRouteHandler(handler)) })
}

func addRoutesRegistar(reg routesRegistar) {
	routesRegiatarList = append(routesRegiatarList, reg)
}

// Register GIN route for consul healthcheck
func registerRouteForConsulHealthcheck(router *gin.Engine) {
	router.GET(core.GetPropStr(core.PROP_CONSUL_HEALTHCHECK_URL), consul.DefaultHealthCheck)
}

func startHttpServer(ctx context.Context, server *http.Server) {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("http.Server ListenAndServe: %s", err)
	}
}

func createHttpServer(router http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%s", core.GetPropStr(core.PROP_SERVER_HOST), core.GetPropStr(core.PROP_SERVER_PORT))
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	return server
}

// Configure logging level and output target based on loaded configuration.
func ConfigureLogging(rail core.Rail) {

	// determine the writer that we will use for logging (loggerOut and loggerErrOut)
	if core.ContainsProp(core.PROP_LOGGING_ROLLING_FILE) {
		loggerOut = core.BuildRollingLogFileWriter(core.GetPropStr(core.PROP_LOGGING_ROLLING_FILE))
		loggerErrOut = loggerOut
	}

	logrus.SetOutput(loggerOut)

	if core.HasProp(core.PROP_LOGGING_LEVEL) {
		if level, ok := core.ParseLogLevel(core.GetPropStr(core.PROP_LOGGING_LEVEL)); ok {
			logrus.SetLevel(level)
		}
	}
}

func callPostServerBootstrapListeners(rail core.Rail) error {
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
func PostServerBootstrapped(callback func(rail core.Rail) error) {
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
func PreServerBootstrap(callback func(rail core.Rail) error) {
	if callback == nil {
		return
	}
	preServerBootstrapListener = append(preServerBootstrapListener, callback)
}

func callPreServerBootstrapListeners(rail core.Rail) error {
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
//
// e.g.,
//
//	RegisterBootstrapCallback(func(_ context.Context, c core.Rail) error {
//		if !consul.IsConsulEnabled() {
//			return nil
//		}
//
//		// create consul client
//		if _, e := consul.GetConsulClient(); e != nil {
//			return core.TraceErrf(e, "Failed to establish connection to Consul")
//		}
//
//		// deregister on shutdown
//		AddShutdownHook(func() {
//			if e := consul.DeregisterService(); e != nil {
//				c.Errorf("Failed to deregister on Consul, %v", e)
//			}
//		})
//
//		if e := consul.RegisterService(); e != nil {
//			return core.TraceErrf(e, "Failed to register on Consul")
//		}
//		return nil
//	})
func RegisterBootstrapCallback(bootstrapComponent func(ctx context.Context, rail core.Rail) error) {
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

	server.PreServerBootstrap(func(c core.Rail) error {
		// do something right after configuration being loaded, but server hasn't been bootstraped yet
	});

	server.PostServerBootstrapped(func(c core.Rail) error {
		// do something after the server bootstrap
	});

	// start the server
	server.BootstrapServer(os.Args)
*/
func BootstrapServer(args []string) {
	var c core.Rail = core.EmptyRail()

	start := time.Now().UnixMilli()
	defer triggerShutdownHook()
	AddShutdownHook(func() { MarkServerShuttingDown() })

	ctx, cancel := context.WithCancel(context.Background())
	AddShutdownHook(func() { cancel() })

	// default way to load configuration
	core.DefaultReadConfig(args, c)

	// configure logging
	ConfigureLogging(c)

	appName := core.GetPropStr(core.PROP_APP_NAME)
	if appName == "" {
		c.Fatalf("Propertity '%s' is required", core.PROP_APP_NAME)
	}

	c.Infof("\n\n---------------------------------------------- starting %s -------------------------------------------------------\n", appName)
	c.Infof("Miso Version: %s", core.MisoVersion)

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	if e := callPreServerBootstrapListeners(c); e != nil {
		c.Fatalf("Error occurred while invoking pre server bootstrap callbacks, %v", e)
	}

	// bootstrap components
	for _, bootstrap := range serverBootrapCallbacks {
		if e := bootstrap(ctx, c); e != nil {
			c.Fatalf("Failed to bootstrap server component, %v", e)
		}
	}

	end := time.Now().UnixMilli()
	c.Infof("\n\n---------------------------------------------- %s started (took: %dms) --------------------------------------------\n", appName, end-start)

	// invoke listener for serverBootstraped event
	if e := callPostServerBootstrapListeners(c); e != nil {
		c.Fatalf("Error occurred while invoking post server bootstrap callbacks, %v", e)
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	osSigQuit := make(chan os.Signal, 2)
	signal.Notify(osSigQuit, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-osSigQuit:
		c.Infof("Received OS signal: %v, exiting", sig)
	case <-manualSigQuit: // or wait for maunal shutdown signal
		c.Infof("Received manual shutdown signal, exiting")
	}
}

// Shutdown server
func Shutdown() {
	manualSigQuit <- 1
}

// Register http routes on gin.Engine
func registerServerRoutes(c core.Rail, engine *gin.Engine) {
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
	timeout := core.GetPropInt(core.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC)
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
		DispatchErrJson(c, rail, err)
		return
	}

	DispatchErrJson(c, rail, core.NewWebErr("Unknown error, please try again later"))
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

// Perf Middleware that calculates how much time each request takes
func PerfMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next() // continue the handler chain
		core.TraceLogger(ctx).Infof("%-6v %-60v [%s]", ctx.Request.Method, ctx.Request.RequestURI, time.Since(start))
	}
}

// Tracing Middleware
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// propagate tracing key/value pairs with context
		ctx := c.Request.Context()
		propagatedKeys := append(core.GetPropagationKeys(), core.X_SPANID, core.X_TRACEID)

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
func BuildRail(c *gin.Context) core.Rail {
	if !core.GetPropBool(core.PROP_SERVER_PROPAGATE_INBOUND_TRACE) {
		return core.EmptyRail()
	}

	if c.Keys == nil {
		c.Keys = map[string]any{}
	}

	tracked := core.GetPropagationKeys()
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
	rail := core.NewRail(ctx)

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
func NewMappedTRouteHandler[Req any, Res any](handler MappedTRouteHandler[Req, Res]) func(c *gin.Context) {
	return func(c *gin.Context) {
		rail := core.NewRail(c)

		// bind to payload boject
		var req Req
		MustBind(c, &req)

		// validate request
		if e := core.Validate(req); e != nil {
			HandleResult(c, rail, nil, e)
			return
		}

		// handle the requests
		res, err := handler(c, rail, req)

		// wrap result and error
		HandleResult(c, rail, res, err)
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
		HandleResult(c, rail, r, e)
	}
}

// Handle route's result
func HandleResult(c *gin.Context, rail core.Rail, r any, e error) {
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
func MustBind(c *gin.Context, ptr any) {
	if err := c.ShouldBind(ptr); err != nil {
		core.TraceLogger(c.Request.Context()).Errorf("Bind payload failed, %v", err)
		panic("Illegal Arguments")
	}
}

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, body)
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, rail core.Rail, err error) {
	c.JSON(http.StatusOK, core.WrapResp(nil, err, rail))
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, core.ErrorResp(msg))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	c.JSON(http.StatusOK, core.OkResp())
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, core.OkRespWData(data))
}

// Extract userNo from request header
//
// return:
//
//	userNo, isOk
func UserNo(c *gin.Context) (string, bool) {
	id := c.GetHeader("userno")
	if id == "" {
		return "", false
	}
	return id, true
}

// Extract user id from request header
//
// return:
//
//	userId, isOk
func UserId(c *gin.Context) (string, bool) {
	id := c.GetHeader("id")
	if id == "" {
		return "", false
	}
	return id, true
}

/* Extract core.User from request headers */
func ExtractUser(c *gin.Context) core.User {
	idHeader := c.GetHeader("id")
	if idHeader == "" {
		return core.NilUser()
	}
	id, err := strconv.Atoi(idHeader)
	if err != nil {
		id = 0
	}

	return core.User{
		UserId:   id,
		Username: c.GetHeader("username"),
		UserNo:   c.GetHeader("userno"),
		RoleNo:   c.GetHeader("roleno"),
		IsNil:    false,
	}
}

// Check whether current request is authenticated
func IsRequestAuthenticated(c *gin.Context) bool {
	id := c.GetHeader("id")
	return id != ""
}
