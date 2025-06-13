package dbquery

import (
	"reflect"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"gorm.io/gorm"
)

type Query struct {
	_db *gorm.DB
	tx  *gorm.DB

	updateColumns map[string]any
}

func (q *Query) CopyNew() *Query {
	return NewQuery(q._db)
}

// Same as *Query.Table().
//
// It was a mistake to call it From(), since we also use *Query to update tables :(
func (q *Query) From(table string) *Query {
	return q.Table(table)
}

func (q *Query) Table(table string) *Query {
	q.tx = q.tx.Table(table)
	return q
}

func (q *Query) Joins(query string, args ...any) *Query {
	q.tx = q.tx.Joins(query, args...)
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

// !=
func (q *Query) Ne(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" != ?", args...)
	return q
}

// !=
func (q *Query) NeIf(cond bool, col string, args ...any) *Query {
	if cond {
		q.tx = q.tx.Where(col+" != ?", args...)
	}
	return q
}

// =
func (q *Query) EqIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Eq(col, args...)
	}
	return q
}

// =
func (q *Query) EqNotEmpty(col string, v any) *Query {
	var cond bool = true
	switch vs := v.(type) {
	case string:
		if vs == "" {
			cond = false
		}
	case *string:
		if vs == nil || *vs == "" {
			cond = false
		}
	}
	return q.EqIf(cond, col, v)
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
	q.tx = q.tx.Where(f(q.CopyNew()).tx)
	return q
}

func (q *Query) And(f func(*Query) *Query) *Query {
	return q.WhereFunc(f)
}

func (q *Query) If(cond bool, f func(*Query) *Query) *Query {
	if cond {
		return f(q)
	}
	return q
}

func (q *Query) WhereIf(addWhere bool, query string, args ...any) *Query {
	if addWhere {
		return q.Where(query, args...)
	}
	return q
}

func (q *Query) WhereNotNil(query string, v any) *Query {
	if v != nil {
		return q.Where(query, v)
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
	q.tx = q.tx.Or(f(q.CopyNew()).tx)
	return q
}

func (q *Query) Scan(ptr any) (rowsAffected int64, err error) {
	tx := q.tx.Scan(ptr)
	rowsAffected = tx.RowsAffected
	err = tx.Error

	if v, ok := ptr.(Nilable); ok && v != nil {
		v.MarkZero(rowsAffected < 1)
	}
	return
}

func (q *Query) Exec(sql string, args ...any) (rowsAffected int64, err error) {
	tx := q.tx.Exec(sql, args...)
	rowsAffected = tx.RowsAffected
	err = tx.Error
	return
}

func (q *Query) Update() (rowsAffected int64, err error) {
	if len(q.updateColumns) < 1 {
		return 0, nil
	}
	tx := q.tx.Updates(q.updateColumns)
	rowsAffected = tx.RowsAffected
	err = tx.Error
	return
}

func (q *Query) Set(col string, arg any) *Query {
	q.updateColumns[col] = arg
	return q
}

func (q *Query) SetCols(arg any, cols ...string) *Query {
	if arg == nil {
		return q
	}

	rv := reflect.ValueOf(arg)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return q
	}

	colSet := util.NewSet[string]()
	for _, c := range cols {
		colSet.AddAll(strings.Split(c, ","))
	}
	colName := func(s string) string { return q.DB().NamingStrategy.ColumnName("", s) }

	rt := rv.Type()
	for i := range rv.NumField() {
		ft := rt.Field(i)
		fname := colName(ft.Name)
		if !colSet.IsEmpty() && !colSet.Has(fname) && !colSet.Has(ft.Name) {
			continue
		}

		fv := rv.Field(i)
		val, ok := util.ReflectBasicValue(fv)
		if ok {
			q.Set(fname, val)
		}
	}

	return q
}

func (q *Query) SetIf(cond bool, col string, arg any) *Query {
	if cond {
		return q.Set(col, arg)
	}
	return q
}

func (q *Query) Create(v any) (rowsAffected int64, err error) {
	tx := q.tx.Create(v)
	rowsAffected = tx.RowsAffected
	err = tx.Error
	return
}

func (q *Query) DB() *gorm.DB {
	return q.tx
}

func NewQuery(db *gorm.DB) *Query {
	return &Query{tx: db, _db: db, updateColumns: map[string]any{}}
}

type ChainedPageQuery func(q *Query) *Query

// Create param for page query.
type PageQuery[V any] struct {
	db          *gorm.DB
	selectQuery ChainedPageQuery       // Add SELECT query and ORDER BY query, e.g., return tx.Select(`*`).
	baseQuery   ChainedPageQuery       // Base query, e.g., return tx.Table(`myTable`).Where(...)
	mapTo       util.Transform[V]      // callback triggered on each record, the value returned will overwrite the value passed in.
	mapToAsync  util.TransformAsync[V] // callback triggered on each record, the value returned will overwrite the value passed in.
}

func NewPagedQuery[V any](db *gorm.DB) *PageQuery[V] {
	return &PageQuery[V]{db: db}
}

func (pq *PageQuery[V]) WithSelectQuery(qry ChainedPageQuery) *PageQuery[V] {
	pq.selectQuery = qry
	return pq
}

func (pq *PageQuery[V]) WithBaseQuery(qry ChainedPageQuery) *PageQuery[V] {
	pq.baseQuery = qry
	return pq
}

func (pq *PageQuery[V]) Transform(t util.Transform[V]) *PageQuery[V] {
	pq.mapTo = t
	return pq
}

func (pq *PageQuery[V]) TransformAsync(t util.TransformAsync[V]) *PageQuery[V] {
	pq.mapToAsync = t
	return pq
}

type IteratePageParam struct {
	Limit int `json:"limit" desc:"page limit"`
}

func (pq *PageQuery[V]) IterateAll(rail miso.Rail, param IteratePageParam, forEach func(v V) (stop bool, err error)) error {
	caller := miso.GetCallerFn()
	rail.Debugf("IterateAll '%v' start", caller)
	defer rail.Debugf("IterateAll '%v' finished", caller)
	if param.Limit < 1 {
		param.Limit = 1
	}
	p := miso.Paging{Page: 1, Limit: param.Limit}
	for {
		rail.Debugf("IterateAll '%v', page: %v", caller, p.Page)
		l, err := pq.scan(rail, p, false)
		if err != nil {
			return miso.WrapErr(err)
		}
		if len(l.Payload) < p.Limit {
			return nil
		}
		for _, l := range l.Payload {
			stop, err := forEach(l)
			if err != nil || stop {
				return err
			}
		}
		p.NextPage()
	}
}

func (pq *PageQuery[V]) Scan(rail miso.Rail, reqPage miso.Paging) (miso.PageRes[V], error) {
	return pq.scan(rail, reqPage, true)
}

func (pq *PageQuery[V]) scan(rail miso.Rail, reqPage miso.Paging, doCount bool) (miso.PageRes[V], error) {
	newQuery := func() *Query {
		return pq.baseQuery(NewQuery(pq.db))
	}

	var countFuture util.Future[int]
	if doCount {
		countFuture = util.RunAsync(func() (int, error) {
			var total int
			_, err := newQuery().Select("COUNT(*)").Scan(&total)
			return total, err
		})
	}

	pageFuture := util.RunAsync(func() ([]V, error) {
		var payload []V
		qry := newQuery()
		if pq.selectQuery != nil {
			qry = pq.selectQuery(qry)
		}
		_, err := qry.Offset(reqPage.GetOffset()).
			Limit(reqPage.GetLimit()).
			Scan(&payload)
		if err != nil {
			return nil, err
		}

		if pq.mapTo != nil {
			for i := range payload {
				payload[i] = pq.mapTo(payload[i])
			}
		}

		if pq.mapToAsync != nil {
			futures := make([]util.Future[V], 0, len(payload))
			for _, p := range payload {
				futures = append(futures, pq.mapToAsync(p))
			}
			for i := range payload {
				v, err := futures[i].Get()
				if err != nil {
					rail.Warnf("Failed to resolve Future, skipped %v", err)
					continue
				}
				payload[i] = v
			}
		}
		return payload, nil
	})

	var res miso.PageRes[V]
	var total int
	if doCount {
		if t, err := countFuture.Get(); err != nil {
			return res, err
		} else {
			total = t
		}
	}

	payload, err := pageFuture.Get()
	if err != nil {
		return res, err
	}

	res = miso.PageRes[V]{Payload: payload, Page: miso.RespPage(reqPage, total)}
	return res, nil
}

type Nilable interface {
	IsZero() bool
	MarkZero(isZero bool)
}

var (
	_ Nilable = (*NilableValue)(nil)
)

type NilableValue struct {
	zero bool
}

func (n *NilableValue) IsZero() bool {
	return n.zero
}

func (n *NilableValue) MarkZero(isZero bool) {
	n.zero = isZero
}
