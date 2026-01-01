package dbquery

import (
	"bytes"
	"context"
	"encoding/gob"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/async"
	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/json"
	"github.com/curtisnewbie/miso/util/rfutil"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/curtisnewbie/miso/util/strutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	contextKeyLogSQL    = "dbquery:log-sql"
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
	omitedColumns hash.Set[string]

	rail                 *miso.Rail
	logSQL               bool
	notLogSQL            bool
	notInsertModelFields bool
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
	if q.logSQL {
		cp = cp.LogSQL()
	}
	cp.notInsertModelFields = q.notInsertModelFields
	return cp
}

// Do not log current SQL statement.
func (q *Query) NotLogSQL() *Query {
	q.notLogSQL = true

	// statement is never nil, but just in case
	if q.tx.Statement != nil && q.tx.Statement.Context != nil {
		q.tx.Statement.Context = context.WithValue(q.tx.Statement.Context, contextKeyNotLogSQL, true) //lint:ignore SA1029 added a prefix already, should be fine
	}
	return q
}

// Log current SQL statement.
//
// [Query.LogSQL] has higher precedence over [Query.NotLogSQL].
func (q *Query) LogSQL() *Query {
	q.logSQL = true

	// statement is never nil, but just in case
	if q.tx.Statement != nil && q.tx.Statement.Context != nil {
		q.tx.Statement.Context = context.WithValue(q.tx.Statement.Context, contextKeyLogSQL, true) //lint:ignore SA1029 added a prefix already, should be fine
	}
	return q
}

// Do not automatically insert model fields.
//
// See [PrepareCreateModelHook] and [PrepareUpdateModelHook].
func (q *Query) NotInsertModelFields() *Query {
	q.notInsertModelFields = true
	return q
}

// Obtain Rail in this Query if there is any.
func (q *Query) Rail() (miso.Rail, bool) {
	if q.rail != nil {
		return *q.rail, true
	}
	return miso.Rail{}, false
}

// Table.
func (q *Query) From(table string) *Query {
	return q.Table(table)
}

// Table.
func (q *Query) Table(table string) *Query {
	q.tx = q.tx.Table(table)
	return q
}

// Add JOIN statements.
func (q *Query) Joins(query string, args ...any) *Query {
	q.tx = q.tx.Joins(query, args...)
	return q
}

// Add SELECT statements.
func (q *Query) Select(cols string, args ...any) *Query {
	q.tx = q.tx.Select(cols, args...)
	return q
}

// Add gorm clauses.
func (q *Query) Clauses(c ...clause.Expression) *Query {
	q.tx = q.tx.Clauses(c...)
	return q
}

// Add SELECT statements based on the given struct value.
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

// Get column name for given golang field name.
func (q *Query) ColumnName(s string) string {
	return q.DB().NamingStrategy.ColumnName("", s)
}

// Add WHERE statement.
func (q *Query) Where(query string, args ...any) *Query {
	q.tx = q.tx.Where(query, args...)
	return q
}

// Add IN (...) condition.
func (q *Query) In(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" IN ?", args...)
	return q
}

// Add NOT IN (...) condition.
func (q *Query) NotIn(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" NOT IN ?", args...)
	return q
}

// Scan and check if there is any record that matches the specified conditions.
func (q *Query) HasAny() (bool, error) {
	var v int
	n, err := q.Select("1").
		Limit(1).
		Scan(&v)
	return n > 0, err
}

// Equal to.
func (q *Query) Eq(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" = ?", args...)
	return q
}

// Not equal to.
func (q *Query) Ne(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" != ?", args...)
	return q
}

// Not equal to if true.
func (q *Query) NeIf(cond bool, col string, args ...any) *Query {
	if cond {
		q.tx = q.tx.Where(col+" != ?", args...)
	}
	return q
}

// Equal to if true.
func (q *Query) EqIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Eq(col, args...)
	}
	return q
}

// Equal to if v is not empty string.
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

// Less than or equal to.
func (q *Query) Le(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" <= ?", args...)
	return q
}

// Less than or equal to if true.
func (q *Query) LeIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Le(col, args...)
	}
	return q
}

// Less than.
func (q *Query) Lt(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" < ?", args...)
	return q
}

// Less than if true.
func (q *Query) LtIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Lt(col, args...)
	}
	return q
}

// Greater than or equal to.
func (q *Query) Ge(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" >= ?", args...)
	return q
}

// Greater than or equal to if true.
func (q *Query) GeIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Ge(col, args...)
	}
	return q
}

// Greater than.
func (q *Query) Gt(col string, args ...any) *Query {
	q.tx = q.tx.Where(col+" > ?", args...)
	return q
}

// Greater than if true.
func (q *Query) GtIf(cond bool, col string, args ...any) *Query {
	if cond {
		return q.Gt(col, args...)
	}
	return q
}

// Add IS NULL condition.
func (q *Query) IsNull(col string) *Query {
	q.tx = q.tx.Where(col + " IS NULL")
	return q
}

// Add IS NOT NULL condition.
func (q *Query) IsNotNull(col string) *Query {
	q.tx = q.tx.Where(col + " IS NOT NULL")
	return q
}

// Add BETWEEN ? AND ? condition.
func (q *Query) Between(col string, a any, b any) *Query {
	q.tx = q.tx.Where(col+" BETWEEN ? AND ?", a, b)
	return q
}

func (q *Query) WhereFunc(f func(*Query) *Query) *Query {
	q.tx = q.tx.Where(f(q.copyNew()).tx)
	return q
}

// Add AND (...) condition.
func (q *Query) And(f func(*Query) *Query) *Query {
	return q.WhereFunc(f)
}

// Run f if cond is true.
func (q *Query) If(cond bool, f func(*Query) *Query) *Query {
	if cond {
		return f(q)
	}
	return q
}

// Add WHERE if true.
func (q *Query) WhereIf(addWhere bool, query string, args ...any) *Query {
	if addWhere {
		return q.Where(query, args...)
	}
	return q
}

// Add WHERE if v is not nil.
func (q *Query) WhereNotNil(query string, v any) *Query {
	if rfutil.IsAnyNil(v) {
		return q
	}
	return q.Where(query, v)
}

// Add GROUP statement.
func (q *Query) Group(name string) *Query {
	q.tx = q.tx.Group(name)
	return q
}

// Add ORDER statement.
func (q *Query) Order(order string) *Query {
	q.tx = q.tx.Order(order)
	return q
}

// Add ORDER BY ? DESC.
func (q *Query) OrderDesc(col string) *Query {
	q.tx = q.tx.Order(col + " DESC")
	return q
}

// Add ORDER BY ? ASC.
func (q *Query) OrderAsc(col string) *Query {
	q.tx = q.tx.Order(col + " ASC")
	return q
}

// Same as [Query.Joins].
func (q *Query) Join(query string, args ...any) *Query {
	return q.Joins(query, args...)
}

// Add JOIN if true.
func (q *Query) JoinIf(addJoin bool, query string, args ...any) *Query {
	if addJoin {
		return q.Join(query, args...)
	}
	return q
}

// Add LIMIT.
func (q *Query) Limit(n int) *Query {
	q.tx = q.tx.Limit(n)
	return q
}

// Add OFFSET.
func (q *Query) Offset(n int) *Query {
	q.tx = q.tx.Offset(n)
	return q
}

// Add OFFSET and LIMIT for given page.
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

// Add raw SQL.
func (q *Query) Raw(sql string, args ...any) *Query {
	sql = strings.TrimSpace(sql)
	q.tx = q.tx.Raw(sql, args...)
	return q
}

// Add OR (...) condition if true.
func (q *Query) OrIf(cond bool, query string, args ...any) *Query {
	if cond {
		return q.Or(query, args...)
	}
	return q
}

// Add OR (...) condition.
func (q *Query) Or(query string, args ...any) *Query {
	q.tx = q.tx.Or(query, args...)
	return q
}

// Add OR (...) condition.
func (q *Query) OrFunc(f func(*Query) *Query) *Query {
	q.tx = q.tx.Or(f(q.copyNew()).tx)
	return q
}

// Run SQL and scan result.
//
// If ptr is of type [Nilable] (e.g., by embedding [NilableValue]), [Nilable.MarkZero] is automatically called based on rowsAffected.
func (q *Query) Scan(ptr any) (rowsAffected int64, err error) {
	tx := q.tx.Scan(ptr)
	rowsAffected = tx.RowsAffected
	err = errs.Wrap(tx.Error)
	if v, ok := ptr.(Nilable); ok && v != nil {
		v.MarkZero(rowsAffected < 1)
	}
	return
}

// Run SQL and scan result.
//
// If ptr is of type [Nilable] (e.g., by embedding [NilableValue]), [Nilable.MarkZero] is automatically called based on rowsAffected.
func (q *Query) ScanAny(ptr any) (ok bool, err error) {
	n, err := q.Scan(ptr)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Run SQL and scan result.
//
// If ptr is of type [Nilable] (e.g., by embedding [NilableValue]), [Nilable.MarkZero] is automatically called based on rowsAffected.
func (q *Query) ScanVal(ptr any) (err error) {
	_, err = q.Scan(ptr)
	return err
}

// Exec SQL.
func (q *Query) ExecAny(sql string, args ...any) error {
	_, err := q.Exec(sql, args...)
	return err
}

// Exec SQL.
func (q *Query) Exec(sql string, args ...any) (rowsAffected int64, err error) {
	sql = strings.TrimSpace(sql)
	tx := q.tx.Exec(sql, args...)
	rowsAffected = tx.RowsAffected
	err = errs.Wrap(tx.Error)
	return
}

// Exec UPDATE SQL.
func (q *Query) Update() (rowsAffected int64, err error) {
	if len(q.updateColumns) < 1 {
		return 0, nil
	}
	q.runUpdateHooks()
	tx := q.tx.Updates(q.updateColumns)
	rowsAffected = tx.RowsAffected
	err = errs.Wrap(tx.Error)
	return
}

// Exec UPDATE SQL.
func (q *Query) UpdateAny() error {
	_, err := q.Update()
	return err
}

// Add SET ? statements.
//
// UPDATE is not exected until one of [Query.Update] or [Query.UpdateAny] is called.
func (q *Query) Set(col string, arg any) *Query {
	q.updateColumns[col] = arg
	return q
}

// Run SQL and get COUNT(?) or COUNT(*) result.
func (q *Query) Count() (int64, error) {
	var n int64
	tx := q.tx.Count(&n)
	return n, errs.Wrap(tx.Error)
}

// Add multiple SET ? statements based on given struct / map value.
//
// UPDATE is not exected until one of [Query.Update] or [Query.UpdateAny] is called.
func (q *Query) SetCols(arg any, cols ...string) *Query {
	q.doSetCols(arg, true, cols...)
	return q
}

// Add multiple SET ? statements based on given struct / map value, ignore empty field.
//
// UPDATE is not exected until one of [Query.Update] or [Query.UpdateAny] is called.
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

// Add SET ? statements if true.
//
// UPDATE is not exected until one of [Query.Update] or [Query.UpdateAny] is called.
func (q *Query) SetIf(cond bool, col string, arg any) *Query {
	if cond {
		return q.Set(col, arg)
	}
	return q
}

// Run CREATE IGNORE to insert given value.
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

// Insert given value.
func (q *Query) CreateAny(v any) error {
	_, err := q.Create(v)
	return err
}

// Run CREATE IGNORE to insert given value.
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
	if q.omitedColumns.Has(ft.Name) { // MyField
		return
	}

	fname := q.ColumnName(ft.Name)
	if q.omitedColumns.Has(fname) { // my_field
		return
	}

	tagSet := schema.ParseTagSetting(ft.Tag.Get("gorm"), ";")
	if v, ok := tagSet["-"]; ok && strings.TrimSpace(v) == "-" {
		return
	}

	if c, ok := tagSet["COLUMN"]; ok {
		fname = c
	}
	if q.omitedColumns.Has(fname) { // my_field_alias
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

// Insert given value.
func (q *Query) Create(v any) (rowsAffected int64, err error) {
	rows := q.CreateInsertRowMaps(v)
	q.runCreateHooks(rows)
	if len(rows) < 1 {
		return 0, nil
	}
	tx := q.tx.Create(rows)
	rowsAffected = tx.RowsAffected
	err = errs.Wrap(tx.Error)
	return
}

// Exec DELETE.
func (q *Query) Delete() (rowsAffected int64, err error) {
	tx := q.tx.Delete(nil)
	rowsAffected = tx.RowsAffected
	err = errs.Wrap(tx.Error)
	return
}

// Exec DELETE.
func (q *Query) DeleteAny() error {
	_, err := q.Delete()
	return err
}

// Omit columns.
func (q *Query) Omit(col ...string) *Query {
	q.tx = q.tx.Omit(col...)
	q.omitedColumns.AddAll(col)
	return q
}

// Get underlying [*gorm.DB] .
func (q *Query) DB() *gorm.DB {
	return q.tx
}

func RunTransaction(rail miso.Rail, db *gorm.DB, callback func(qry func() *Query) error) error {
	return db.Transaction(func(db *gorm.DB) error {
		nq := func() *Query { return NewQuery(rail, db) }
		return callback(nq)
	})
}

// Create New [*Query].
//
// Param opts can be [*gorm.DB], [miso.Rail] or [context.Context].
//
// If [*gorm.DB] is missing, [GetDB] is called to obtain the primary one.
//
// If [miso.Rail] or [context.Context] is provided, tracing baggages (e.g., trace_id, span_id) are automatically propagated to the SQL logger registered in gorm.
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
	q := &Query{tx: db, _db: db, rail: r, updateColumns: map[string]any{}, omitedColumns: hash.NewSet[string]()}
	return q
}

func NewQueryFunc(table string, ops ...func(q *Query) *Query) func(r miso.Rail, db *gorm.DB) *Query {
	return func(r miso.Rail, db *gorm.DB) *Query {
		q := NewQuery(r, db).Table(table)
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
	selectQuery ChainedPageQuery          // Add SELECT query and ORDER BY query, e.g., return tx.Select(`*`).
	baseQuery   ChainedPageQuery          // Base query, e.g., return tx.Table(`myTable`).Where(...)
	mapTo       func(t V) V               // callback triggered on each record, the value returned will overwrite the value passed in.
	mapToAsync  func(t V) async.Future[V] // callback triggered on each record, the value returned will overwrite the value passed in.
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

func (pq *PageQuery[V]) Transform(t func(t V) V) *PageQuery[V] {
	pq.mapTo = t
	return pq
}

func (pq *PageQuery[V]) TransformAsync(t func(t V) async.Future[V]) *PageQuery[V] {
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
			return errs.Wrap(err)
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
			return errs.Wrap(err)
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

func (pq *PageQuery[V]) Scan(rail miso.Rail, reqPage miso.Paging) (miso.PageRes[V], error) {
	return pq.scan(rail, reqPage, true)
}

func (pq *PageQuery[V]) scan(rail miso.Rail, reqPage miso.Paging, doCount bool) (miso.PageRes[V], error) {
	newQuery := func() *Query {
		return pq.baseQuery(NewQuery(rail, pq.db))
	}

	var countFuture async.Future[int]
	if doCount {
		countFuture = async.Run(func() (int, error) {
			var total int
			_, err := newQuery().Select("COUNT(*)").Scan(&total)
			return total, err
		})
	}

	pageFuture := async.Run(func() ([]V, error) {
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
			futures := make([]async.Future[V], 0, len(payload))
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

type IterateByOffset1Param[V, Offset any] struct {
	Limit       int    // limit, by default 100
	OffsetCol   string // col name in ORDER BY (col)
	BuildQuery  func(rail miso.Rail, q *Query) *Query
	GetOffset   func(v V) Offset
	ForEachPage func(p []V) (stop bool, err error)
}

// Iterate all matched records ordered by column (OffsetCol).
//
// E.g.,
//
//	func ListRecords(rail miso.Rail, forEachPage func(v []Record) (err error)) error {
//		return dbquery.IterateAllByOffset1(rail, GetBI(), dbquery.IterateByOffset2Param[Record, atom.Time]{
//			OffsetCol: "rec_time",
//			Limit:      100,
//			BuildQuery: func(rail miso.Rail, q *dbquery.Query) *dbquery.Query {
//				return q.Table("my_table").
//					Eq("my_col", "").
//					SelectCols(Record{})
//			},
//			ForEachPage: func(p []Record) (stop bool, err error) {
//				return false, forEachPage(p)
//			},
//			GetOffset: func(v Record) (atom.Time) {
//				return v.RecTime
//			},
//		})
//	}
func IterateAllByOffset1[V any, Offset any](rail miso.Rail, db *gorm.DB, p IterateByOffset1Param[V, Offset]) error {
	caller := miso.GetCallerFn()
	rail.Infof("IterateAllByOffset1 '%v' start", caller)
	defer rail.Infof("IterateAllByOffset1 '%v' finished", caller)
	if p.Limit < 1 {
		p.Limit = 100
	}

	var offset Offset
	firstPage := true
	for {
		rail.Infof("IterateAllByOffset1 '%v', offset: %v", caller, offset)

		q := p.BuildQuery(rail, NewQuery(rail, db)).OrderAsc(p.OffsetCol).Limit(p.Limit)
		if firstPage {
			firstPage = false
		} else {
			q = q.Gt(p.OffsetCol, offset)
		}
		var l []V
		err := q.ScanVal(&l)
		if err != nil {
			return errs.Wrap(err)
		}
		if len(l) < 1 {
			return nil
		}
		stop, err := p.ForEachPage(l)
		if err != nil || stop {
			return err
		}
		if miso.IsShuttingDown() {
			return miso.ErrServerShuttingDown.New()
		}

		offset = p.GetOffset(l[len(l)-1])
	}
}

type IterateByOffset2Param[V, Offset1, Offset2 any] struct {
	Limit       int    // limit, by default 100
	OffsetCol1  string // col1 name in ORDER BY (col1, col2)
	OffsetCol2  string // col2 name in ORDER BY (col1, col2)
	BuildQuery  func(rail miso.Rail, q *Query) *Query
	GetOffset   func(v V) (Offset1, Offset2)
	ForEachPage func(p []V) (stop bool, err error)
}

// Iterate all matched records ordered by columns (OffsetCol1, OffsetCol2).
//
// E.g.,
//
//	func ListRecords(rail miso.Rail, forEachPage func(v []Record) (err error)) error {
//		return dbquery.IterateAllByOffset2(rail, GetBI(), dbquery.IterateByOffset2Param[Record, atom.Time, string]{
//			OffsetCol1: "rec_time",
//			OffsetCol2: "rec_id",
//			Limit:      100,
//			BuildQuery: func(rail miso.Rail, q *dbquery.Query) *dbquery.Query {
//				return q.Table("my_table").
//					Eq("my_col", "").
//					SelectCols(Record{})
//			},
//			ForEachPage: func(p []Record) (stop bool, err error) {
//				return false, forEachPage(p)
//			},
//			GetOffset: func(v Record) (atom.Time, string) {
//				return v.RecTime, v.RecId
//			},
//		})
//	}
func IterateAllByOffset2[V any, Offset1, Offset2 any](rail miso.Rail, db *gorm.DB, p IterateByOffset2Param[V, Offset1, Offset2]) error {
	caller := miso.GetCallerFn()
	rail.Infof("IterateAllByOffset2 '%v' start", caller)
	defer rail.Infof("IterateAllByOffset2 '%v' finished", caller)
	if p.Limit < 1 {
		p.Limit = 100
	}

	var offset1 Offset1
	var offset2 Offset2
	firstPage := true
	for {
		rail.Infof("IterateAllByOffset2 '%v', offset: (%v, %v)", caller, offset1, offset2)

		q := p.BuildQuery(rail, NewQuery(rail, db)).OrderAsc(p.OffsetCol1 + ", " + p.OffsetCol2).Limit(p.Limit)
		if firstPage {
			firstPage = false
		} else {
			q = q.Ge(p.OffsetCol1, offset1).
				Gt(p.OffsetCol2, offset2)
		}
		var l []V
		err := q.ScanVal(&l)
		if err != nil {
			return errs.Wrap(err)
		}
		if len(l) < 1 {
			return nil
		}
		stop, err := p.ForEachPage(l)
		if err != nil || stop {
			return err
		}
		if miso.IsShuttingDown() {
			return miso.ErrServerShuttingDown.New()
		}

		offset1, offset2 = p.GetOffset(l[len(l)-1])
	}
}

type IterateByOffsetParam[V, Offset any] struct {
	InitialOffset Offset
	FetchPage     func(rail miso.Rail, db *gorm.DB, offset Offset) ([]V, error)
	GetOffset     func(v V) Offset
	ForEach       func(v V) (stop bool, err error)
	ForEachPage   func(p []V) (stop bool, err error)
}

// Deprecated: Use [IterateAllByOffset1] or [IterateAllByOffset2] instead.
func IterateAllByOffset[V any, Offset any](rail miso.Rail, db *gorm.DB, p IterateByOffsetParam[V, Offset]) error {
	caller := miso.GetCallerFn()
	rail.Debugf("IterateAllByOffset '%v' start", caller)
	defer rail.Debugf("IterateAllByOffset '%v' finished", caller)
	offset := p.InitialOffset
	for {
		rail.Debugf("IterateAllByOffset '%v', offset: %v", caller, offset)
		l, err := p.FetchPage(rail, db, offset)
		if err != nil {
			return errs.Wrap(err)
		}
		if p.ForEachPage != nil {
			stop, err := p.ForEachPage(l)
			if err != nil || stop {
				return err
			}
		} else {
			for _, v := range l {
				stop, err := p.ForEach(v)
				if err != nil || stop {
					return err
				}
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

type IterateAllParam[V any] struct {
	Limit       int
	BuildQuery  func(rail miso.Rail, q *Query) *Query
	ForEachPage func(p []V) (stop bool, err error)
}

func (i *IterateAllParam[V]) scan(rail miso.Rail, db *gorm.DB) ([]V, error) {
	var l []V
	return l, i.BuildQuery(rail, NewQuery(rail, db)).Limit(i.Limit).ScanVal(&l)
}

func (i *IterateAllParam[V]) count(rail miso.Rail, db *gorm.DB) (int64, error) {
	return i.BuildQuery(rail, NewQuery(rail, db)).Count()
}

// Iterate all records until none left.
//
// Records should not be matched again in the next iteration once they are processed.
//
// Before looping pages, total count of records matched is checked to prevent infinite loop (just in case some records are never processed properly).
func IterateAll[V any](rail miso.Rail, db *gorm.DB, p IterateAllParam[V]) error {
	caller := miso.GetCallerFn()
	rail.Infof("IterateAll '%v' start", caller)
	defer rail.Infof("IterateAll '%v' finished", caller)
	cnt, err := p.count(rail, db)
	if err != nil {
		return err
	}
	if p.Limit < 1 {
		p.Limit = 100
	}
	maxRound := (cnt / int64(p.Limit))
	if cnt%int64(p.Limit) > 0 {
		maxRound += 1
	}
	for i := range maxRound {
		rail.Infof("IterateAll '%v', curr_round: %v, max_round: %v", caller, i+1, maxRound)
		l, err := p.scan(rail, db)
		if err != nil {
			return errs.Wrap(err)
		}
		if len(l) < 1 {
			return nil
		}
		stop, err := p.ForEachPage(l)
		if err != nil || stop {
			return err
		}
		if miso.IsShuttingDown() {
			return miso.ErrServerShuttingDown.New()
		}
	}
	return nil
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
