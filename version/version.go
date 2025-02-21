package version

import (
	"runtime/debug"
)

var (
	Version = "v0.1.15-beta.2"
)

func init() {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range buildInfo.Deps {
			if dep.Path == "github.com/curtisnewbie/miso" {
				if dep.Version != "" {
					Version = dep.Version
				}
				break
			}
		}
	}
}
