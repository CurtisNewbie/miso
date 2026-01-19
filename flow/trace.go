package flow

import (
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/spf13/cast"
)

const (
	XTraceId  = "X-B3-TraceId"
	XSpanId   = "X-B3-SpanId"
	XUsername = "x-username"
	XUserNo   = "x-userno"
	XRoleNo   = "x-roleno"
)

var (
	propagationKeys = PropagationKeys{keys: hash.NewSyncSet(XTraceId, XSpanId, XUsername, XUserNo, XRoleNo)}
)

type PropagationKeys struct {
	keys *hash.SyncSet[string]
}

// Add propagation key for tracing
func AddPropagationKeys(keys ...string) {
	propagationKeys.keys.AddAll(keys)
}

// Add propagation key for tracing
func AddPropagationKey(key string) {
	propagationKeys.keys.Add(key)
}

// Get all existing propagation key
func GetPropagationKeys() []string {
	return propagationKeys.keys.CopyKeys()
}

func UsePropagationKeys(forEach func(key string)) {
	for _, k := range GetPropagationKeys() {
		forEach(k)
	}
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
