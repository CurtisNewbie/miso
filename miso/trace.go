package miso

import (
	"sync"

	"github.com/curtisnewbie/miso/util/hash"
	"github.com/spf13/cast"
)

const (
	XTraceId = "X-B3-TraceId"
	XSpanId  = "X-B3-SpanId"
)

var (
	propagationKeys = PropagationKeys{keys: hash.NewSet[string]()}
)

type PropagationKeys struct {
	keys hash.Set[string]
	rwmu sync.RWMutex
}

func init() {
	propagationKeys.keys.Add(XTraceId)
	propagationKeys.keys.Add(XSpanId)
}

// Read property and find propagation keys .
//
// This func looks for following property.
//
//	"tracing.propagation.keys"
func LoadPropagationKeys(r Rail) {
	propagationKeys.rwmu.Lock()
	defer propagationKeys.rwmu.Unlock()

	keys := GetPropStrSlice(PropTracingPropagationKeys)
	for _, k := range keys {
		propagationKeys.keys.Add(k)
	}

	r.Infof("Loaded propagation keys for tracing: %v", propagationKeys.keys.String())
}

// Add propagation key for tracing
func AddPropagationKeys(keys ...string) {
	propagationKeys.rwmu.Lock()
	defer propagationKeys.rwmu.Unlock()
	propagationKeys.keys.AddAll(keys)
}

// Add propagation key for tracing
func AddPropagationKey(key string) {
	propagationKeys.rwmu.Lock()
	defer propagationKeys.rwmu.Unlock()

	propagationKeys.keys.Add(key)
}

// Get all existing propagation key
func GetPropagationKeys() []string {
	propagationKeys.rwmu.RLock()
	defer propagationKeys.rwmu.RUnlock()
	return propagationKeys.keys.CopyKeys()
}

func UsePropagationKeys(forEach func(key string)) {
	propagationKeys.rwmu.RLock()
	defer propagationKeys.rwmu.RUnlock()

	propagationKeys.keys.ForEach(func(v string) (stop bool) {
		forEach(v)
		return false
	})
}

func LoadPropagationKeysFromHeaders[T any](rail Rail, headers map[string]T) Rail {
	UsePropagationKeys(func(k string) {
		if hv, ok := headers[k]; ok {
			rail = rail.WithCtxVal(k, cast.ToString(hv))
		}
	})
	return rail
}

func BuildTraceHeadersAny(rail Rail) map[string]any {
	headers := map[string]any{}
	UsePropagationKeys(func(key string) {
		headers[key] = rail.CtxValue(key)
	})
	return headers
}

func BuildTraceHeadersStr(rail Rail) map[string]string {
	headers := map[string]string{}
	UsePropagationKeys(func(key string) {
		headers[key] = cast.ToString(rail.CtxValue(key))
	})
	return headers
}
