package svc

import (
	"embed"

	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/svc"
	"gorm.io/gorm"
)

type schemaMigrateOptions struct {
	Locker              Locker
	IgnorePreviousError bool
}

// Enable auto schema migration.
//
// Put all SQL scripts in a specific directory. Embed the directory as follows:
//
//	//go:embed schema/managed/*.sql
//	var schemaFs embed.FS
//
// SQL files should start with 'v' using a name that clearly indicates which version it belongs to, E.g., 'schema/managed/v0.0.1.sql'.
//
// See [WithRedisLocker], [WithLocker], [WithPrevErrIgnored].
func EnableSchemaMigrate(fs embed.FS, baseDir string, startVersion string, ops ...func(*schemaMigrateOptions)) {
	mysql.AddMySQLBootstrapCallback(func(rail miso.Rail, db *gorm.DB) error {
		op := &schemaMigrateOptions{}
		for _, f := range ops {
			f(op)
		}
		conf := svc.MigrateConfig{
			App:                 miso.GetPropStr(miso.PropAppName),
			Fs:                  fs,
			BaseDir:             baseDir,
			StartingVersion:     startVersion,
			IgnorePreviousError: op.IgnorePreviousError,
		}
		if op.Locker != nil {
			lc := LockContext{Rail: rail, App: conf.App}
			if err := op.Locker.Lock(lc); err != nil {
				return err
			}
			defer op.Locker.Unlock()
		}

		return svc.MigrateSchema(db, rail, conf)
	})
}

// Enable auto schema migration only in production mode.
//
// See [EnableSchemaMigrate].
func EnableSchemaMigrateOnProd(fs embed.FS, baseDir string, startVersion string, ops ...func(*schemaMigrateOptions)) {
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		if miso.IsProdMode() {
			EnableSchemaMigrate(fs, baseDir, startVersion, ops...)
		}
		return nil
	})
}

func ExcludeSchemaFile(name string) {
	svc.ExcludeFile(name)
}

type LockContext struct {
	Rail miso.Rail
	App  string
}

type Locker interface {
	Lock(lc LockContext) error
	Unlock()
}

// Use custom Locker for schema migration.
func WithLocker(l Locker) func(*schemaMigrateOptions) {
	return func(smo *schemaMigrateOptions) {
		smo.Locker = l
	}
}

// Ignore previous error if any.
func WithPrevErrIgnored() func(*schemaMigrateOptions) {
	return func(smo *schemaMigrateOptions) {
		smo.IgnorePreviousError = true
	}
}
