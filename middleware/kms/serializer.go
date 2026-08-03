package kms

import (
	"context"
	"reflect"
	"strings"

	"github.com/curtisnewbie/miso/errs"
	"gorm.io/gorm/schema"
)

func init() {
	schema.RegisterSerializer("kms", KMSSerializer{})
}

const kmsPrefix = "{kms}"

// KMSSerializer is a GORM serializer that encrypts string fields using KMS envelope encryption on write
// and decrypts on read. Encrypted values are stored with a "{kms}" prefix to distinguish from plaintext.
// Use with gorm tag: `gorm:"serializer:kms"`.
//
// Recommended column type: VARCHAR(1024).
type KMSSerializer struct{}

// Scan decrypts the database value into the struct field.
// Values without the "{kms}" prefix are treated as plaintext and returned as-is.
func (KMSSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	if dbValue == nil {
		return nil
	}

	var raw string
	switch v := dbValue.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return errs.NewErrf("kms serializer: unsupported db value type: %T", dbValue)
	}

	if raw == "" {
		return nil
	}

	// No prefix → plaintext, return as-is
	if !strings.HasPrefix(raw, kmsPrefix) {
		field.ReflectValueOf(ctx, dst).SetString(raw)
		return nil
	}

	plaintext, err := module().decrypt(raw[len(kmsPrefix):])
	if err != nil {
		return errs.NewErrf("kms serializer: decrypt failed: %v", err)
	}

	field.ReflectValueOf(ctx, dst).SetString(plaintext)
	return nil
}

// Value encrypts the struct field value for database storage.
// The encrypted value is stored with a "{kms}" prefix.
func (KMSSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	s, ok := fieldValue.(string)
	if !ok {
		return nil, errs.NewErrf("kms serializer: unsupported field type: %T, expected string", fieldValue)
	}
	if s == "" {
		return "", nil
	}
	encrypted, err := module().encrypt([]byte(s))
	if err != nil {
		return nil, errs.NewErrf("kms serializer: encrypt failed: %v", err)
	}
	return kmsPrefix + encrypted, nil
}
