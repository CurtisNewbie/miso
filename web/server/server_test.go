package server

import (
	"testing"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/gin-gonic/gin"
)

func TestBootstrapServer(t *testing.T) {

	args := make([]string, 2)
	args[0] = "profile=dev"
	args[1] = "configFile=../../app-conf-dev.json"
	_, conf := config.DefaultParseProfConf(args)

	BootstrapServer(conf, func(router *gin.Engine) {
	})
}
