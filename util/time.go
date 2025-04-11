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

	ClassicDateTimeLocaleFormat = "2006/01/02 15:04:05 (MST)"
	ClassicDateTimeFormat       = "2006/01/02 15:04:05"
	StdDateTimeFormat           = "2006-01-02 15:04:05"
	StdDateTimeMilliFormat      = "2006-01-02 15:04:05.000"
	StdDateTimeLocaleFormat     = "2006-01-02 15:04:05 (MST)"
	SQLDateTimeFormat           = "2006-01-02 15:04:05.999999"
	SQLDateFormat               = "2006-01-02"
)

var (
	etimeMarshalFormat = ""
)

// ETime enhanced wrapper of time.Time.
//
// This type implements sql.Scanner and driver.Valuer, and thus can be safely used in GORM just like time.Time. It also
// implements json/encoding Marshaler and Unmarshaler to support json marshalling (in forms of epoch milliseconds 'by default').
//
// In previous releases, ETime was a type alias to time.Time. Since v0.1.2, ETime embeds time.Time to access all of it's methods.
//
// To cast from time.Time to ETime, use ToETime() method. To cast from ETime to time.Time, use ETime.ToTime() method.
type ETime struct {
	time.Time
}

func Now() ETime {
	return ToETime(time.Now())
}

func ToETime(t time.Time) ETime {
	return ETime{t}
}

func (t ETime) EndOfDay() ETime {
	yyyy, mm, dd := t.Date()
	tt := time.Date(yyyy, mm, dd, 23, 59, 59, 999_999999, t.Location())
	return ETime{tt}
}

func (t ETime) ToTime() time.Time {
	return t.Time
}

func (t ETime) Add(d time.Duration) ETime {
	t.Time = t.Time.Add(d)
	return t
}

func (t ETime) Sub(u ETime) time.Duration {
	return t.Time.Sub(u.Time)
}

func (t ETime) AddDate(years int, months int, days int) ETime {
	t.Time = t.Time.AddDate(years, months, days)
	return t
}

func (t ETime) After(u ETime) bool {
	return t.Time.After(u.Time)
}

func (t ETime) Before(u ETime) bool {
	return t.Time.Before(u.Time)
}

func (t ETime) FormatDate() string {
	return t.ToTime().Format(time.DateOnly)
}

func (t ETime) FormatClassic() string {
	return t.ToTime().Format(ClassicDateTimeFormat)
}

func (t ETime) FormatClassicLocale() string {
	return t.ToTime().Format(ClassicDateTimeLocaleFormat)
}

func (t ETime) FormatStd() string {
	return t.ToTime().Format(StdDateTimeFormat)
}

func (t ETime) FormatStdMilli() string {
	return t.ToTime().Format(StdDateTimeMilliFormat)
}

func (t ETime) FormatStdLocale() string {
	return t.ToTime().Format(StdDateTimeLocaleFormat)
}

// Implements driver.Valuer in database/sql.
func (t ETime) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Format(SQLDateTimeFormat), nil
}

func (t ETime) String() string {
	return t.FormatStdMilli()
}

// Implements encoding/json Marshaler
func (t ETime) MarshalJSON() ([]byte, error) {
	var v string
	if etimeMarshalFormat != "" {
		v = QuoteStr(t.ToTime().Format(etimeMarshalFormat)) // other format configured
	} else {
		v = fmt.Sprintf("%d", t.UnixMilli()) // epoch milli by default
	}
	return UnsafeStr2Byt(v), nil
}

// Implements encoding/json Unmarshaler.
func (t *ETime) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "" {
		return nil
	}
	millisec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		s = UnquoteStr(s)
		// try SQLDateTimeFormat
		if xer := t.Scan(s); xer != nil {
			return fmt.Errorf("failed to UnmarshalJSON, tried epoch milliseconds format %w, tried '2006-01-02 15:04:05.999999' format %w", err, xer)
		} else {
			return nil
		}
	}

	pt := time.UnixMilli(millisec)
	*t = ToETime(pt)
	return nil
}

var jsonParseTimeFormats = []string{
	SQLDateTimeFormat,
	SQLDateFormat,
}

// Implements sql.Scanner in database/sql.
func (et *ETime) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*et = ToETime(v)
	case []byte:
		sv := string(v)
		var t time.Time
		t, err := FuzzParseTimeLoc(jsonParseTimeFormats, sv, time.Local)
		if err != nil {
			return err
		}
		*et = ToETime(t)
	case string:
		var t time.Time
		t, err := FuzzParseTimeLoc(jsonParseTimeFormats, v, time.Local)
		if err != nil {
			return err
		}
		*et = ToETime(t)
	case *string:
		var t time.Time
		t, err := FuzzParseTimeLoc(jsonParseTimeFormats, *v, time.Local)
		if err != nil {
			return err
		}
		*et = ToETime(t)
	case int64, int, uint, uint64, int32, uint32, int16, uint16, *int64, *int, *uint, *uint64, *int32, *uint32, *int16, *uint16:
		val := reflect.Indirect(reflect.ValueOf(v)).Int()
		if val > unixSecPersudoMax {
			*et = ToETime(time.UnixMilli(val)) // in milli-sec
		} else {
			*et = ToETime(time.Unix(val, 0)) // in sec
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

var classicDateTimeFmt = []string{SQLDateTimeFormat, ClassicDateTimeFormat}

// Parse classic datetime format using patterns: "2006-01-02 15:04:05", "2006/01/02 15:04:05".
func ParseClassicDateTime(val string, loc *time.Location) (time.Time, error) {
	return FuzzParseTimeLoc(classicDateTimeFmt, val, loc)
}

func SetETimeMarshalFormat(fmt string) {
	etimeMarshalFormat = fmt
}
