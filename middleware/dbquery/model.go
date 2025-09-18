package dbquery

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/slutil"
)

var (
	CreatedByExtractor func(rail miso.Rail) string = func(rail miso.Rail) string {
		return rail.Username()
	}
	UpdatedByExtractor func(rail miso.Rail) string = func(rail miso.Rail) string {
		return rail.Username()
	}
)

// UpdateModel.
//
// This is for reference only. The UpdateModel Hooks will still work without this type.
type UpdateModel struct {
	UpdatedBy string
	TraceId   string
}

// CreateModel.
//
// Must be embedded as pointer otherwise the CreateModel hooks won't work.
type CreateModel struct {
	CreatedBy string
	TraceId   string
}

func (c *CreateModel) SetupCreateModel(cm CreateModel) {
	if c != nil {
		c.TraceId = cm.TraceId
		if c.CreatedBy == "" {
			c.CreatedBy = cm.CreatedBy
		}
	}
}

// Prepare CreateModel Hook that will run when [Query] call INSERT related methods, see [AddCreateHooks].
//
// Call this func before miso bootstraps.
//
// Usage: Register this hook, and Embed *CreateModel in your model type.
//
// E.g.,
//
//	type MyModel struct {
//		*CreateModel
//	}
//
// Then CreatedBy and TraceId are automatically set as part of INSERT SQL when you call [Query] methods.
func PrepareCreateModelHook() {
	AddCreateHooks(func(table string, q *Query, v any) {
		r, ok := q.Rail()
		if ok {
			if cm, ok := v.(interface {
				SetupCreateModel(CreateModel)
			}); ok {
				cm.SetupCreateModel(CreateModel{
					TraceId:   r.TraceId(),
					CreatedBy: r.Username(),
				})
			}
		}
	})
}

// Prepare UpdateModel Hook that will run when [Query] call UPDATED related methods, see [AddUpdateHooks].
//
// Return whether the hook should run for current table, e.g., some table may not have trace_id and updated_by fields.
//
// Call this func before miso bootstraps.
func PrepareUpdateModelHooks(optionalFn ...func(table string) (ok bool)) error {
	fn, ok := slutil.SliceFirst(optionalFn)
	if !ok {
		fn = func(_ string) (ok bool) { return true }
	}

	AddUpdateHooks(func(table string, q *Query) {
		r, ok := q.Rail()
		if ok {
			if ok := fn(table); ok {
				q.Set("trace_id", r.TraceId())

				if q.updateColumns != nil {
					if _, ok := q.updateColumns["updated_by"]; !ok {
						q.Set("updated_by", r.Username())
					}
				}
			}
		}
	})
	return nil
}
