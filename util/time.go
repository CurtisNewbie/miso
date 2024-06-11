package util

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

const (
	unixSecPersudoMax = 9999999999 // 2286-11-21, should be enough :D

	SQLDateTimeFormat = "2006/01/02 15:04:05"
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
	return UnsafeStr2Byt(fmt.Sprintf("%d", tt.UnixMilli())), nil
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

func (t ETime) FormatDate() string {
	return t.ToTime().Format(time.DateOnly)
}

func (t ETime) FormatClassic() string {
	return t.ToTime().Format(SQLDateTimeFormat)
}

func (t ETime) FormatClassicLocale() string {
	return t.ToTime().Format("2006/01/02 15:04:05 (MST)")
}

func (t ETime) Add(d time.Duration) ETime {
	return ETime(t.ToTime().Add(d))
}

// Implements driver.Valuer in database/sql.
func (et ETime) Value() (driver.Value, error) {
	t := time.Time(et)
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(SQLDateTimeFormat), nil
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
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*et = ETime(v)
	case []byte:
		var t time.Time
		t, err := time.Parse(SQLDateTimeFormat, string(v))
		if err != nil {
			return err
		}
		*et = ETime(t)
	case string:
		var t time.Time
		t, err := time.Parse(SQLDateTimeFormat, v)
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

func FuzzParseTime(formats []string, value string) (time.Time, error) {
	return FuzzParseTimeLoc(formats, value, time.UTC)
}

func FuzzParseTimeLoc(formats []string, value string, loc *time.Location) (time.Time, error) {
	if len(formats) < 1 {
		return time.Time{}, errors.New("formats is empty")
	}
	if loc == nil {
		loc = time.UTC
	}

	var t time.Time
	var err error
	for _, f := range formats {
		t, err = time.ParseInLocation(f, value, loc)
		if err == nil {
			return t, nil
		}
	}
	return t, fmt.Errorf("failed to parse time '%s'", value)
}

// Parse classic datetime format using patterns: "2006-01-02 15:04:05", "2006/01/02 15:04:05".
func ParseClassicDateTime(val string, loc *time.Location) (time.Time, error) {
	return FuzzParseTimeLoc([]string{
		time.DateTime,
		SQLDateTimeFormat,
	}, val, loc)
}
