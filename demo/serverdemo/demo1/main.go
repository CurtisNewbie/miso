package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/gin-gonic/gin"
)

type SomeReqPayload struct {
}

func main() {
	myJob := func() {}
	myHandler := func(c *gin.Context, ec common.Rail, r SomeReqPayload) (any, error) {
		return nil, nil
	}

	// maybe some scheduling (not distributed)
	common.ScheduleCron("0 0/15 * * * *", true, myJob)

	// register routes and handlers
	server.IPost(server.OpenApiPath("/path"), myHandler)

	// bootstrap server
	server.BootstrapServer(os.Args)
}
