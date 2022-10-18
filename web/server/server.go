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
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

type RegisterRoutesHandler func(*gin.Engine)

type PostShutdownHandler func()

/* Bootstrap Server */
func BootstrapServer(serverConf *config.ServerConfig, registerRoutesHandler RegisterRoutesHandler) {

	if config.IsProdMode() {
		log.Info("Using prod profile, will run gin with ReleaseMode")
		gin.SetMode(gin.ReleaseMode)
	}

	// gin engine
	engine := gin.Default()

	// register customer recovery func
	engine.Use(gin.CustomRecovery(DefaultRecovery))

	// register routes
	registerRoutesHandler(engine)

	// start the server
	addr := fmt.Sprintf("%v:%v", serverConf.Host, serverConf.Port)

	go func() {
		err := engine.Run(addr)
		if err != nil {
			log.Errorf("Failed to bootstrap gin engine (web server), %v", err)
			return
		}
		log.Printf("Web server bootstrapped on address: %s", addr)
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logrus.Println("Shutting down server ...")

	// deregister consul if necessary
	consul.DeregisterService(&config.GlobalConfig.ConsulConf)

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
