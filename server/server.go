package server

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

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/consul"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/rabbitmq"
	"github.com/curtisnewbie/gocommon/redis"
	"github.com/curtisnewbie/gocommon/task"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Raw route handler.
type RawTRouteHandler func(c *gin.Context, ec common.ExecContext)

// Route handler.
//
// The returned result and error are automatically wrapped in a Resp object.
type TRouteHandler func(c *gin.Context, ec common.ExecContext) (any, error)

// Route handler.
//
// Request payload is automatically resolved to object t (e.g., from query parameters of JSON payload depending on the content-type).
//
// The returned result and error are automatically wrapped in a Resp object.
type ITRouteHandler[T any, V any] func(c *gin.Context, ec common.ExecContext, t T) (V, error)

type RoutesRegistar func(*gin.Engine)

type HttpRoute struct {
	Url         string
	Method      string
	Extra       map[string]any
	HandlerName string
}

const (
	OPEN_API_PREFIX = "/open/api"

	HTTP_GET    = "GET"
	HTTP_PUT    = "PUT"
	HTTP_POST   = "POST"
	HTTP_DELETE = "DELETE"
	HTTP_HEAD   = "HEAD"
)

var (
	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	routesRegiatarList []RoutesRegistar = []RoutesRegistar{}

	shuttingDown   bool         = false
	shutingDownRwm sync.RWMutex // rwmutex for shuttingDown

	shutdownHook []func()
	shmu         sync.Mutex // mutex for shutdownHook

	preServerBootstrapListener  []func(c common.ExecContext) error = []func(c common.ExecContext) error{}
	postServerBootstrapListener []func(c common.ExecContext) error = []func(c common.ExecContext) error{}
	serverHttpRoutes            []HttpRoute                        = []HttpRoute{}
)

func init() {
	common.SetDefProp(common.PROP_SERVER_ENABLED, true)
	common.SetDefProp(common.PROP_SERVER_HOST, "0.0.0.0")
	common.SetDefProp(common.PROP_SERVER_PORT, 8080)
	common.SetDefProp(common.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC, 5)
	common.SetDefProp(common.PROP_SERVER_PERF_ENABLED, false)
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
func recordHttpServerRoute(url string, method string, handlerName string, extra ...common.StrPair) {
	serverHttpRoutes = append(serverHttpRoutes, HttpRoute{
		Url:         url,
		Method:      method,
		HandlerName: handlerName,
		Extra:       common.MergeStrPairs(extra...),
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

// Register GET request route (raw version)
func RawGet(url string, handler RawTRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_GET, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewRawTRouteHandler(handler)) })
}

// Register POST request route (raw version)
func RawPost(url string, handler RawTRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_POST, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewRawTRouteHandler(handler)) })
}

// Register PUT request route (raw version)
func RawPut(url string, handler RawTRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_PUT, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewRawTRouteHandler(handler)) })
}

// Register DELETE request route (raw version)
func RawDelete(url string, handler RawTRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_DELETE, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewRawTRouteHandler(handler)) })
}

// Add RoutesRegistar for GET request.
//
// The result or error is wrapped in Resp automatically.
func Get(url string, handler TRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_GET, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for POST request.
//
// The result or error is wrapped in Resp automatically.
func Post(url string, handler TRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_POST, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func Put(url string, handler TRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_PUT, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for DELETE request.
//
// The result and error are wrapped in Resp automatically as json.
func Delete(url string, handler TRouteHandler, extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_DELETE, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for POST request with automatic payload binding.
//
// The result or error is wrapped in Resp automatically.
func IPost[T any, V any](url string, handler ITRouteHandler[T, V], extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_POST, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewITRouteHandler(handler)) })
}

// Add RoutesRegistar for GET request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json.
func IGet[T any, V any](url string, handler ITRouteHandler[T, V], extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_POST, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewITRouteHandler(handler)) })
}

// Add RoutesRegistar for DELETE request with automatic payload binding.
//
// The result and error are wrapped in Resp automatically as json
func IDelete[T any, V any](url string, handler ITRouteHandler[T, V], extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_DELETE, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewITRouteHandler(handler)) })
}

// Add RoutesRegistar for PUT request.
//
// The result and error are wrapped in Resp automatically as json.
func IPut[T any, V any](url string, handler ITRouteHandler[T, V], extra ...common.StrPair) {
	recordHttpServerRoute(url, HTTP_PUT, common.FuncName(handler), extra...)
	addRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewITRouteHandler(handler)) })
}

// Add RoutesRegistar
func addRoutesRegistar(reg RoutesRegistar) {
	routesRegiatarList = append(routesRegiatarList, reg)
}

// Register GIN route for consul healthcheck
func registerRouteForConsulHealthcheck(router *gin.Engine) {
	router.GET(common.GetPropStr(common.PROP_CONSUL_HEALTHCHECK_URL), consul.DefaultHealthCheck)
}

func startHttpServer(ctx context.Context, server *http.Server) {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("http.Server ListenAndServe: %s", err)
	}
}

func createHttpServer(router http.Handler) *http.Server {
	addr := fmt.Sprintf("%s:%s", common.GetPropStr(common.PROP_SERVER_HOST), common.GetPropStr(common.PROP_SERVER_PORT))
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	return server
}

// Configurae Logging, e.g., logger's output, log level, etc
func ConfigureLogging(c common.ExecContext) {

	// determine the writer that we will use for logging (loggerOut and loggerErrOut)
	if common.ContainsProp(common.PROP_LOGGING_ROLLING_FILE) {
		loggerOut = common.BuildRollingLogFileWriter(common.GetPropStr(common.PROP_LOGGING_ROLLING_FILE))
		loggerErrOut = loggerOut
	}

	if !common.IsProdMode() {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.SetOutput(loggerOut)

	if common.HasProp(common.PROP_LOGGING_LEVEL) {
		if level, ok := parseLogLevel(common.GetPropStr(common.PROP_LOGGING_LEVEL)); ok {
			logrus.SetLevel(level)
		}
	}
}

func parseLogLevel(logLevel string) (logrus.Level, bool) {
	logLevel = strings.ToUpper(logLevel)
	switch logLevel {
	case "INFO":
		return logrus.InfoLevel, true
	case "DEBUG":
		return logrus.DebugLevel, true
	case "WARN":
		return logrus.WarnLevel, true
	case "ERROR":
		return logrus.ErrorLevel, true
	case "TRACE":
		return logrus.TraceLevel, true
	case "FATAL":
		return logrus.FatalLevel, true
	case "PANIC":
		return logrus.PanicLevel, true
	}
	return logrus.InfoLevel, false
}

func callPostServerBootstrapListeners(c common.ExecContext) error {
	for _, callback := range postServerBootstrapListener {
		if e := callback(c); e != nil {
			return e
		}
	}
	return nil
}

// Add listener that is invoked when server is finally bootstrapped
func PostServerBootstrapped(callback func(c common.ExecContext) error) {
	if callback == nil {
		return
	}
	postServerBootstrapListener = append(postServerBootstrapListener, callback)
}

// Add listener that is invoked before the server is fully bootstrapped
func PreServerBootstrap(callback func(c common.ExecContext) error) {
	if callback == nil {
		return
	}
	preServerBootstrapListener = append(preServerBootstrapListener, callback)
}

func callPreServerBootstrapListeners(c common.ExecContext) error {
	for _, callback := range preServerBootstrapListener {
		if e := callback(c); e != nil {
			return e
		}
	}
	return nil
}

/*
Bootstrap server

This func will attempt to create http server, connect to MySQL, Redis or Consul based on the configuration loaded.

It also handles service registration/de-registration on Consul before Gin bootstraped and after
SIGTERM/INTERRUPT signals are received.

Graceful shutdown for the http server is also enabled and can be configured through props.

To configure server, MySQL, Redis, Consul and so on, see PROPS_* in prop.go.

It's also possible to register callbacks that are triggered before/after server bootstrap


	server.PreServerBootstrapped(func(c common.ExecContext) error {
		// do something right after configuration being loaded, but server hasn't been bootstraped yet
	});

	server.PostServerBootstrapped(func(c common.ExecContext) error {
		// do something after the server bootstrap
	});

	// start the server
	server.BootstrapServer(os.Args)

*/
func BootstrapServer(args []string) {
	var c common.ExecContext = common.EmptyExecContext()

	start := time.Now().UnixMilli()
	defer triggerShutdownHook()
	AddShutdownHook(func() { MarkServerShuttingDown() })

	ctx, cancel := context.WithCancel(context.Background())
	AddShutdownHook(func() { cancel() })

	// default way to load configuration
	common.DefaultReadConfig(args, c)

	// configure logging
	ConfigureLogging(c)

	appName := common.GetPropStr(common.PROP_APP_NAME)
	if appName == "" {
		c.Log.Fatalf("Propertity '%s' is required", common.PROP_APP_NAME)
	}

	c.Log.Infof("\n\n---------------------------------------------- starting %s -------------------------------------------------------\n", appName)
	c.Log.Infof("Gocommon Version: %s", common.GOCOMMON_VERSION)

	// invoke callbacks to setup server, sometime we need to setup stuff right after the configuration being loaded
	if e := callPreServerBootstrapListeners(c); e != nil {
		c.Log.Fatalf("Error occurred while invoking pre server bootstrap callbacks, %v", e)
	}

	// mysql
	if mysql.IsMySqlEnabled() {
		if e := mysql.InitMySqlFromProp(); e != nil {
			c.Log.Fatalf("Failed to establish connection to MySQL, %v", e)
		}
	}

	// redis
	if redis.IsRedisEnabled() {
		if _, e := redis.InitRedisFromProp(); e != nil {
			c.Log.Fatalf("Failed to establish connection to Redis, %v", e)
		}
	}

	// rabbitmq
	if rabbitmq.IsEnabled() {
		if e := rabbitmq.StartRabbitMqClient(ctx); e != nil {
			c.Log.Fatalf("Failed to establish connection to RabbitMQ, %v", e)
		}
	}

	// web server
	if common.GetPropBool(common.PROP_SERVER_ENABLED) {
		c.Log.Info("Starting http server")

		// Load propagation keys for tracing
		common.LoadPropagationKeyProp(c)

		// always set to releaseMode
		gin.SetMode(gin.ReleaseMode)

		// gin engine
		engine := gin.New()
		engine.Use(TraceMiddleware())

		if !common.IsProdMode() {
			engine.Use(gin.Logger()) // default logger for debugging
		}

		if common.GetPropBool(common.PROP_SERVER_PERF_ENABLED) {
			engine.Use(PerfMiddleware())
		}

		// register customer recovery func
		engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

		// register consul health check
		if consul.IsConsulEnabled() {
			registerRouteForConsulHealthcheck(engine)
		}

		// register http routes
		registerServerRoutes(c, engine)

		// start the http server
		server := createHttpServer(engine)
		c.Log.Infof("Serving HTTP on %s", server.Addr)
		go startHttpServer(ctx, server)

		AddShutdownHook(func() { shutdownHttpServer(server) })
	}

	// consul
	if consul.IsConsulEnabled() {

		// create consul client
		if _, e := consul.GetConsulClient(); e != nil {
			c.Log.Fatalf("Failed to establish connection to Consul, %v", e)
		}

		// deregister on shutdown
		AddShutdownHook(func() {
			if e := consul.DeregisterService(); e != nil {
				c.Log.Errorf("Failed to deregister on Consul, %v", e)
			}
		})

		if e := consul.RegisterService(); e != nil {
			c.Log.Fatalf("Failed to register on Consul, %v", e)
		}
	}

	// distributed task scheduler
	if task.IsTaskSchedulerPending() {
		if !task.IsTaskSchedulingDisabled() {
			task.StartTaskSchedulerAsync()
			c.Log.Info("TaskScheduler started")
			AddShutdownHook(func() { task.StopTaskScheduler() })
		}
	} else {
		// cron scheduler, note that task scheduler internally wraps cron scheduler, we only starts one of them
		if common.HasScheduler() {
			common.StartSchedulerAsync()
			c.Log.Info("Scheduler started")
			AddShutdownHook(func() { common.StopScheduler() })
		}
	}

	end := time.Now().UnixMilli()
	c.Log.Infof("\n\n---------------------------------------------- %s started (took: %dms) --------------------------------------------\n", appName, end-start)

	// invoke listener for serverBootstraped event
	if e := callPostServerBootstrapListeners(c); e != nil {
		c.Log.Fatalf("Error occurred while invoking post server bootstrap callbacks, %v", e)
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}

// Register http routes on gin.Engine
func registerServerRoutes(c common.ExecContext, engine *gin.Engine) {
	// no route
	engine.NoRoute(func(ctx *gin.Context) {
		c := BuildExecContext(ctx)
		c.Log.Warnf("NoRoute for %s '%s', returning 404", ctx.Request.Method, ctx.Request.RequestURI)
		ctx.AbortWithStatus(404)
	})

	// register custom routes
	for _, registerRoute := range routesRegiatarList {
		registerRoute(engine)
	}

	for _, r := range GetHttpRoutes() {
		c.Log.Infof("%-6s %-45s --> %s", r.Method, r.Url, r.HandlerName)
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
	timeout := common.GetPropInt(common.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC)
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
	if err, ok := e.(error); ok {
		DispatchErrJson(c, err)
		return
	}
	if msg, ok := e.(string); ok {
		DispatchErrMsgJson(c, msg)
		return
	}

	DispatchErrJson(c, common.NewWebErr("Unknown error, please try again later"))
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

// Create a predicate that tries to match the given url against the pattern (excluding query parameters)
func UrlMatchPredicate(pattern string) common.Predicate[string] {
	return func(url string) bool {
		uurl := url
		j := strings.Index(uurl, "?")
		if j > -1 {
			rw := common.GetRuneWrp(uurl)
			uurl = rw.Substr(0, j)
		}
		return strings.EqualFold(uurl, pattern)
	}
}

// Add url based route authentication whitelist
func AddUrlBasedRouteAuthWhitelist(url string) {
	AddRouteAuthWhitelist(UrlMatchPredicate(url))
}

// Add route authentication whitelist predicate
//
// deprecated, this does nothing
func AddRouteAuthWhitelist(pred common.Predicate[string]) {
	// does nothing, for backword compatibility
}

// Check whether url is in route whitelist
//
// deprecated, will always return false
func IsRouteWhitelist(url string) bool {
	// keep this for backword compatibility
	return false
}

// Perf Middleware that calculates how much time each request takes
func PerfMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer common.LTimeOp(time.Now(), ctx.Request.RequestURI)
		ctx.Next()
	}
}

// Tracing Middleware
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// propagate tracing key/value pairs with context
		ctx := c.Request.Context()
		propagatedKeys := append(common.GetPropagationKeys(), common.X_SPANID, common.X_TRACEID)

		for _, k := range propagatedKeys {
			if h := c.GetHeader(k); h != "" {
				ctx = context.WithValue(ctx, k, h) //lint:ignore SA1029 keys must be exposed for the users to retrieve the values
			}
		}

		// replace the context
		c.Request = c.Request.WithContext(ctx)

		// follow the chain
		c.Next()
	}
}

// Build ExecContext from the Gin's request context
func BuildExecContext(c *gin.Context) common.ExecContext {
	user, _ := ExtractUser(c)
	return common.NewExecContext(c.Request.Context(), user)
}

// Build route handler with the required payload object, context, user (optional, may be nil), and logger prepared
func NewITRouteHandler[T any, V any](handler ITRouteHandler[T, V]) func(c *gin.Context) {
	return func(c *gin.Context) {
		user, _ := ExtractUser(c) // optional
		ctx := c.Request.Context()

		// bind to payload boject
		var t T
		MustBind(c, &t)

		// validate request
		if e := common.Validate(t); e != nil {
			HandleResult(c, nil, e)
			return
		}

		// actual handling
		r, e := handler(c, common.NewExecContext(ctx, user), t)

		// wrap result and error
		HandleResult(c, r, e)
	}
}

// Build RawTRouteHandler with context, user (optional, may be nil), and logger prepared
func NewRawTRouteHandler(handler RawTRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		handler(c, BuildExecContext(c))
	}
}

// Build TRouteHandler with context, user (optional, may be nil), and logger prepared
func NewTRouteHandler(handler TRouteHandler) func(c *gin.Context) {
	return func(c *gin.Context) {
		user, _ := ExtractUser(c) // optional
		ctx := c.Request.Context()
		r, e := handler(c, common.NewExecContext(ctx, user))
		HandleResult(c, r, e)
	}
}

// Handle route's result
func HandleResult(c *gin.Context, r any, e error) {
	if e != nil {
		DispatchErrJson(c, e)
		return
	}

	if r != nil {
		DispatchOkWData(c, r)
		return
	}
	DispatchOk(c)
}

// Must bind json content to the given pointer, else panic
func MustBindJson(c *gin.Context, ptr any) {
	if err := c.ShouldBindJSON(ptr); err != nil {
		common.TraceLogger(c.Request.Context()).Errorf("Bind Json failed, %v", err)
		panic("Illegal Arguments")
	}
}

// Must bind request payload to the given pointer, else panic
func MustBind(c *gin.Context, ptr any) {
	if err := c.ShouldBind(ptr); err != nil {
		common.TraceLogger(c.Request.Context()).Errorf("Bind payload failed, %v", err)
		panic("Illegal Arguments")
	}
}

// Dispatch a json response
func DispatchJson(c *gin.Context, body interface{}) {
	c.JSON(http.StatusOK, body)
}

// Dispatch error response in json format
func DispatchErrJson(c *gin.Context, err error) {
	c.JSON(http.StatusOK, common.WrapResp(nil, err))
}

// Dispatch error response in json format
func DispatchErrMsgJson(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, common.ErrorResp(msg))
}

// Dispatch an ok response in json format
func DispatchOk(c *gin.Context) {
	c.JSON(http.StatusOK, common.OkResp())
}

// Dispatch an ok response with data in json format
func DispatchOkWData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, common.OkRespWData(data))
}

// Extract user from request headers, panic if failed
func RequireUser(c *gin.Context) *common.User {
	u, e := ExtractUser(c)
	if e != nil {
		panic(e)
	}
	return u
}

// Extract role from request header
//
// return:
//
//	role, isOk
//
// deprecated
func Role(c *gin.Context) (string, bool) {
	id := c.GetHeader("role")
	if id == "" {
		return "", false
	}
	return id, true
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

/* Extract common.User from request headers */
func ExtractUser(c *gin.Context) (*common.User, error) {
	id := c.GetHeader("id")
	if id == "" {
		return nil, common.NewWebErr("Please sign up first")
	}

	var services []string
	servicesStr := c.GetHeader("services")
	if servicesStr == "" {
		services = make([]string, 0)
	} else {
		services = strings.Split(servicesStr, ",")
	}

	return &common.User{
		UserId:   id,
		Username: c.GetHeader("username"),
		UserNo:   c.GetHeader("userno"),
		Role:     c.GetHeader("role"),
		RoleNo:   c.GetHeader("roleNo"),
		Services: services,
	}, nil
}

// Check whether current request is authenticated
func IsRequestAuthenticated(c *gin.Context) bool {
	id := c.GetHeader("id")
	return id != ""
}
