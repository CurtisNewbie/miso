package sqlite

import "github.com/curtisnewbie/miso/miso"

// misoapi-config-section: SQLite Configuration
const (

	// misoapi-config: path to SQLite database file
	PropSqliteFile = "sqlite.file"

	// misoapi-config: enable WAL mode | true
	PropSqliteWalEnabled = "sqlite.wal.enabled"
)

func init() {
	miso.SetDefProp(PropSqliteWalEnabled, true)
}
