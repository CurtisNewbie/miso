//go:build !excl_sqlite
// +build !excl_sqlite

package miso

import (
	"fmt"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sqliteDb   *gorm.DB = nil
	sqliteOnce sync.Once
)

func init() {
	SetDefProp(PropSqliteWalEnabled, true)
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap SQLite",
		Bootstrap: SqliteBootstrap,
		Condition: SqliteBootstrapCondition,
		Order:     BootstrapOrderL1,
	})
}

/*
Get sqlite client.

Client is initialized if necessary.

This func looks for prop:

	PROP_SQLITE_FILE
*/
func GetSqlite() *gorm.DB {
	sqliteOnce.Do(initSqlite)
	if IsDebugLevel() {
		return sqliteDb.Debug()
	}
	return sqliteDb
}

func initSqlite() {
	sq, err := newSqlite(GetPropStr(PropSqliteFile))
	if err != nil {
		panic(err)
	}
	sqliteDb = sq

	// https://www.sqlite.org/pragma.html#pragma_journal_mode
	if GetPropBool(PropSqliteWalEnabled) {
		Debug("Enabling SQLite WAL mode")
		var mode string
		t := sq.Raw("PRAGMA journal_mode=WAL").Scan(&mode)
		if err := t.Error; err != nil {
			panic(fmt.Errorf("failed to enable WAL mode, %v", err))
		} else {
			Debugf("Enabled SQLite WAL mode, result: %v", mode)
		}
	}
}

func newSqlite(path string) (*gorm.DB, error) {
	Infof("Connecting to SQLite database '%s'", path)

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite, %v", err)
	}

	tx, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to connect SQLite, %v", err)
	}

	// make sure the handle is actually connected
	err = tx.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping SQLite, %v", err)
	}

	Infof("SQLite connected")
	return db, nil
}

func SqliteBootstrap(rail Rail) error {
	GetSqlite()
	return nil
}

func SqliteBootstrapCondition(rail Rail) (bool, error) {
	return !IsBlankStr(GetPropStr(PropSqliteFile)), nil
}
