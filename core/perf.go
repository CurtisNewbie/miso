package core

import (
	"log"
	"time"

	"github.com/sirupsen/logrus"
)

// Run timer for named operation and print result using logrus
//
// e.g.,
//
//	defer LTimeOp(ec, time.Now(), "someOperation")
func LTimeOp(start time.Time, name string) {
	logrus.Infof("%s took %s", name, time.Since(start))
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
