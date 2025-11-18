package dbquery

import (
	"context"
	"fmt"
	"reflect"

	"github.com/curtisnewbie/miso/util/json"
	"gorm.io/gorm/schema"
)

func init() {
	// overwrite default json serializer
	schema.RegisterSerializer("json", MisoJSONSerializer{})
}

// MisoJSONSerializer json serializer
type MisoJSONSerializer struct {
}

// Scan implements serializer interface
func (MisoJSONSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
	fieldValue := reflect.New(field.FieldType)

	if dbValue != nil {
		var bytes []byte
		switch v := dbValue.(type) {
		case []byte:
			bytes = v
		case string:
			bytes = []byte(v)
		default:
			return fmt.Errorf("failed to unmarshal JSONB value: %#v", dbValue)
		}

		err = json.ParseJson(bytes, fieldValue.Interface())
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value implements serializer interface
func (MisoJSONSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	return json.SWriteJson(fieldValue)
}
