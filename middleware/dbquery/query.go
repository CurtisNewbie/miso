package dbquery

import (
	"bytes"
	"context"
	"encoding/gob"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/rfutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	contextKeyNotLogSQL = "dbquery:not-log-sql"
)

var (
	typeColCache = sync.Map{}

	updateHooks = slutil.NewSyncSlice[func(table string, q *Query)](0)
	createHooks = slutil.NewSyncSlice[func(table string, q *Query, v []map[string]any)](0)
)

type Query struct {
	_db           *gorm.DB
	tx            *gorm.DB
	updateColumns map[string]any

	rail      *miso.Rail
	notLogSQL bool
}

func (q *Query) copyNew() *Query {
	r, ok := q.Rail()
	var cp *Query
	if ok {
		cp = NewQuery(r, q._db)
	} else {
		cp = NewQuery(q._db)
	}
	if q.notLogSQL {
		cp = cp.NotLogSQL()
	}
	return cp
}

func (q *Query) NotLogSQL() *Query {
	q.notLogSQL = true

	// statement is never nil, but just in case
	if q.tx.Statement != nil && q.tx.Statement.Context != nil {
		q.tx.Statement.Context = context.WithValue(q.tx.Statement.Context, contextKeyNotLogSQL, true)
	}
	return q
}

func (q *Query) Rail() (miso.Rail, bool) {
	if q.rail != nil {
		return *q.rail, true
	}
	return miso.Rail{}, false
}

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

func (q *Query) Clauses(c ...clause.Expression) *Query {
	q.tx = q.tx.Clauses(c...)
	return q
}

func (q *Query) SelectCols(v any) *Query {
	if v == nil {
		return q
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return q
	}
	if rv.NumField() < 1 {
		return q
	}

	rt := rv.Type()
	if selected, ok := typeColCache.Load(rt); ok {
		return q.Select(selected.(string))
	}

	colSet := hash.NewSet[string]()
	for i := range rt.NumField() {
		q.selectFields(colSet, rt.Field(i))
	}
	selected := strings.Join(colSet.CopyKeys(), ",")
	typeColCache.Store(rt, selected)
	return q.Select(selected)
}

func (q *Query) selectFields(colSet hash.Set[string], ft reflect.StructField) {

	tagSet := schema.ParseTagSetting(ft.Tag.Get("gorm"), ";")
	if v, ok := tagSet["-"]; ok && strings.TrimSpace(v) == "-" {
		return
	}

	if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
		for j := range ft.Type.NumField() {
			q.selectFields(colSet, ft.Type.Field(j))
		}
		return
	}

	fname := q.ColumnName(ft.Name)
	if c, ok := tagSet["COLUMN"]; ok {
		fname = c
	}

	colSet.Add(fname)
}

func (q *Query) ColumnName(s string) string {
	return q.DB().NamingStrategy.ColumnName("", s)
}

func (q *Query) Where(query string, args ...any) *Query {
	q.tx = q.tx.Where(query, args...)
	return q
}

func (q *Query) In(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" IN ?", args...)
	return q
}

func (q *Query) NotIn(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" NOT IN ?", args...)
	return q
}

func (q *Query) HasAny() (bool, error) {
	var v int
	n, err := q.Select("1").
		Limit(1).
		Scan(&v)
	return n > 0, err
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
	q.tx = q.tx.Where(f(q.copyNew()).tx)
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
	if rfutil.IsAnyNil(v) {
		return q
	}
	return q.Where(query, v)
}

func (q *Query) Group(name string) *Query {
	q.tx = q.tx.Group(name)
	return q
}

func (q *Query) Order(order string) *Query {
	q.tx = q.tx.Order(order)
	return q
}

func (q *Query) OrderDesc(col string) *Query {
	q.tx = q.tx.Order(col + " DESC")
	return q
}

func (q *Query) OrderAsc(col string) *Query {
	q.tx = q.tx.Order(col + " ASC")
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

func (q *Query) AtPage(p miso.Paging) *Query {
	return q.Limit(p.GetLimit()).Offset(p.GetOffset())
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
	sql = strings.TrimSpace(sql)
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
	q.tx = q.tx.Or(f(q.copyNew()).tx)
	return q
}

func (q *Query) Scan(ptr any) (rowsAffected int64, err error) {
	tx := q.tx.Scan(ptr)
	rowsAffected = tx.RowsAffected
	err = errs.WrapErr(tx.Error)
	if v, ok := ptr.(Nilable); ok && v != nil {
		v.MarkZero(rowsAffected < 1)
	}
	return
}

func (q *Query) ScanAny(ptr any) (ok bool, err error) {
	n, err := q.Scan(ptr)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (q *Query) ScanVal(ptr any) (err error) {
	_, err = q.Scan(ptr)
	return err
}

func (q *Query) ExecAny(sql string, args ...any) error {
	_, err := q.Exec(sql, args...)
	return err
}

func (q *Query) Exec(sql string, args ...any) (rowsAffected int64, err error) {
	sql = strings.TrimSpace(sql)
	tx := q.tx.Exec(sql, args...)
	rowsAffected = tx.RowsAffected
	err = errs.WrapErr(tx.Error)
	return
}

func (q *Query) Update() (rowsAffected int64, err error) {
	if len(q.updateColumns) < 1 {
		return 0, nil
	}
	q.runUpdateHooks()
	tx := q.tx.Updates(q.updateColumns)
	rowsAffected = tx.RowsAffected
	err = errs.WrapErr(tx.Error)
	return
}

func (q *Query) UpdateAny() error {
	_, err := q.Update()
	return err
}

func (q *Query) Set(col string, arg any) *Query {
	q.updateColumns[col] = arg
	return q
}

func (q *Query) Count() (int64, error) {
	var n int64
	tx := q.tx.Count(&n)
	return n, errs.WrapErr(tx.Error)
}

func (q *Query) SetCols(arg any, cols ...string) *Query {
	q.doSetCols(arg, true, cols...)
	return q
}

func (q *Query) SetColsIgnoreEmpty(arg any, cols ...string) *Query {
	q.doSetCols(arg, false, cols...)
	return q
}

func (q *Query) doSetCols(arg any, inclEmpty bool, cols ...string) *Query {
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

	colSet := hash.NewSet[string]()
	for _, c := range cols {
		colSet.AddAll(strutil.SplitStr(c, ","))
	}

	rt := rv.Type()
	for i := range rv.NumField() {
		ft := rt.Field(i)
		fv := rv.Field(i)
		q.setField(colSet, ft, fv, inclEmpty)
	}

	return q
}

func (q *Query) setField(colSet hash.Set[string], ft reflect.StructField, fv reflect.Value, inclEmpty bool) {
	fname := q.ColumnName(ft.Name)

	tagSet := schema.ParseTagSetting(ft.Tag.Get("gorm"), ";")
	if v, ok := tagSet["-"]; ok && strings.TrimSpace(v) == "-" {
		return
	}

	nameAlias := fname
	if c, ok := tagSet["COLUMN"]; ok {
		nameAlias = c
	}

	if !colSet.IsEmpty() && !colSet.Has(fname) && !colSet.Has(ft.Name) && !colSet.Has(nameAlias) {
		return // specified column names explicitly, check if it's in the name set
	}
	fname = nameAlias

	// embedded struct
	if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
		for i := range ft.Type.NumField() {
			q.setField(colSet, ft.Type.Field(i), fv.Field(i), inclEmpty)
		}
		return
	}

	if !inclEmpty {
		switch fv.Kind() {
		case reflect.Pointer:
			if fv.IsNil() {
				return
			}
			ele := fv.Elem()
			if ele.Kind() == reflect.String && ele.Interface().(string) == "" {
				return
			}
		case reflect.String:
			if fv.Interface().(string) == "" {
				return
			}
		}
	}

	var val any
	switch fv.Kind() {
	case reflect.Pointer:
		if fv.IsNil() {
			val = nil
		} else {
			val = fv.Elem().Interface()
		}
	default:
		val = fv.Interface()
	}

	if val != nil {
		if v, ok := q.serializeValueWithTagSet(tagSet, fv, val); ok {
			val = v
		}
	}

	q.Set(fname, val)
}

func (q *Query) SetIf(cond bool, col string, arg any) *Query {
	if cond {
		return q.Set(col, arg)
	}
	return q
}

func (q *Query) CreateIgnoreAny(v any) error {
	_, err := q.CreateIgnore(v)
	return err
}

func (q *Query) runCreateHooks(v []map[string]any) {
	createHooks.ForEach(func(f func(string, *Query, []map[string]any)) (stop bool) {
		f(q.stmtTable(), q, v)
		return false
	})
}

func (q *Query) runUpdateHooks() {
	updateHooks.ForEach(func(f func(string, *Query)) (stop bool) {
		f(q.stmtTable(), q)
		return false
	})
}

func (q *Query) stmtTable() string {
	table := ""
	if q.tx.Statement != nil {
		table = q.tx.Statement.Table
	}
	return table
}

func (q *Query) CreateAny(v any) error {
	_, err := q.Create(v)
	return err
}

func (q *Query) CreateIgnore(v any) (rowsAffected int64, err error) {
	q.tx = q.tx.Clauses(clause.Insert{Modifier: "IGNORE"})
	return q.Create(v)
}

func (q *Query) insertOneRowMaps(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return map[string]any{}
	}
	rv = reflect.Indirect(rv)

	if cv, ok := rv.Interface().(map[string]any); ok {
		return cv
	}

	m := map[string]any{}
	if rv.Kind() == reflect.Map {
		if rv.Type().Key().Kind() != reflect.String {
			return m
		}
		// cast map[string]? to map[string]any
		mr := rv.MapRange()
		for mr.Next() {
			k := mr.Key().Interface().(string)
			v := mr.Value().Interface()
			m[k] = v
		}
		return m
	}

	if rv.Kind() != reflect.Struct {
		return m
	}

	rt := rv.Type()
	for i := range rv.NumField() {
		ft := rt.Field(i)
		fv := rv.Field(i)
		q.setInsertRowMap(m, ft, fv)
	}

	return m
}

func (q *Query) CreateInsertRowMaps(v any) []map[string]any {
	m := []map[string]any{}
	if v == nil {
		return m
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return m
	}
	rv = reflect.Indirect(rv)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			ele := rv.Index(i)
			m = append(m, q.insertOneRowMaps(ele.Interface()))
		}
	default:
		m = append(m, q.insertOneRowMaps(v))
	}
	return m
}

func (q *Query) setInsertRowMap(m map[string]any, ft reflect.StructField, fv reflect.Value) {
	fname := q.ColumnName(ft.Name)

	tagSet := schema.ParseTagSetting(ft.Tag.Get("gorm"), ";")
	if v, ok := tagSet["-"]; ok && strings.TrimSpace(v) == "-" {
		return
	}

	// embedded struct
	if ft.Anonymous && ft.Type.Kind() == reflect.Struct {
		for i := range ft.Type.NumField() {
			q.setInsertRowMap(m, ft.Type.Field(i), fv.Field(i))
		}
		return
	}

	var val any
	switch fv.Kind() {
	case reflect.Pointer:
		if fv.IsNil() {
			val = nil
		} else {
			val = fv.Elem().Interface()
		}
	default:
		val = fv.Interface()
	}

	if val != nil {
		if v, ok := q.serializeValueWithTagSet(tagSet, fv, val); ok {
			val = v
		}
	}

	if c, ok := tagSet["COLUMN"]; ok {
		fname = c
	}

	m[fname] = val
}

func (q *Query) serializeValueWithTagSet(tagSet map[string]string, fv reflect.Value, val any) (any, bool) {
	var serializerName = tagSet["JSON"]
	if serializerName == "" {
		serializerName = tagSet["SERIALIZER"]
	}
	if v, ok := q.serializeValue(serializerName, fv, val); ok {
		return v, true
	}
	return val, false
}

func (q *Query) serializeValue(serializer string, fv reflect.Value, val any) (any, bool) {
	// support default gob, unixtime serializer
	// default json serializer is replaced with MisoJSONSerializer
	switch serializer {
	case "json":
		vs, err := json.SWriteJson(val)
		if err == nil {
			return vs, true
		}
	case "gob":
		buf := new(bytes.Buffer)
		err := gob.NewEncoder(buf).Encode(val)
		if err == nil {
			return buf.Bytes(), true
		}
	case "unixtime":
		switch v := val.(type) {
		case int64, int, uint, uint64, int32, uint32, int16, uint16, *int64, *int, *uint, *uint64, *int32, *uint32, *int16, *uint16:
			val = time.Unix(reflect.Indirect(reflect.ValueOf(v)).Int(), 0)
			return val, true
		}
	default:
		ser, ok := schema.GetSerializer(serializer)
		if ok {
			// last resort, unfortunately, field can't be provided for now
			ser.Value(context.Background(), nil, fv, val)
		}
	}
	return val, false
}

func (q *Query) Create(v any) (rowsAffected int64, err error) {
	rows := q.CreateInsertRowMaps(v)
	q.runCreateHooks(rows)
	if len(rows) < 1 {
		return 0, nil
	}
	tx := q.tx.Create(rows)
	rowsAffected = tx.RowsAffected
	err = errs.WrapErr(tx.Error)
	return
}

func (q *Query) Delete() (rowsAffected int64, err error) {
	tx := q.tx.Delete(nil)
	rowsAffected = tx.RowsAffected
	err = errs.WrapErr(tx.Error)
	return
}

func (q *Query) DeleteAny() error {
	_, err := q.Delete()
	return err
}

func (q *Query) Omit(col ...string) *Query {
	q.tx = q.tx.Omit(col...)
	return q
}

func (q *Query) DB() *gorm.DB {
	return q.tx
}

func RunTransaction(rail miso.Rail, db *gorm.DB, callback func(qry func() *Query) error) error {
	return db.Transaction(func(db *gorm.DB) error {
		nq := func() *Query { return NewQueryRail(rail, db) }
		return callback(nq)
	})
}

// Create New *Query.
//
// opts can be [*gorm.DB], [miso.Rail] or [context.Context].
//
// If *gorm.DB is missing, [GetDB] is called to obtain the primary one.
func NewQuery(opts ...any) *Query {
	var (
		db *gorm.DB
		r  *miso.Rail
		c  context.Context
	)
	for _, o := range opts {
		cp := o
		switch v := cp.(type) {
		case *gorm.DB:
			if db == nil {
				db = v
			}
		case miso.Rail:
			if r == nil {
				r = &v
			}
		case *miso.Rail:
			if r == nil {
				r = v
			}
		case context.Context:
			if c == nil {
				c = v
			}
		}
	}
	if db == nil {
		db = GetDB()
	}
	if r != nil {
		db = db.WithContext(r.Context())
	} else if c != nil {
		db = db.WithContext(c)
	}
	q := &Query{tx: db, _db: db, rail: r, updateColumns: map[string]any{}}
	return q
}

func NewQueryRail(r miso.Rail, db *gorm.DB) *Query {
	return NewQuery(r, db)
}

func NewQueryFunc(table string, ops ...func(q *Query) *Query) func(r miso.Rail, db *gorm.DB) *Query {
	return func(r miso.Rail, db *gorm.DB) *Query {
		q := NewQueryRail(r, db).Table(table)
		for _, op := range ops {
			q = op(q)
		}
		return q
	}
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
			return errs.WrapErr(err)
		}
		for _, l := range l.Payload {
			stop, err := forEach(l)
			if err != nil || stop {
				return err
			}
		}
		if len(l.Payload) < p.Limit {
			return nil
		}
		if miso.IsShuttingDown() {
			return miso.ErrServerShuttingDown.New()
		}

		p.NextPage()
	}
}

func (pq *PageQuery[V]) IterateAllPages(rail miso.Rail, param IteratePageParam, forEachPage func(v []V) (stop bool, err error)) error {
	caller := miso.GetCallerFn()
	rail.Debugf("IterateAllPages '%v' start", caller)
	defer rail.Debugf("IterateAllPages '%v' finished", caller)
	if param.Limit < 1 {
		param.Limit = 1
	}
	p := miso.Paging{Page: 1, Limit: param.Limit}
	for {
		rail.Debugf("IterateAllPages '%v', page: %v", caller, p.Page)
		l, err := pq.scan(rail, p, false)
		if err != nil {
			return errs.WrapErr(err)
		}
		stop, err := forEachPage(l.Payload)
		if err != nil || stop {
			return err
		}
		if len(l.Payload) < p.Limit {
			return nil
		}
		if miso.IsShuttingDown() {
			return miso.ErrServerShuttingDown.New()
		}

		p.NextPage()
	}
}

type IterateByOffsetParam[V, T any] struct {
	Limit         int
	InitialOffset T
	FetchPage     func(rail miso.Rail, db *gorm.DB, offset T) ([]V, error)
	GetOffset     func(v V) T
	ForEach       func(v V) (stop bool, err error)
}

func IterateAllByOffset[V any, T any](rail miso.Rail, db *gorm.DB, p IterateByOffsetParam[V, T]) error {
	caller := miso.GetCallerFn()
	rail.Debugf("IterateAllByOffset '%v' start", caller)
	defer rail.Debugf("IterateAllByOffset '%v' finished", caller)
	if p.Limit < 1 {
		p.Limit = 1
	}
	offset := p.InitialOffset
	for {
		rail.Debugf("IterateAllByOffset '%v', offset: %v", caller, offset)
		l, err := p.FetchPage(rail, db, offset)
		if err != nil {
			return errs.WrapErr(err)
		}
		for _, l := range l {
			stop, err := p.ForEach(l)
			if err != nil || stop {
				return err
			}
		}
		if len(l) < 1 {
			return nil
		}
		if miso.IsShuttingDown() {
			return miso.ErrServerShuttingDown.New()
		}

		offset = p.GetOffset(l[len(l)-1])
	}
}

func (pq *PageQuery[V]) Scan(rail miso.Rail, reqPage miso.Paging) (miso.PageRes[V], error) {
	return pq.scan(rail, reqPage, true)
}

func (pq *PageQuery[V]) scan(rail miso.Rail, reqPage miso.Paging, doCount bool) (miso.PageRes[V], error) {
	newQuery := func() *Query {
		return pq.baseQuery(NewQueryRail(rail, pq.db))
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

/*
func isValueKind(v reflect.Value) (any, bool) {
	k := v.Kind()
	switch k {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8,
		reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.String, reflect.Complex64,
		reflect.Complex128:
		return v.Interface(), true
	}
	if _, ok := v.Interface().(driver.Valuer); ok {
		return v.Interface(), true
	}
	return nil, false
}

func reflectValue(rv reflect.Value) (any, bool) {
	if v, ok := isValueKind(rv); ok {
		return v, true
	}
	ftk := rv.Kind()
	if ftk == reflect.Pointer {
		if rv.IsNil() {
			return nil, true
		}

		rve := rv.Elem()
		if v, ok := isValueKind(rve); ok {
			return v, true
		}
	}
	return nil, false
}
*/

func ExecSQL(rail miso.Rail, db *gorm.DB, sql string, args ...any) error {
	return NewQuery(rail, db).ExecAny(sql, args...)
}

// Register hooks for [Query.Create], [Query.CreateAny] and [Query.CreateIgnore].
func AddCreateHooks(f func(table string, q *Query, v []map[string]any)) {
	createHooks.Append(f)
}

// Register hooks for [Query.Update] and [Query.UpdateAny].
func AddUpdateHooks(f func(table string, q *Query)) {
	updateHooks.Append(f)
}
