package sqlite

import "github.com/curtisnewbie/miso/miso"

// Configuration properties for SQLite
const (
	PropSqliteFile       = "sqlite.file"
	PropSqliteWalEnabled = "sqlite.wal.enabled"
)

func init() {
	miso.SetDefProp(PropSqliteWalEnabled, true)
}
