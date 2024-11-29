package sqlite

import (
	"os"
	"testing"

	"github.com/curtisnewbie/miso/miso"
)

func TestGetSqlite(t *testing.T) {
	miso.SetLogLevel("debug")
	miso.SetProp(PropSqliteFile, "test.db")
	tx := GetDB()
	db, e := tx.DB()
	if e != nil {
		t.Error(e)
	}

	if e = db.Ping(); e != nil {
		t.Error(e)
	}

	v := tx.Exec(`
		create table if not exists dummy (
			id integer primary key autoincrement,
			name varchar(25) not null default ''
		)
		`)
	if v.Error != nil {
		t.Error(v.Error)
	}

	if e = os.Remove("test.db"); e != nil {
		miso.Infof("Failed to delete test.db, %v", e)
	}
	os.Remove("test.db-shm")
	os.Remove("test.db-wal")
}
