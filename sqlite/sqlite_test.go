package sqlite

import (
	"os"
	"testing"

	"github.com/curtisnewbie/miso/core"
	"github.com/sirupsen/logrus"
)

func TestGetSqlite(t *testing.T) {
	core.SetProp(core.PROP_SQLITE_FILE, "test.db")
	tx := GetSqlite()
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
		logrus.Infof("Failed to delete test.db, %v", e)
	}

}
