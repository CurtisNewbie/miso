package server

import (
	"context"
	"fmt"
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
	routesRegiatarList []RoutesRegistar = []RoutesRegistar{}

	shuttingDown   bool = false
	shutingDownRwm sync.RWMutex

	shutdownHook []func()
	shmu         sync.Mutex

	routesAuthWhitelist []common.Predicate[string] = []common.Predicate[string]{}
	rawlmu              sync.RWMutex
)

func init() {
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

/*
	Add RoutesRegistar

	This func should be called before BootstrapServer
*/
func AddRoutesRegistar(reg RoutesRegistar) {
	routesRegiatarList = append(routesRegiatarList, reg)
}

// Register GIN route for consul healthcheck
func registerRouteForConsulHealthcheck(router *gin.Engine) {
	if consul.IsConsulEnabled() {
		router.GET(common.GetPropStr(common.PROP_CONSUL_HEALTHCHECK_URL), consul.DefaultHealthCheck)
	}
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
	Bootstrap server

	This func will attempt to connect MySQL, Redis, Consul based on the props loaded

	It also handles service registration/de-registration on Consul before Gin bootstraped and after
	SIGTERM/INTERRUPT signals are received.

	Graceful shutdown for the web server is also enabled and can be configured through props.

	To configure server, MySQL, Redis, Consul and so on, see PROPS_* in prop.go
*/
func BootstrapServer() {
	start := time.Now().UnixMilli()
	defer triggerShutdownHook()

	ctx, cancel := context.WithCancel(context.Background())
	AddShutdownHook(func() { cancel() })

	if common.IsProdMode() {
		logrus.Info("Bootstraping Gin with ReleaseMode")
		gin.SetMode(gin.ReleaseMode)
	}

	// mysql
	if mysql.IsMySqlEnabled() {
		mysql.MustInitMySqlFromProp()
	}

	// redis
	if redis.IsRedisEnabled() {
		redis.InitRedisFromProp()
	}

	// gin router
	router := gin.New()
	router.Use(AuthMiddleware())

	if !common.IsProdMode() {
		router.Use(gin.Logger())
	}

	// register customer recovery func
	router.Use(gin.CustomRecovery(DefaultRecovery))

	// register consul health check
	registerRouteForConsulHealthcheck(router)

	// register custom routes
	for _, registar := range routesRegiatarList {
		registar(router)
	}

	// start the http server
	server := createHttpServer(router)
	go startHttpServer(ctx, server)
	AddShutdownHook(func() { shutdownServer(server) })

	// register on consul
	if consul.IsConsulEnabled() {
		// create consul client
		consul.MustInitConsulClient()
		AddShutdownHook(func() { consul.UnsubscribeServerList() })

		// register on consul, retry until we success, the Consul server may not be ready or may be down temporarily
		go func() {
			retry := 0
			for {
				select {
				case <-ctx.Done():
					logrus.Info("Aborting consul registration", ctx.Err())
					return
				default:
					if regerr := consul.RegisterService(); regerr == nil {
						return // success
					}

					logrus.Errorf("Failed to register on consul, has retried %d times.", retry)
					retry++
					time.Sleep(1 * time.Second)
				}
			}
		}()
	}

	if rabbitmq.IsEnabled() {
		rabbitmq.StartRabbitMqClient(ctx)
	}
	end := time.Now().UnixMilli()
	logrus.Infof("Server bootstraped, took: %dms", end-start)

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}

/*
	shutdown server, register on Consul if necessary

	This func looks for following prop:

	PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC
*/
func shutdownServer(server *http.Server) {
	logrus.Info("Shutting down server gracefully")
	MarkServerShuttingDown()

	// deregister on consul if necessary
	if e := consul.DeregisterService(); e != nil {
		logrus.Errorf("Failed to deregister on consul, err: %v", e)
	}

	// set timeout for graceful shutdown
	timeout := common.GetPropInt(common.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC)
	if timeout <= 0 {
		timeout = 5
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// shutdown web server with the timeout
	server.Shutdown(ctx)
	logrus.Infof("HttpServer exited")
}

// Resolve handler path
func ResolvePath(relPath string, isOpenApi bool) string {
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

func AddRouteAuthWhitelist(pred common.Predicate[string]) {
	rawlmu.Lock()
	defer rawlmu.Unlock()

	routesAuthWhitelist = append(routesAuthWhitelist, pred)
}

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

// Authentication Middleware, only works for request url that starts with "/open/api"
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Request.RequestURI

		if strings.HasPrefix(strings.ToLower(url), OPEN_API_PREFIX) {

			if !IsRouteWhitelist(url) && !IsRequestAuthenticated(c) {
				logrus.Infof("Unauthenticated request rejected, url: '%s'", url)
				DispatchErrMsgJson(c, "Please sign up first")
				c.Abort()
				return
			}
		}

		ctx := c.Request.Context()
		if spanId := c.GetHeader(common.X_B3_SPANID); spanId != "" {
			ctx = context.WithValue(ctx, common.X_B3_SPANID, spanId)
		}

		if traceId := c.GetHeader(common.X_B3_TRACEID); traceId != "" {
			ctx = context.WithValue(ctx, common.X_B3_TRACEID, traceId)
		}

		r2 := c.Request.WithContext(ctx)
		c.Request = r2

		// follow the chain
		c.Next()
	}
}
