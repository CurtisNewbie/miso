package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/consul"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/redis"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Routes registar
type RoutesRegistar func(*gin.Engine)

var (
	routesRegiatarList []RoutesRegistar = []RoutesRegistar{}

	shuttingDown   bool = false
	shutingDownRwm sync.RWMutex
)

func init() {
	common.SetDefProp(common.PROP_SERVER_HOST, "localhost")
	common.SetDefProp(common.PROP_SERVER_PORT, 8080)
	common.SetDefProp(common.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC, 5)
}

/*
	Add RoutesRegistar

	This func should be called before BootstrapServer
*/
func AddRoutesRegistar(reg RoutesRegistar) {
	routesRegiatarList = append(routesRegiatarList, reg)
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

	if common.IsProdMode() {
		logrus.Info("Bootstraping Gin with ReleaseMode")
		gin.SetMode(gin.ReleaseMode)
	}

	// mysql
	if mysql.IsMySqlEnabled() {
		if err := mysql.InitMySqlFromProp(); err != nil {
			panic(err)
		}
	}

	// redis
	if redis.IsRedisEnabled() {
		redis.InitRedisFromProp()
	}

	// gin router
	router := gin.Default()

	// register customer recovery func
	router.Use(gin.CustomRecovery(DefaultRecovery))

	// register consul health check
	if consul.IsConsulEnabled() {
		router.GET(common.GetPropStr(common.PROP_CONSUL_HEALTHCHECK_URL), consul.DefaultHealthCheck)
	}

	// register custom routes
	for _, registar := range routesRegiatarList {
		registar(router)
	}

	addr := fmt.Sprintf("%s:%s", common.GetPropStr(common.PROP_SERVER_HOST), common.GetPropStr(common.PROP_SERVER_PORT))
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// start the web server
	go func() {
		logrus.Infof("Listening and serving HTTP on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("HttpServer ListenAndServe: %s", err)
		}
	}()

	// register on consul
	if consul.IsConsulEnabled() {
		logrus.Infof("Consul enabled, will register as a service")

		// register on consul, retry until we success
		go func() {
			if _, err := consul.GetConsulClient(); err != nil {
				logrus.Errorf("Failed to init Concul client, %v", err)
				shutdownServer(server)
				panic(err)
			}

			retry := 0
			for {
				if IsShuttingDown() {
					break
				}

				if regerr := consul.RegisterService(); regerr == nil {
					break
				}

				logrus.Errorf("Failed to register on consul, has retried %d times.", retry)
				retry++
				time.Sleep(1 * time.Second)
			}
		}()
	} else {
		logrus.Infof("Consul config disabled, will not register on Consul")
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// start to shutdown gracefully
	logrus.Info("Shutting down server gracefully")
	shutdownServer(server)
}

/*
	shutdown server, register on Consul if necessary

	This func looks for following prop:

	PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC
*/
func shutdownServer(server *http.Server) {
	MarkServerShuttingDown()

	// deregister on consul if necessary
	if e := consul.DeregisterService(); e != nil {
		logrus.Errorf("Failed to de-register on consul, err: %v", e)
	}

	consul.UnsubscribeServerList()

	// set timeout for graceful shutdown
	timeout := common.GetPropInt(common.PROP_SERVER_GRACEFUL_SHUTDOWN_TIME_SEC)
	if timeout <= 0 {
		timeout = 5
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// shutdown web server with the timeout
	server.Shutdown(ctx)
	logrus.Infof("Server exited")
}

// Resolve handler path
func ResolvePath(relPath string, isOpenApi bool) string {
	if isOpenApi {
		return "/open/api" + relPath
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
