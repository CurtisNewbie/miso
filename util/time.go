package util

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/curtisnewbie/miso/util/hash"
	"github.com/curtisnewbie/miso/util/strutil"
)

const (
	unixSecPersudoMax = 9999999999 // 2286-11-21, should be enough :D

	ClassicDateTimeLocaleFormat = "2006/01/02 15:04:05 (MST)"
	ClassicDateTimeFormat       = "2006/01/02 15:04:05"
	StdDateTimeFormat           = "2006-01-02 15:04:05"
	StdDateTimeMilliFormat      = "2006-01-02 15:04:05.000"
	StdDateTimeLocaleFormat     = "2006-01-02 15:04:05 (MST)"
	SQLDateTimeFormat           = "2006-01-02 15:04:05.999999"
	SQLDateTimeFormatWithT      = "2006-01-02T15:04:05.999999"
	SQLDateFormat               = "2006-01-02"
)

var (
	etimeMarshalFormat = ""
	ToETime            = WrapTime
)

type ETime = Time

// Enhanced wrapper of time.Time.
//
// This type implements sql.Scanner and driver.Valuer, and thus can be safely used in GORM just like time.Time. It also
// implements json/encoding Marshaler and Unmarshaler to support json marshalling (in forms of epoch milliseconds 'by default').
//
// In previous releases, Time was a type alias to time.Time. Since v0.1.2, Time embeds time.Time to access all of it's methods.
//
// To cast from time.Time to Time, use [WrapTime] method. To cast from Time to time.Time, use [Time.Unwrap] method.
type Time struct {
	time.Time
}

func Now() Time {
	return ToETime(time.Now())
}

func NowUTC() Time {
	return ToETime(time.Now().UTC())
}

func NowPtr() *Time {
	t := Now()
	return &t
}

func NowUTCPtr() *Time {
	t := NowUTC()
	return &t
}

func WrapTime(t time.Time) Time {
	return Time{t}
}

func (t Time) GoString() string {
	return t.String()
}

// At 23:59:59.999999.
func (t Time) EndOfDay() Time {
	yyyy, mm, dd := t.Date()
	tt := time.Date(yyyy, mm, dd, 23, 59, 59, 999_999000, t.Location())
	return Time{tt}
}

// At 00:00:00.000000.
func (t Time) StartOfDay() Time {
	yyyy, mm, dd := t.Date()
	tt := time.Date(yyyy, mm, dd, 0, 0, 0, 0, t.Location())
	return Time{tt}
}

func (t Time) LastWeekday(w time.Weekday) Time {
	wkd := t.Weekday()
	diff := 0
	if wkd < w {
		diff = 7 - int(w-wkd)
	} else if wkd == w {
		diff = 7
	} else {
		diff = int(wkd - w)
	}
	return t.AddDate(0, 0, -diff)
}

func (t Time) NextWeekday(w time.Weekday) Time {
	wkd := t.Weekday()
	diff := 0
	if wkd < w {
		diff = int(w - wkd)
	} else if wkd == w {
		diff = 7
	} else {
		diff = 7 - int(wkd-w)
	}
	return t.AddDate(0, 0, diff)
}

// Deprecated: change to [Time.Unwrap].
func (t Time) ToTime() time.Time {
	return t.Time
}

func (t Time) Unwrap() time.Time {
	return t.Time
}

func (t Time) Add(d time.Duration) Time {
	t.Time = t.Time.Add(d)
	return t
}

func (t Time) Sub(u Time) time.Duration {
	return t.Time.Sub(u.Time)
}

func (t Time) AddDate(years int, months int, days int) Time {
	t.Time = t.Time.AddDate(years, months, days)
	return t
}

func (t Time) After(u Time) bool {
	return t.Time.After(u.Time)
}

func (t Time) Before(u Time) bool {
	return t.Time.Before(u.Time)
}

func (t Time) In(z *time.Location) Time {
	return ToETime(t.Unwrap().In(z))
}

func (t Time) InZone(diffInHours int) Time {
	if diffInHours == 0 {
		return t.In(time.UTC)
	}
	return t.In(time.FixedZone("", diffInHours*60*60))
}

func (t Time) FormatDate() string {
	return t.Unwrap().Format(time.DateOnly)
}

func (t Time) FormatClassic() string {
	return t.Unwrap().Format(ClassicDateTimeFormat)
}

func (t Time) FormatClassicLocale() string {
	return t.Unwrap().Format(ClassicDateTimeLocaleFormat)
}

func (t Time) FormatStd() string {
	return t.Unwrap().Format(StdDateTimeFormat)
}

func (t Time) FormatStdMilli() string {
	return t.Unwrap().Format(StdDateTimeMilliFormat)
}

func (t Time) FormatStdLocale() string {
	return t.Unwrap().Format(StdDateTimeLocaleFormat)
}

// Implements driver.Valuer in database/sql.
func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	// some db (e.g., Aliyun ADB) only supports .999999, we have to manully trim the precision
	return t.Format(SQLDateTimeFormat), nil
}

func (t Time) String() string {
	return t.Unwrap().Format("2006-01-02 15:04:05.999999 (MST)")
}

// Implements encoding/json Marshaler
func (t Time) MarshalJSON() ([]byte, error) {
	var v string
	if etimeMarshalFormat != "" {
		v = strutil.QuoteStr(t.Unwrap().Format(etimeMarshalFormat)) // other format configured
	} else {
		v = fmt.Sprintf("%d", t.UnixMilli()) // epoch milli by default
	}
	return UnsafeStr2Byt(v), nil
}

// Implements encoding/json Unmarshaler.
func (t *Time) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "" {
		return nil
	}
	millisec, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		s = strutil.UnquoteStr(s)
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
	SQLDateTimeFormatWithT,
	time.RFC3339Nano,
}

func AddETimeParseFormat(fmt ...string) {
	m := hash.NewSet[string](jsonParseTimeFormats...)
	m.AddAll(fmt)
	jsonParseTimeFormats = m.CopyKeys()
}

// Implements sql.Scanner in database/sql.
func (et *ETime) Scan(value interface{}) error {
	return et.ScanLoc(value, time.Local)
}

func (et *ETime) ScanLoc(value interface{}, loc *time.Location) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*et = ToETime(v.In(loc))
	case []byte:
		sv := string(v)
		var t time.Time
		t, err := FuzzParseTimeLoc(jsonParseTimeFormats, sv, loc)
		if err != nil {
			return err
		}
		*et = ToETime(t)
	case string:
		var t time.Time
		t, err := FuzzParseTimeLoc(jsonParseTimeFormats, v, loc)
		if err != nil {
			return err
		}
		*et = ToETime(t)
	case *string:
		var t time.Time
		t, err := FuzzParseTimeLoc(jsonParseTimeFormats, *v, loc)
		if err != nil {
			return err
		}
		*et = ToETime(t)
	case int64, int, uint, uint64, int32, uint32, int16, uint16, *int64, *int, *uint, *uint64, *int32, *uint32, *int16, *uint16:
		val := reflect.Indirect(reflect.ValueOf(v)).Int()
		if val > unixSecPersudoMax {
			*et = ToETime(time.UnixMilli(val).In(loc)) // in milli-sec
		} else {
			*et = ToETime(time.Unix(val, 0).In(loc)) // in sec
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

func ParseETime(v any) (ETime, error) {
	var t ETime
	return t, t.Scan(v)
}

func MayParseETime(v any) ETime {
	var t ETime
	t.Scan(v)
	return t
}

func ParseETimeLoc(v any, loc *time.Location) (ETime, error) {
	var t ETime
	return t, t.ScanLoc(v, loc)
}

func MayParseETimeLoc(v any, loc *time.Location) ETime {
	var t ETime
	t.ScanLoc(v, loc)
	return t
}
