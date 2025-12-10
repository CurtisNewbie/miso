package miso

import (
	"github.com/curtisnewbie/miso/flow"
	"github.com/spf13/cast"
)

func init() {
	PreServerBootstrap(func(rail Rail) error {
		LoadPropagationKeys(rail)
		return nil
	})
}

// Load propagation keys from configuration.
func LoadPropagationKeys(r Rail) {
	keys := GetPropStrSlice(PropTracingPropagationKeys)
	flow.AddPropagationKeys(keys...)
	r.Infof("Loaded propagation keys for tracing: %v", keys)
}

func LoadPropagationKeysFromHeaders[T any](rail Rail, headers map[string]T) Rail {
	UsePropagationKeys(func(k string) {
		if hv, ok := headers[k]; ok {
			rail = rail.WithCtxVal(k, cast.ToString(hv))
		}
	})
	return rail
}
