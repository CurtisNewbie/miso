package server

import (
	"context"
	"fmt"
	"io"
	"log"
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
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Routes registar
type RoutesRegistar func(*gin.Engine)

const (
	OPEN_API_PREFIX = "/open/api"
)

var (
	loggerOut    io.Writer = os.Stdout
	loggerErrOut io.Writer = os.Stderr

	routesRegiatarList []RoutesRegistar = []RoutesRegistar{}

	shuttingDown   bool = false
	shutingDownRwm sync.RWMutex

	shutdownHook []func()
	shmu         sync.Mutex

	routesAuthWhitelist []common.Predicate[string] = []common.Predicate[string]{}
	rawlmu              sync.RWMutex
)

func init() {
	common.SetDefProp(common.PROP_SERVER_WEB_ENABLED, true)
	common.SetDefProp(common.PROP_SERVER_HOST, "localhost")
	common.SetDefProp(common.PROP_SERVER_PORT, 8080)
	common.SetDefProp(common.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC, 5)
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

// Register GET request route, route is whitelisted, no authentication requires
func PubGet(url string, handlers ...gin.HandlerFunc) {
	AddUrlBasedRouteAuthWhitelist(url)
	AddRoutesRegistar(func(e *gin.Engine) { e.GET(url, handlers...) })
}

// Register POST request route, route is whitelisted, no authentication requires
func PubPost(url string, handlers ...gin.HandlerFunc) {
	AddUrlBasedRouteAuthWhitelist(url)
	AddRoutesRegistar(func(e *gin.Engine) { e.POST(url, handlers...) })
}

// Register PUT request route, route is whitelisted, no authentication requires
func PubPut(url string, handlers ...gin.HandlerFunc) {
	AddUrlBasedRouteAuthWhitelist(url)
	AddRoutesRegistar(func(e *gin.Engine) { e.PUT(url, handlers...) })
}

// Register DELETE request route, route is whitelisted, no authentication requires
func PubDelete(url string, handlers ...gin.HandlerFunc) {
	AddUrlBasedRouteAuthWhitelist(url)
	AddRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, handlers...) })
}

// Add RoutesRegistar for Get request
func Get(url string, handler TRouteHandler) {
	AddRoutesRegistar(func(e *gin.Engine) { e.GET(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for Post request
func Post(url string, handler TRouteHandler) {
	AddRoutesRegistar(func(e *gin.Engine) { e.POST(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for Put request
func Put(url string, handler TRouteHandler) {
	AddRoutesRegistar(func(e *gin.Engine) { e.PUT(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar for Delete request
func Delete(url string, handler TRouteHandler) {
	AddRoutesRegistar(func(e *gin.Engine) { e.DELETE(url, NewTRouteHandler(handler)) })
}

// Add RoutesRegistar
func AddRoutesRegistar(reg RoutesRegistar) {
	routesRegiatarList = append(routesRegiatarList, reg)
}

// Register GIN route for consul healthcheck
func registerRouteForConsulHealthcheck(router *gin.Engine) {
	router.GET(common.GetPropStr(common.PROP_CONSUL_HEALTHCHECK_URL), consul.DefaultHealthCheck)
}

func startHttpServer(ctx context.Context, server *http.Server) {
	logrus.Infof("Listening and serving HTTP on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.Fatalf("HttpServer ListenAndServe: %s", err)
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

/*
	Default way to Bootstrap server, basically the same as follows:

		common.DefaultReadConfig(args)
		// ... plus some configuration for logging and so on
		BootstrapServer()
*/
func DefaultBootstrapServer(args []string) {
	// default way to load configuration
	common.DefaultReadConfig(args)

	// determine the writer that we will use for logging and so on
	if common.IsProdMode() && common.ContainsProp(common.PROP_LOGGING_ROLLING_FILE) {
		loggerOut = common.BuildRollingLogFileWriter(common.GetPropStr(common.PROP_LOGGING_ROLLING_FILE))
		loggerErrOut = loggerOut
	}
	logrus.SetOutput(loggerOut)

	// bootstraping 
	BootstrapServer()
}

/*
	Bootstrap server 

	This func will attempt to create http server, connect to MySQL, Redis or Consul based on the configuration loaded.

	It also handles service registration/de-registration on Consul before Gin bootstraped and after
	SIGTERM/INTERRUPT signals are received.

	Graceful shutdown for the http server is also enabled and can be configured through props.

	To configure server, MySQL, Redis, Consul and so on, see PROPS_* in prop.go.
*/
func BootstrapServer() {
	start := time.Now().UnixMilli()
	defer triggerShutdownHook()
	AddShutdownHook(func() { MarkServerShuttingDown() })

	ctx, cancel := context.WithCancel(context.Background())
	AddShutdownHook(func() { cancel() })

	logrus.Info("\n\n############# Bootstrapping Server #############\n")

	// mysql
	if mysql.IsMySqlEnabled() {
		if e := mysql.InitMySqlFromProp(); e != nil {
			logrus.Fatalf("Failed to establish connection to MySQL, %v", e)
		}
	}

	// redis
	if redis.IsRedisEnabled() {
		if _, e := redis.InitRedisFromProp(); e != nil {
			logrus.Fatalf("Failed to establish connection to Redis, %v", e)
		}
	}

	// rabbitmq
	if rabbitmq.IsEnabled() {
		rabbitmq.StartRabbitMqClientAsync(ctx)
	}

	// web server
	if common.GetPropBool(common.PROP_SERVER_WEB_ENABLED) {
		logrus.Info("Starting http server")

		// Load propagation keys for tracing
		common.LoadPropagationKeyProp()

		if common.IsProdMode() {
			gin.SetMode(gin.ReleaseMode)
		}

		// gin engine
		engine := gin.New()
		engine.Use(AuthMiddleware())
		engine.Use(TraceMiddleware())

		if !common.IsProdMode() {
			engine.Use(gin.Logger()) // default logger for debugging
		}

		// register customer recovery func
		engine.Use(gin.RecoveryWithWriter(loggerErrOut, DefaultRecovery))

		// register consul health check
		if consul.IsConsulEnabled() {
			registerRouteForConsulHealthcheck(engine)
		}

		// register custom routes
		for _, registerRoute := range routesRegiatarList {
			registerRoute(engine)
		}

		// start the http server
		server := createHttpServer(engine)
		go startHttpServer(ctx, server)
		AddShutdownHook(func() { shutdownHttpServer(server) })
	}

	// consul
	if consul.IsConsulEnabled() {
		logrus.Info("Creating Consul client")

		// create consul client
		if _, e := consul.GetConsulClient(); e != nil {
			log.Fatalf("Failed to establish connection to Consul, %v", e)
		}

		// deregister on shutdown
		AddShutdownHook(func() {
			if e := consul.DeregisterService(); e != nil {
				logrus.Errorf("Failed to deregister on Consul, %v", e)
			}
		})

		if e := consul.RegisterService(); e != nil {
			log.Fatalf("Failed to register on Consul, %v", e)
		}
	}

	// scheduler
	if common.HasScheduler() {
		logrus.Info("Starting scheduler asynchronously")
		common.GetScheduler().StartAsync()
		AddShutdownHook(func() { common.GetScheduler().Stop() })
	}

	end := time.Now().UnixMilli()
	logrus.Infof("\n\n############# Server Bootstraped (took: %dms) #############\n", end-start)

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
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
		timeout = 5
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// shutdown web server with the timeout
	server.Shutdown(ctx)
	logrus.Infof("Http server exited")
}

// Resolve handler path for open api
func OpenApiPath(relPath string) string {
	return ResolvePath(relPath, true)
}

// Resolve handler path for internal endpoints
func InternalApiPath(relPath string) string {
	return ResolvePath(relPath, false)
}

// Resolve handler path
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
func AddRouteAuthWhitelist(pred common.Predicate[string]) {
	rawlmu.Lock()
	defer rawlmu.Unlock()
	routesAuthWhitelist = append(routesAuthWhitelist, pred)
}

// Check whether url is in route whitelist
func IsRouteWhitelist(url string) bool {
	rawlmu.RLock()
	defer rawlmu.RUnlock()

	for _, predicate := range routesAuthWhitelist {
		if isok := predicate(url); isok {
			return true
		}
	}
	return false
}

// Tracing Middleware
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// propagate tracing key/value pairs with context
		ctx := c.Request.Context()
		propagatedKeys := append(common.GetPropagationKeys(), common.X_SPANID, common.X_TRACEID)

		for _, k := range propagatedKeys {
			if h := c.GetHeader(k); h != "" {
				ctx = context.WithValue(ctx, k, h)
			}
		}

		// replace the context
		c.Request = c.Request.WithContext(ctx)

		// follow the chain
		c.Next()
	}
}

// Authentication Middleware, only validates request url that starts with "/open/api"
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Request.RequestURI

		if strings.HasPrefix(strings.ToLower(url), OPEN_API_PREFIX) {

			if !IsRouteWhitelist(url) && !IsRequestAuthenticated(c) {
				logrus.Infof("Unauthenticated request rejected: %v '%s'", c.Request.Method, url)
				DispatchErrMsgJson(c, "Please sign in first")
				c.Abort()
				return // request rejected
			}
		}

		// follow the chain
		c.Next()
	}
}
