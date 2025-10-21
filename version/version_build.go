package version

import (
	"runtime/debug"
)

func init() {
	ver := ReadMisoBuildVersion()
	if ver != "" {
		Version = ver
	}
}

func ReadMisoBuildVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range buildInfo.Deps {
			if dep.Path == "github.com/curtisnewbie/miso" {
				if dep.Version != "" {
					return dep.Version
				}
				break
			}
		}
	}
	return ""
}
