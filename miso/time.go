package miso

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

const (
	F_E_YEAR  = "2006"
	F_E_MONTH = "01"
	F_E_DAY   = "02"
	F_E_HOUR  = "15"
	F_E_MIN   = "04"
	F_E_SEC   = "05"

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

func (t *ETime) UnmarshalJSON(b []byte) error {
	millisec, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return err
	}

	pt := time.UnixMilli(millisec)
	*t = ETime(pt)
	return nil
}

func (t ETime) String() string {
	return time.Time(t).String()
}

func (t *ETime) ToTime() time.Time {
	return time.Time(*t)
}

func (t *ETime) UnixMilli() int64 {
	return t.ToTime().UnixMilli()
}

func (t *ETime) FormatClassic() string {
	return t.ToTime().Format("2006/01/02 15:04:05")
}

// database driver -> ETime
func (et *ETime) Scan(value interface{}) (err error) {
	switch v := value.(type) {
	case time.Time:
		*et = ETime(v)
	case []byte:
		var t time.Time
		t, err = time.Parse(sqlTimeFormat, string(v))
		*et = ETime(t)
	case int64, int, uint, uint64, int32, uint32, int16, uint16, *int64, *int, *uint, *uint64, *int32, *uint32, *int16, *uint16:
		val := reflect.Indirect(reflect.ValueOf(v)).Int()
		if val > unixSecPersudoMax {
			*et = ETime(time.UnixMilli(val)) // in milli-sec
		} else {
			*et = ETime(time.Unix(val, 0)) // in sec
		}
	default:
		err = fmt.Errorf("invalid field type '%v' for ETime, unable to convert, %#v", reflect.TypeOf(value), v)
	}
	return
}

// ETime -> database driver
func (et ETime) Value() (driver.Value, error) {
	t := time.Time(et)
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(sqlTimeFormat), nil
}
