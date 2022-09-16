package server

import (
	"fmt"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/curtisnewbie/gocommon/util"
	"github.com/curtisnewbie/gocommon/weberr"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

/* Bootstrap Server */
func BootstrapServer(serverConf *config.ServerConfig, isProd bool, registerRoutesHandler func(*gin.Engine)) error {

	if isProd {
		log.Info("Using prod profile, will run with ReleaseMode")
		gin.SetMode(gin.ReleaseMode)
	}

	// gin
	engine := gin.Default()

	// register customer recovery func
	engine.Use(gin.CustomRecovery(DefaultRecovery))

	// register routes
	registerRoutesHandler(engine)

	// start the server
	addr := fmt.Sprintf("%v:%v", serverConf.Host, serverConf.Port)
	err := engine.Run(addr)
	if err != nil {
		log.Errorf("Failed to bootstrap gin engine (web server), %v", err)
		return err
	}
	log.Printf("Web server bootstrapped on address: %s", addr)

	return nil
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
