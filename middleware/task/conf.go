package task

import "github.com/curtisnewbie/miso/miso"

const (
	/*
		------------------------------------

		Prop for distributed task scheduling

		------------------------------------
	*/

	PropTaskSchedulingEnabled = "task.scheduling.enabled"
	PropTaskSchedulingGroup   = "task.scheduling.group"
)

func init() {
	miso.SetDefProp(PropTaskSchedulingEnabled, true)
	miso.SetDefProp(PropTaskSchedulingGroup, "${app.name}")
}
