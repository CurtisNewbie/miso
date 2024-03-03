package migrate

import (
	"embed"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/svc"
	"gorm.io/gorm"
)

// Enable auto schema migration.
//
// Put all SQL scripts in a specific directory. Embed the directory as follows:
//
//	//go:embed schema/managed/*.sql
//	var schemaFs embed.FS
//
// SQL files should start with 'v' using a name that clearly indicates which version it belongs to, E.g., 'schema/managed/v0.0.1.sql'.
func EnableSchemaMigrate(fs embed.FS, baseDir string, startVersion string) {
	miso.AddMySQLBootstrapCallback(func(rail miso.Rail, db *gorm.DB) error {
		conf := svc.MigrateConfig{
			App:             miso.GetPropStr(miso.PropAppName),
			Fs:              fs,
			BaseDir:         baseDir,
			StartingVersion: startVersion,
		}
		return svc.MigrateSchema(db, rail, conf)
	})
}

// Enable auto schema migration on production mode.
func EnableSchemaMigrateOnProd(fs embed.FS, baseDir string, startVersion string) {
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		if miso.IsProdMode() {
			EnableSchemaMigrate(fs, baseDir, startVersion)
		}
		return nil
	})
}

func ExcludeSchemaFile(name string) {
	svc.ExcludeFile(name)
}
