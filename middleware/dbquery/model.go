package dbquery

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/slutil"
)

const (
	ColUpdatedBy = "updated_by"
	ColCreatedBy = "created_by"
	ColTraceId   = "trace_id"
)

var (
	CreatedByExtractor func(rail miso.Rail) string = func(rail miso.Rail) string {
		return rail.Username()
	}

	UpdatedByExtractor func(rail miso.Rail) string = func(rail miso.Rail) string {
		return rail.Username()
	}
)

// CreateModel.
//
// This is for reference only, it's not needed for [PrepareCreateModelHook].
type CreateModel struct {
	CreatedBy string
	TraceId   string
}

// UpdateModel.
//
// This is for reference only, it's not needed for [PrepareUpdateModelHook].
type UpdateModel struct {
	UpdatedBy string
	TraceId   string
}

// Prepare Hook that will run when [Query] call INSERT related methods, see [AddCreateHooks] and [CreateModel].
//
// The created_by and trace_id fields are extracted from trace and [CreatedByExtractor], and are automatically set as part of INSERT SQL when you call [Query] INSERT releated methods.
//
// Arg optionalFn should return whether the hook should run for current table, e.g., some tables may not have the trace_id and created_by fields.
//
// Call this func before miso bootstraps.
func PrepareCreateModelHook(optionalFn ...func(table string) (ok bool)) {
	fn, ok := slutil.First(optionalFn)
	if !ok {
		fn = func(_ string) (ok bool) { return true }
	}

	AddCreateHooks(func(table string, q *Query, insertRows []map[string]any) {
		if q.notInsertModelFields {
			return
		}
		if len(insertRows) < 1 {
			return
		}
		if !fn(table) {
			return
		}

		r, ok := q.Rail()
		if !ok {
			return
		}

		addInsertColStr := func(c string, v string, force bool) {
			for i, r := range insertRows {
				if force {
					r[c] = v
				} else {
					prev, ok := r[c]
					if !ok {
						// doesn't have the column at all
						r[c] = v
					} else {
						if prevStr, ok := prev.(string); ok && prevStr == "" {
							// column is empty for this row
							r[c] = v
						} else {
							// nothing to overwrite
							continue
						}
					}
				}
				insertRows[i] = r
			}
		}

		addInsertColStr(ColTraceId, r.TraceId(), true)
		addInsertColStr(ColCreatedBy, CreatedByExtractor(r), false)
	})
	miso.Info("Registered CreateModelHook")
}

// Prepare UpdateModel Hook that will run when [Query] call UPDATE related methods, see [AddUpdateHooks] and [UpdateModel].
//
// This hook will attempt to extract trace_id and updated_by field from trace ([miso.Rail]) and [UpdatedByExtractor], these fields are then updated to database along with other fields.
//
// Arg optionalFn should return whether the hook should run for current table, e.g., some tables may not have the trace_id and updated_by fields.
//
// By default, hooks are executed for all tables unless optionalFn is provided.
//
// Call this func before miso bootstraps.
func PrepareUpdateModelHook(optionalFn ...func(table string) (ok bool)) {
	fn, ok := slutil.First(optionalFn)
	if !ok {
		fn = func(_ string) (ok bool) { return true }
	}

	AddUpdateHooks(func(table string, q *Query) {
		if q.notInsertModelFields {
			return
		}
		if !fn(table) {
			return
		}
		if len(q.updateColumns) < 1 {
			return
		}

		r, ok := q.Rail()
		if ok {
			q.Set(ColTraceId, r.TraceId())

			if _, ok := q.updateColumns[ColUpdatedBy]; !ok {
				q.Set(ColUpdatedBy, UpdatedByExtractor(r))
			}
		}
	})
	miso.Info("Registered UpdateModelHook")
}
