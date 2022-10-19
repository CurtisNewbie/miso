package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/curtisnewbie/gocommon/consul"
	"github.com/curtisnewbie/gocommon/util"
	"github.com/curtisnewbie/gocommon/weberr"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// Routes registar
type RoutesRegistar func(*gin.Engine)

// Bootstrap Server With Gin
func BootstrapServer(conf *config.Configuration, routesRegistar RoutesRegistar) {

	if config.IsProdMode() {
		logrus.Info("Using prod profile, will run gin with ReleaseMode")
		gin.SetMode(gin.ReleaseMode)
	}

	// gin engine
	engine := gin.Default()

	// register customer recovery func
	engine.Use(gin.CustomRecovery(DefaultRecovery))

	// register routes
	routesRegistar(engine)

	// start the server
	go func() {
		addr := fmt.Sprintf("%v:%v", conf.ServerConf.Host, conf.ServerConf.Port)
		err := engine.Run(addr)
		if err != nil {
			logrus.Errorf("Failed to bootstrap gin engine, %v", err)
			return
		}
		logrus.Printf("Server bootstrapped on address: %s", addr)
	}()

	// register on consul
	if conf.ConsulConf != nil && conf.ConsulConf.Enabled {
		if _, err := consul.InitConsulClient(conf.ConsulConf); err != nil {
			panic(err)
		}

		if err := consul.RegisterService(conf.ConsulConf, &conf.ServerConf); err != nil {
			panic(err)
		}
	}

	// wait for Interrupt or SIGTERM, and shutdown gracefully
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server gracefully")

	// deregister on consul if necessary
	if config.GlobalConfig.ConsulConf != nil && conf.ConsulConf.Enabled {
		consul.DeregisterService(config.GlobalConfig.ConsulConf)
	}

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second) // at most 5 seconds
	defer cancel()

	logrus.Println("Server exited")
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
