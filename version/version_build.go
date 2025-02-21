package version

import "runtime/debug"

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
