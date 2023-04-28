package common

import (
	"log"
	"time"

	"github.com/sirupsen/logrus"
)

// Run timer for named operation and print result using logrus
//
// e.g.,
// 	defer LTimeOp(ec, time.Now(), "someOperation")
func LTimeOp(start time.Time, name string) {
	logrus.Infof("Op '%s' took '%s'", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
// 	defer TimeOp(ec, time.Now(), "someOperation")
func TimeOp(ec ExecContext, start time.Time, name string) {
	ec.Log.Infof("Op '%s' took '%s'", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
// 	defer PTimeOp(time.Now(), "someOperation")
func PTimeOp(start time.Time, name string) {
	log.Printf("Op '%s' took '%s'", name, time.Since(start))
}
