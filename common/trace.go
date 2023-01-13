package common

import (
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	X_TRACEID  = "x-b3-traceid"
	X_SPANID   = "x-b3-spanid"
	X_USERNAME = "username"
)

var (
	propagationKeys = PropagationKeys{keys: NewSet[string]()}
)

type PropagationKeys struct {
	keys Set[string]
	rwmu sync.RWMutex
}

// Read property and find propagation keys .
//
// This func looks for following property.
//
// 	"tracing.propagation.keys"
func LoadPropagationKeyProp() {
	propagationKeys.rwmu.Lock()
	defer propagationKeys.rwmu.Unlock()

	keys := GetPropStringSlice(PROP_TRACING_PROPAGATION_KEYS)
	for _, k := range keys {
		propagationKeys.keys.Add(k)
	}

	logrus.Infof("Loaded propagation keys for tracing: %v", keys)
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
		keys = append(keys, k)
	}
	return keys
}
