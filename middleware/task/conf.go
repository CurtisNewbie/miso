package task

import "github.com/curtisnewbie/miso/miso"

// misoapi-config-section: Distributed Task Scheduling Configuration
const (

	// misoapi-config: enable distributed task scheduling | true
	PropTaskSchedulingEnabled = "task.scheduling.enabled"

	// misoapi-config: name of the cluster | `${app.name}`
	PropTaskSchedulingGroup = "task.scheduling.group"
)

func init() {
	miso.SetDefProp(PropTaskSchedulingEnabled, true)
	miso.SetDefProp(PropTaskSchedulingGroup, "${app.name}")
}
