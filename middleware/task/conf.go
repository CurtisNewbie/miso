package task

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: Distributed Task Scheduling Configuration
const (

	// misoconfig-prop: enable distributed task scheduling | true
	PropTaskSchedulingEnabled = "task.scheduling.enabled"

	// misoconfig-prop: name of the cluster | `"${app.name}"`
	PropTaskSchedulingGroup = "task.scheduling.group"

	// misoconfig-prop: disable specific task by it's name | false
	// misoconfig-doc-only
	PropTaskSchedulingTaskDisabled = "task.scheduling.${taskName}.disabled"

	// misoconfig-prop: enable api to manually trigger tasks on one of the worker nodes | false
	PropTaskSchedulingApiTriggerJobEnabled = "task.scheduling.api.trigger-task.enabled"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropTaskSchedulingEnabled, true)
	miso.SetDefProp(PropTaskSchedulingGroup, "${app.name}")
	miso.SetDefProp(PropTaskSchedulingApiTriggerJobEnabled, false)
}

// misoconfig-default-end
