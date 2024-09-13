package miso

import (
	"github.com/curtisnewbie/miso/util"
)

// Deprecated, use util.* instead.
const (
	LOOPBACK_LOCALHOST = util.LoopbackLocalHost
	LOOPBACK_127       = util.Loopback127
	LOCAL_IP_ANY       = util.LocalIpAny
)

// Deprecated, use util.* instead.
var (
	GetLocalIPV4   = util.GetLocalIPV4
	IsLocalAddress = util.IsLocalAddress
)
