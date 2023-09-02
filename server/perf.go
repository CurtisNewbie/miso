package server

import (
	"time"

	"github.com/curtisnewbie/miso/core"
	"github.com/gin-gonic/gin"
)

var (
	perfLogExcluded core.Set[string] = core.NewSet[string]()
)

// Perf Middleware that calculates how much time each request takes
func PerfMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uri := ctx.Request.RequestURI

		if perfLogExcluded.Has(uri) {
			ctx.Next()
			return
		}

		start := time.Now()
		ctx.Next() // continue the handler chain
		core.TraceLogger(ctx).Infof("%-6v %-60v [%s]", ctx.Request.Method, ctx.Request.RequestURI, time.Since(start))
	}
}

// Ask PerfMiddleware to stop measuring perf of provided path
func PerfLogExclPath(path string) {
	perfLogExcluded.Add(path)
}
