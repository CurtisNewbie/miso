package flow

import (
	"sync"

	"github.com/curtisnewbie/miso/util/hash"
	"github.com/spf13/cast"
)

const (
	XTraceId  = "X-B3-TraceId"
	XSpanId   = "X-B3-SpanId"
	XUsername = "x-username"
)

var (
	propagationKeys = PropagationKeys{keys: hash.NewSet(XTraceId, XSpanId, XUsername)}
)

type PropagationKeys struct {
	keys hash.Set[string]
	rwmu sync.RWMutex
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
