package main

import (
	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	c := common.EmptyExecContext()

	// load configuration from 'app-conf-dev.yml'
	common.LoadConfigFromFile("app-conf-dev.yml", c)

	// add GET request handler
	server.RawGet("/some/path", func(c *gin.Context, ec common.ExecContext) {
		logrus.Info("Received request")
	})

	// bootstrap server
	server.BootstrapServer(c)
}
