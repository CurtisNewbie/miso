package sqlite

import "gorm.io/gorm"

func TableHasColumnFunc(table string, column string) func(*gorm.DB) (ok bool, err error) {
	return func(d *gorm.DB) (ok bool, err error) {
		return TableHasColumn(table, column, d)
	}
}

func TableHasColumn(table string, column string, d *gorm.DB) (ok bool, err error) {
	var c int
	err = d.Raw(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&c).Error
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

func NotTableHasColumn(table string, column string, d *gorm.DB) (ok bool, err error) {
	v, e := TableHasColumn(table, column, d)
	return !v, e
}
