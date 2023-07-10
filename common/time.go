package common

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
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
)

/* same as time.Time but will be serialized using format '02/01/2006 15:04' */
type WTime time.Time

/* same as time.Time but will be serialized/deserialized using format '2006-01-02 15:04:05' */
type TTime time.Time

/*
EpochTime, same as time.Time but will be serialized/deserialized as epoch milliseconds

This type can be safely used in GORM just like time.Time
*/
type ETime time.Time

var (
	/*
		yyyy : 2006
		MM : 01
		dd : 02
		HH : 15
		mm : 04
		ss : 05

		yyyy-MM-dd HH:mm:ss
		2006-01-02 15:04:05
	*/
	formatDict map[string]string

	// format for WTime
	wTimeFormat string
	// format for TTime
	tTimeFormat string

	sqlTimeFormat string
)

func init() {
	/*
		Why they format time like this, it's frustrating :D

		https://stackoverflow.com/questions/20234104/how-to-format-current-time-using-a-yyyymmddhhmmss-format
	*/
	formatDict = map[string]string{}
	formatDict["yyyy"] = F_E_YEAR
	formatDict["MM"] = F_E_MONTH
	formatDict["dd"] = F_E_DAY
	formatDict["HH"] = F_E_HOUR
	formatDict["mm"] = F_E_MIN
	formatDict["ss"] = F_E_SEC

	wTimeFormat = TranslateFormat("dd/MM/yyyy HH:mm")
	tTimeFormat = TranslateFormat("yyyy-MM-dd HH:mm:ss")
	sqlTimeFormat = TranslateFormat("yyyy/MM/dd HH:mm:ss")
}

func Now() ETime {
	return ETime(time.Now())
}

// Translate yyyy-MM-dd HH:mm:ss style format of time
func TranslateFormat(format string) string {
	for k := range formatDict {
		format = strings.ReplaceAll(format, k, formatDict[k])
	}
	return format
}

func (s WTime) MarshalJSON() ([]byte, error) {
	t := time.Time(s)
	return []byte(t.Format(`"` + wTimeFormat + `"`)), nil
}

func (t TTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	// yyyy/mm/dd hh:mm:ss
	return []byte(tt.Format(`"` + tTimeFormat + `"`)), nil
}

func (t *TTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	pt, err := time.ParseInLocation(tTimeFormat, s, time.Local)
	if err != nil {
		return err
	}
	*t = TTime(pt)
	return nil
}

func (t TTime) String() string {
	return time.Time(t).String()
}

func (t WTime) String() string {
	return time.Time(t).String()
}

// pretty print time
func TimePrettyPrint(t *time.Time) string {
	return fmt.Sprintf("%s (%s)", t.Format(`"`+tTimeFormat+`"`), t.Location())
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

func (t *ETime) String() string {
	return time.Time(*t).String()
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
