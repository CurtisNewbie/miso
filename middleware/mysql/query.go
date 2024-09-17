package mysql

import (
	"gorm.io/gorm"
)

type Query struct {
	_db *gorm.DB
	tx  *gorm.DB
}

func (q *Query) cloneNew() *Query {
	return NewQuery(q._db)
}

func (q *Query) From(table string) *Query {
	q.tx = q.tx.Table(table)
	return q
}

func (q *Query) Select(cols string, args ...any) *Query {
	q.tx = q.tx.Select(cols, args...)
	return q
}

func (q *Query) Where(query string, args ...any) *Query {
	q.tx = q.tx.Where(query, args...)
	return q
}

// =
func (q *Query) Eq(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" = ?", args...)
	return q
}

// =
func (q *Query) EqIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Eq(col, args...)
	}
	return q
}

// <=
func (q *Query) Le(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" <= ?", args...)
	return q
}

// <=
func (q *Query) LeIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Le(col, args...)
	}
	return q
}

// <
func (q *Query) Lt(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" < ?", args...)
	return q
}

// <
func (q *Query) LtIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Lt(col, args...)
	}
	return q
}

// >=
func (q *Query) Ge(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" >= ?", args...)
	return q
}

// >=
func (q *Query) GeIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Ge(col, args...)
	}
	return q
}

// >
func (q *Query) Gt(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" > ?", args...)
	return q
}

// >
func (q *Query) GtIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Gt(col, args...)
	}
	return q
}

func (q *Query) IsNull(col string) *Query {
	q.tx = q.tx.Where(col + " IS NULL")
	return q
}

func (q *Query) IsNotNull(col string) *Query {
	q.tx = q.tx.Where(col + " IS NOT NULL")
	return q
}

func (q *Query) Between(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" BETWEEN ? AND ?", args...)
	return q
}

func (q *Query) WhereFunc(f func(*Query) *Query) *Query {
	q.tx = q.tx.Where(f(q.cloneNew()).tx)
	return q
}

func (q *Query) WhereIf(addWhere bool, query string, args ...any) *Query {
	if addWhere {
		return q.Where(query, args...)
	}
	return q
}

func (q *Query) Group(name string) *Query {
	q.tx = q.tx.Group(name)
	return q
}

func (q *Query) Order(order string) *Query {
	q.tx = q.tx.Order(order)
	return q
}

func (q *Query) Join(query string, args ...any) *Query {
	q.tx = q.tx.Joins(query, args...)
	return q
}

func (q *Query) JoinIf(addJoin bool, query string, args ...any) *Query {
	if addJoin {
		return q.Join(query, args...)
	}
	return q
}

func (q *Query) Limit(n int) *Query {
	q.tx = q.tx.Limit(n)
	return q
}

func (q *Query) Offset(n int) *Query {
	q.tx = q.tx.Offset(n)
	return q
}

// LIKE '%?'
func (q *Query) LikeLeftIf(cond bool, col string, val string) *Query {
	if cond {
		return q.LikeLeft(col, val)
	}
	return q
}

// LIKE '%?'
func (q *Query) LikeLeft(col string, val string) *Query {
	return q.Where(col+" LIKE ?", "%"+val)
}

// LIKE '?%'
func (q *Query) LikeRightIf(cond bool, col string, val string) *Query {
	if cond {
		return q.LikeRight(col, val)
	}
	return q
}

// LIKE '?%'
func (q *Query) LikeRight(col string, val string) *Query {
	return q.Where(col+" LIKE ?", val+"%")
}

// LIKE '%?%'
func (q *Query) LikeIf(cond bool, col string, val string) *Query {
	if cond {
		return q.Like(col, val)
	}
	return q
}

// LIKE '%?%'
func (q *Query) Like(col string, val string) *Query {
	return q.Where(col+" LIKE ?", "%"+val+"%")
}

func (q *Query) Raw(sql string, args ...any) *Query {
	q.tx = q.tx.Raw(sql, args...)
	return q
}

func (q *Query) OrIf(cond bool, query string, args ...any) *Query {
	if cond {
		return q.Or(query, args...)
	}
	return q
}

func (q *Query) Or(query string, args ...any) *Query {
	q.tx = q.tx.Or(query, args...)
	return q
}

func (q *Query) OrFunc(f func(*Query) *Query) *Query {
	q.tx = q.tx.Or(f(q.cloneNew()).tx)
	return q
}

func (q *Query) Scan(ptr any) (rowsAffected int64, err error) {
	tx := q.tx.Scan(ptr)
	rowsAffected = tx.RowsAffected
	err = tx.Error
	return
}

func (q *Query) Exec(sql string, args ...any) (rowsAffected int64, err error) {
	tx := q.tx.Exec(sql, args...)
	rowsAffected = tx.RowsAffected
	err = tx.Error
	return
}

func (q *Query) DB() *gorm.DB {
	return q.tx
}

func NewQuery(db *gorm.DB) *Query {
	return &Query{tx: db, _db: db}
}
