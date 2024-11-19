package sqlite

import (
	"fmt"
	"sync"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	miso.SetDefProp(PropSqliteWalEnabled, true)
	miso.RegisterBootstrapCallback(miso.ComponentBootstrap{
		Name:      "Bootstrap SQLite",
		Bootstrap: sqliteBootstrap,
		Condition: sqliteBootstrapCondition,
		Order:     miso.BootstrapOrderL1,
	})
}

type sqliteModule struct {
	sqliteDb   *gorm.DB
	sqliteOnce *sync.Once
}

var module = miso.InitAppModuleFunc(func() *sqliteModule {
	return &sqliteModule{
		sqliteOnce: &sync.Once{},
	}
})

// Get SQLite client.
func (m *sqliteModule) sqlite() *gorm.DB {
	m.initOnce()
	if miso.IsDebugLevel() || !miso.IsProdMode() {
		return m.sqliteDb.Debug()
	}
	return m.sqliteDb
}

func (m *sqliteModule) initOnce() {
	m.sqliteOnce.Do(func() {
		sq, err := NewConn(miso.GetPropStr(PropSqliteFile), miso.GetPropBool(PropSqliteWalEnabled))
		if err != nil {
			panic(err)
		}
		m.sqliteDb = sq
	})
}

// Get SQLite client.
func GetDB() *gorm.DB {
	return module().sqlite()
}

// Create new SQLite connection.
func NewConn(path string, wal bool) (*gorm.DB, error) {
	miso.Infof("Connecting to SQLite database '%s', enable WAL: %v", path, wal)

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite, %w", err)
	}

	tx, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to connect SQLite, %w", err)
	}

	// make sure the handle is actually connected
	err = tx.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping SQLite, %w", err)
	}
	miso.Infof("SQLite connected: '%s'", path)

	// https://www.sqlite.org/pragma.html#pragma_journal_mode
	if wal {
		miso.Debug("Enabling SQLite WAL mode")
		var mode string
		t := db.Raw("PRAGMA journal_mode=WAL").Scan(&mode)
		if err := t.Error; err != nil {
			return db, fmt.Errorf("failed to enable WAL mode, %w", err)
		} else {
			miso.Debugf("Enabled SQLite WAL mode, result: %v", mode)
		}
	}

	return db, nil
}

func sqliteBootstrap(rail miso.Rail) error {
	module().initOnce()
	return nil
}

func sqliteBootstrapCondition(rail miso.Rail) (bool, error) {
	return !util.IsBlankStr(miso.GetPropStr(PropSqliteFile)), nil
}
