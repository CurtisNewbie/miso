package zk

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Zookeeper Configuration
const (

	// misoconfig-prop: enable zk client | false
	PropZkEnabled = "zk.enabled"

	// misoconfig-prop: zk server host (slice of string) | localhost
	PropZkHost = "zk.hosts"

	// misoconfig-prop: zk server session timeout (seconds) | 5
	PropZkSessionTimeout = "zk.session-timeout"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropZkEnabled, false)
	miso.SetDefProp(PropZkHost, "localhost")
	miso.SetDefProp(PropZkSessionTimeout, 5)
}

// misoconfig-default-end
