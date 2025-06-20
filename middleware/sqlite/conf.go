package sqlite

import "github.com/curtisnewbie/miso/miso"

// misoconfig-section: SQLite Configuration
const (

	// misoconfig-prop: path to SQLite database file
	PropSqliteFile = "sqlite.file"

	// misoconfig-prop: enable WAL mode | true
	PropSqliteWalEnabled = "sqlite.wal.enabled"

	// misoconfig-prop: log sql statements | false
	PropSqliteLogSQL = "sqlite.log-sql"
)

// misoconfig-default-start
func init() {
	miso.SetDefProp(PropSqliteWalEnabled, true)
	miso.SetDefProp(PropSqliteLogSQL, false)
}

// misoconfig-default-end
