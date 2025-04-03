package sqlite

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: SQLite Configuration
const (

	// misoconfig-prop: path to SQLite database file
	PropSqliteFile = "sqlite.file"

	// misoconfig-prop: enable WAL mode | true
	PropSqliteWalEnabled = "sqlite.wal.enabled"
)

func init() {
	miso.SetDefProp(PropSqliteWalEnabled, true)
}
