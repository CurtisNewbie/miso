package sqlite

import (
	"fmt"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/middleware/dbquery"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/strutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func init() {
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

var (
	slowThreshold = 200 * time.Millisecond
	dbLogger      = dbquery.NewGormLogger(logger.Config{SlowThreshold: slowThreshold, LogLevel: logger.Warn})
	module        = miso.InitAppModuleFunc(func() *sqliteModule {
		return &sqliteModule{
			sqliteOnce: &sync.Once{},
		}
	})
)

// Get SQLite client.
func (m *sqliteModule) sqlite() *gorm.DB {
	m.initOnce()
	if miso.IsDebugLevel() || !miso.IsProdMode() || miso.GetPropBool(PropSqliteLogSQL) {
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

		dbquery.ImplGetPrimaryDBFunc(func() *gorm.DB { return GetDB() })
	})
}

// Get SQLite client.
func GetDB() *gorm.DB {
	return module().sqlite()
}

// Create new SQLite connection.
func NewConn(path string, wal bool) (*gorm.DB, error) {
	miso.Infof("Connecting to SQLite database '%s', enable WAL: %v", path, wal)

	cfg := &gorm.Config{
		PrepareStmt: true, CreateBatchSize: 100,
		Logger: dbLogger,
	}
	db, err := gorm.Open(sqlite.Open(path), cfg)
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

	miso.Debug("Setting SQLite Cache")
	if err := db.Exec("PRAGMA cache_size = -20000").Error; err != nil { // -20000 = 20mb
		miso.Warnf("Failed to setup SQLite cache, %v", err)
	}

	miso.Debug("Setting SQLite auto_vacuum to INCREMENTAL mode")
	if err := db.Exec("PRAGMA auto_vacuum = INCREMENTAL").Error; err != nil {
		miso.Warnf("Failed to change SQLite auto_vacuum mode to INCREMENTAL, %v", err)
	}

	miso.Debug("Setting SQLite temp_store to MEMORY")
	if err := db.Exec("PRAGMA temp_store = MEMORY").Error; err != nil {
		miso.Warnf("Failed to change SQLite temp_store to MEMORY, %v", err)
	}

	return db, nil
}

func sqliteBootstrap(rail miso.Rail) error {
	module().initOnce()

	colorful := false
	if miso.GetPropStrTrimmed(miso.PropLoggingRollingFile) == "" {
		colorful = true
	}
	if logSQL() {
		dbLogger.UpdateConfig(logger.Config{SlowThreshold: slowThreshold, LogLevel: logger.Info, Colorful: colorful})
	} else {
		dbLogger.UpdateConfig(logger.Config{SlowThreshold: slowThreshold, LogLevel: logger.Warn, Colorful: colorful})
	}

	return nil
}

func sqliteBootstrapCondition(rail miso.Rail) (bool, error) {
	return !strutil.IsBlankStr(miso.GetPropStr(PropSqliteFile)), nil
}

func logSQL() bool {
	return miso.IsDebugLevel() || !miso.IsProdMode() || miso.GetPropBool(PropSqliteLogSQL)
}
