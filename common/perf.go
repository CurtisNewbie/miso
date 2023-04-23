package common

import (
	"log"
	"time"
)

// Run timer for named operation and print result in log
//
// e.g.,
// 	defer TimeOp(ec, time.Now(), "someOperation")
func TimeOp(ec ExecContext, start time.Time, name string) {
	ec.Log.Infof("Operation '%s' took '%s'", name, time.Since(start))
}

// Run timer for named operation and print result in log
//
// e.g.,
// 	defer PTimeOp(time.Now(), "someOperation")
func PTimeOp(start time.Time, name string) {
	log.Printf("Operation '%s' took '%s'", name, time.Since(start))
}
