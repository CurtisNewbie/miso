package miso

import (
	"sync"
)

const (
	XTraceId = "X-B3-TraceId"
	XSpanId  = "X-B3-SpanId"
)

var (
	propagationKeys = PropagationKeys{keys: NewSet[string]()}
	TracePrefix     = "X-"
)

type PropagationKeys struct {
	keys Set[string]
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
func AddPropagationKey(key string) {
	propagationKeys.rwmu.Lock()
	defer propagationKeys.rwmu.Unlock()

	propagationKeys.keys.Add(key)
}

// Get all existing propagation key
func GetPropagationKeys() []string {
	propagationKeys.rwmu.RLock()
	defer propagationKeys.rwmu.RUnlock()

	keys := []string{}
	for k := range propagationKeys.keys.Keys {
		keys = append(keys, TracePrefix+k)
	}
	return keys
}
