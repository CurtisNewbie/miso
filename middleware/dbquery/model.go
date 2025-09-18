package dbquery

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/slutil"
)

var (
	// createModelType = reflect.TypeOf(CreateModel{})
	// CreatedByExtractor func(rail miso.Rail) string = func(rail miso.Rail) string {
	// 	return rail.Username()
	// }

	UpdatedByExtractor func(rail miso.Rail) string = func(rail miso.Rail) string {
		return rail.Username()
	}
)

// CreateModel.
type CreateModel struct {
	CreatedBy string
	TraceId   string
}

/*

func (c *CreateModel) SetupCreateModel(cm CreateModel) bool {
	if c != nil {
		c.TraceId = cm.TraceId
		if c.CreatedBy == "" {
			c.CreatedBy = cm.CreatedBy
		}
		return true
	}
	return false
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

			setupCreateModel := func(v any) {
				if v == nil {
					return
				}

				target, ok := v.(interface {
					SetupCreateModel(CreateModel) bool
				})
				if !ok {
					return
				}

				cm := CreateModel{
					TraceId:   r.TraceId(),
					CreatedBy: r.Username(),
				}
				done := target.SetupCreateModel(cm)
				if done {
					return
				}

				rv := reflect.ValueOf(v)
				if rv.Kind() != reflect.Ptr {
					return
				}
				for i := 0; i < rv.NumField(); i++ {
					field := rv.Field(i)
					typeField := rv.Type().Field(i)

					if typeField.Anonymous && field.Type() == createModelType {
						field.Set(reflect.New(createModelType))
						_ = target.SetupCreateModel(cm)
						return
					}
				}
			}

			rv := reflect.ValueOf(v)
			switch rv.Kind() {
			case reflect.Slice, reflect.Array:
				for i := 0; i < rv.Len(); i++ {
					ele := rv.Index(i)
					setupCreateModel(ele.Interface())
				}
			default:
				setupCreateModel(v)
			}
		}
	})
}

*/

// Prepare UpdateModel Hook that will run when [Query] call UPDATED related methods, see [AddUpdateHooks].
//
// This hook will attempt to extract trace_id and updated_by field from trace ([miso.Rail]) and and update these values to database.
//
// Return whether the hook should run for current table, e.g., some tables may not have the trace_id and updated_by fields.
//
// By default, hooks are executed for all tables unless optionalFn is provided.
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
						q.Set("updated_by", UpdatedByExtractor(r))
					}
				}
			}
		}
	})
	return nil
}
