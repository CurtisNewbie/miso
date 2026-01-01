package dbquery

import (
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"gorm.io/gorm"
)

func InitSchema(rail miso.Rail, initSchemaSegments []string, getDB func() *gorm.DB) error {
	rail.Info("Initializing schema")
	start := time.Now()
	db := getDB()
	for _, seg := range initSchemaSegments {
		if err := db.Exec(seg).Error; err != nil {
			return miso.UnknownErrf(err, "Failed to executed '%v'", seg)
		}
		rail.Debugf("Executed: '%v'", seg)
	}
	rail.Infof("Schema initialized, took: %v", time.Since(start))
	return nil
}

type ConditionalSchemaSegment struct {
	Script    string
	Condition func(*gorm.DB) (ok bool, err error) // considered true if Condition is nil
	AfterExec func(*gorm.DB) error                // can be nil, called when script is executed without error
}

func InitSchemaConditionally(rail miso.Rail, conditionalSegments []ConditionalSchemaSegment, getDB func() *gorm.DB) error {
	rail.Info("Initializing schema")
	start := time.Now()
	db := getDB()
	for _, seg := range conditionalSegments {
		var ok bool = true
		var err error = nil
		if seg.Condition != nil {
			ok, err = seg.Condition(db)
			if err != nil {
				return errs.Wrapf(err, "failed to execute Condition func")
			}
		}
		if ok {
			if err := db.Exec(seg.Script).Error; err != nil {
				return errs.Wrapf(err, "failed to executed '%v'", seg.Script)
			}
			rail.Debugf("Executed: '%v'", seg.Script)

			if seg.AfterExec != nil {
				if err := seg.AfterExec(db); err != nil {
					return errs.Wrapf(err, "failed to call AfterExec for script '%v'", seg.Script)
				}
			}
		}
	}
	rail.Infof("Schema initialized, took: %v", time.Since(start))
	return nil
}
