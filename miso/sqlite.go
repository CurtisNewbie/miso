package miso

import (
	"sync"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sqlitep = &sqliteHolder{sq: nil}
)

type sqliteHolder struct {
	sq *gorm.DB
	mu sync.RWMutex
}

/*
Get sqlite client

# Client is initialized if necessary

This func looks for prop:

	PROP_SQLITE_FILE
*/
func GetSqlite() *gorm.DB {
	if IsSqliteInitialized() {
		if IsProdMode() {
			return sqlitep.sq
		}

		// not prod mode, enable debugging for printing SQLs
		return sqlitep.sq.Debug()
	}

	sqlitep.mu.Lock()
	defer sqlitep.mu.Unlock()

	if sqlitep.sq == nil {
		path := GetPropStr(PROP_SQLITE_FILE)
		logrus.Infof("Connecting to SQLite database '%s'", path)

		db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
		if err != nil {
			panic(err)
		}

		tx, err := db.DB()
		if err != nil {
			panic(tx)
		}

		// make sure the handle is actually connected
		err = tx.Ping()
		if err != nil {
			panic(err)
		}
		logrus.Infof("SQLite conn initialized")
		sqlitep.sq = db
	}

	if IsDebugLevel() {
		return sqlitep.sq.Debug()
	}

	return sqlitep.sq
}

// Check whether sqlite client is initialized
func IsSqliteInitialized() bool {
	sqlitep.mu.RLock()
	defer sqlitep.mu.RUnlock()
	return sqlitep.sq != nil
}
