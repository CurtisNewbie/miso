package miso

import (
	"fmt"
	"time"
)

// Run timer for named operation and print result in log
//
// e.g.,
//
//	defer TimeOp(rail, time.Now(), "someOperation")
func TimeOp(r Rail, start time.Time, name string, args ...any) {
	if len(args) > 0 {
		name = fmt.Sprintf(name, args...)
	}
	r.Infof("%s took %s", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
//
//	defer DebugTimeOp(ec, time.Now(), "someOperation")
func DebugTimeOp(r Rail, start time.Time, name string, args ...any) {
	if len(args) > 0 {
		name = fmt.Sprintf(name, args...)
	}
	r.Debugf("%s took %s", name, time.Since(start))
}
