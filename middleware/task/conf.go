package task

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Distributed Task Scheduling Configuration
const (

	// misoconfig-prop: enable distributed task scheduling | true
	PropTaskSchedulingEnabled = "task.scheduling.enabled"

	// misoconfig-prop: name of the cluster | `${app.name}`
	PropTaskSchedulingGroup = "task.scheduling.group"
)

func init() {
	miso.SetDefProp(PropTaskSchedulingEnabled, true)
	miso.SetDefProp(PropTaskSchedulingGroup, "${app.name}")
}
