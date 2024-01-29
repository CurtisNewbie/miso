package miso

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

const (
	unixSecPersudoMax = 9999999999 // 2286-11-21, should be enough :D

	sqlTimeFormat = "2006/01/02 15:04:05"
)

/*
EpochTime, same as time.Time but will be serialized/deserialized as epoch milliseconds

This type can be safely used in GORM just like time.Time
*/
type ETime time.Time

func Now() ETime {
	return ETime(time.Now())
}

func (t ETime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	return []byte(fmt.Sprintf("%d", tt.UnixMilli())), nil
}

func (t ETime) String() string {
	return time.Time(t).String()
}

func (t ETime) ToTime() time.Time {
	return time.Time(t)
}

func (t ETime) UnixMilli() int64 {
	return t.ToTime().UnixMilli()
}

func (t ETime) FormatClassic() string {
	return t.ToTime().Format("2006/01/02 15:04:05")
}

func (t ETime) FormatClassicLocale() string {
	return t.ToTime().Format("2006/01/02 15:04:05 (MST)")
}

// Implements driver.Valuer in database/sql.
func (et ETime) Value() (driver.Value, error) {
	t := time.Time(et)
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(sqlTimeFormat), nil
}

// implements decorder.Unmarshaler in encoding/json.
func (t *ETime) UnmarshalJSON(b []byte) error {
	millisec, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}

	pt := time.UnixMilli(millisec)
	*t = ETime(pt)
	return nil
}

// Implements sql.Scanner in database/sql.
func (et *ETime) Scan(value interface{}) error {
	switch v := value.(type) {
	case time.Time:
		*et = ETime(v)
	case []byte:
		var t time.Time
		t, err := time.Parse(sqlTimeFormat, string(v))
		if err != nil {
			return err
		}
		*et = ETime(t)
	case string:
		var t time.Time
		t, err := time.Parse(sqlTimeFormat, v)
		if err != nil {
			return err
		}
		*et = ETime(t)
	case int64, int, uint, uint64, int32, uint32, int16, uint16, *int64, *int, *uint, *uint64, *int32, *uint32, *int16, *uint16:
		val := reflect.Indirect(reflect.ValueOf(v)).Int()
		if val > unixSecPersudoMax {
			*et = ETime(time.UnixMilli(val)) // in milli-sec
		} else {
			*et = ETime(time.Unix(val, 0)) // in sec
		}
	default:
		err := fmt.Errorf("invalid field type '%v' for ETime, unable to convert, %#v", reflect.TypeOf(value), v)
		return err
	}
	return nil
}
