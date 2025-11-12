package redis

import (
	"fmt"

	"github.com/curtisnewbie/miso/encoding/json"
	"github.com/curtisnewbie/miso/util/strutil"
)

// Value serializer / deserializer.
type Serializer interface {
	Serialize(t any) (string, error)
	Deserialize(ptr any, v string) error
}

type JsonSerializer struct {
}

func (j JsonSerializer) Serialize(t any) (string, error) {
	if v, ok := t.(string); ok {
		return v, nil
	}

	b, err := json.WriteJson(t)
	if err != nil {
		return "", fmt.Errorf("unable to marshal value to string, %v", err)
	}
	return strutil.UnsafeByt2Str(b), nil
}

func (j JsonSerializer) Deserialize(ptr any, v string) error {
	if p, ok := ptr.(*string); ok {
		*p = v
		return nil
	}

	err := json.ParseJson(strutil.UnsafeStr2Byt(v), ptr)
	if err != nil {
		return fmt.Errorf("unable to unmarshal from string, %v", err)
	}
	return err
}
