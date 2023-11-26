package miso

import (
	"log"
	"time"
)

// Run timer for named operation and print result
//
// e.g.,
//
//	defer LTimeOp(ec, time.Now(), "someOperation")
func LTimeOp(start time.Time, name string) {
	Infof("%s took %s", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
//
//	defer TimeOp(ec, time.Now(), "someOperation")
func TimeOp(r Rail, start time.Time, name string) {
	r.Infof("%s took %s", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
//
//	defer DebugTimeOp(ec, time.Now(), "someOperation")
func DebugTimeOp(r Rail, start time.Time, name string) {
	r.Debugf("%s took %s", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
//
//	defer PTimeOp(time.Now(), "someOperation")
func PTimeOp(start time.Time, name string) {
	log.Printf("%s took %s", name, time.Since(start))
}
