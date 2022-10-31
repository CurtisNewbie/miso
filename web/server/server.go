package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/curtisnewbie/gocommon/consul"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/redis"
	"github.com/curtisnewbie/gocommon/util"
	"github.com/curtisnewbie/gocommon/weberr"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Routes registar
type RoutesRegistar func(*gin.Engine)

// Bootstrap server
//
// This func will attempt to connect MySQL, Redis, Consul if possible
// depending on whether the associated config is found in *config.Configuration
//
// It also handles service registration/de-registration on Consul before Gin bootstraped and after
// SIGTERM/INTERRUPT signals are received
func BootstrapServer(conf *config.Configuration, routesRegistar RoutesRegistar) {

	if config.IsProdMode() {
		logrus.Info("Bootstraping Gin with ReleaseMode")
		gin.SetMode(gin.ReleaseMode)
	}

	// mysql
	if conf.DBConf != nil && conf.DBConf.Enabled {
		if err := mysql.InitDBFromConfig(conf.DBConf); err != nil {
			panic(err)
		}
	} else {
		logrus.Infof("MySQL config disabled, will not connect to MySQL")
	}

	// redis
	if conf.RedisConf != nil && conf.RedisConf.Enabled {
		redis.InitRedisFromConfig(conf.RedisConf)
	} else {
		logrus.Infof("Redis config disabled, will not connect to Redis")
	}

	// gin router
	router := gin.Default()

	// register customer recovery func
	router.Use(gin.CustomRecovery(DefaultRecovery))

	// whether consul is enabled
	isConsulEnabled := conf.ConsulConf != nil && conf.ConsulConf.Enabled

	// register consul health check
	if isConsulEnabled {
		router.GET(conf.ConsulConf.HealthCheckUrl, consul.DefaultHealthCheck)
	}

	// register custom routes
	routesRegistar(router)

	addr := fmt.Sprintf("%v:%v", conf.ServerConf.Host, conf.ServerConf.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// start the web server
	go func() {
		logrus.Infof("Listening and serving HTTP on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("ListenAndServe: %s", err)
		}
	}()

	// register on consul
	if isConsulEnabled {
		if _, err := consul.InitConsulClient(conf.ConsulConf); err != nil {
			logrus.Errorf("Failed to init Concul client, %v", err)
			shutdownServer(conf, server)
			panic(err)
		}

		// at most retry 5 times
		retry := 5
		for {
			if regerr := consul.RegisterService(conf.ConsulConf, &conf.ServerConf); regerr == nil {
				break
			}

			retry--
			if retry == 0 {
				shutdownServer(conf, server)
				panic("failed to register on consul, has retried 5 times.")
			}
			time.Sleep(1 * time.Second)
		}
	} else {
		logrus.Infof("Consul config disabled, will not register on Consul")
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// start to shutdown gracefully
	logrus.Info("Shutting down server gracefully")
	shutdownServer(conf, server)
}

// shutdown server, register on Consul if necessary
func shutdownServer(conf *config.Configuration, server *http.Server) {
	// deregister on consul if necessary
	if e := consul.DeregisterService(); e != nil {
		logrus.Errorf("Failed to de-register on consul, err: %v", e)
	}

	// by default wait for at most 5 seconds
	timeout := conf.ServerConf.GracefulShutdownTimeSec
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
		util.DispatchErrJson(c, err)
		return
	}
	if msg, ok := e.(string); ok {
		util.DispatchErrMsgJson(c, msg)
		return
	}

	util.DispatchErrJson(c, weberr.NewWebErr("Unknown error, please try again later"))
}
