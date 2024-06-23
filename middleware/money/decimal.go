package money

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
	"github.com/spf13/cast"
	"golang.org/x/text/currency"
	inf "gopkg.in/inf.v0"
)

func init() {
	miso.ApiDocTypeAlias["*money.Amt"] = "string"
	miso.ApiDocTypeAlias["money.Amt"] = "string"
}

// Create new signed arbitrary-precision decimal with appropriate scale for the currency and HalfEven rounding.
func UnitDec(amt string, unit currency.Unit) *inf.Dec {
	d := new(inf.Dec)
	d.SetString(amt)
	return RoundUnit(d, unit)
}

// Format amount to appropriate scale for the currency using HalfEven rounding mode.
func UnitFmt(amt string, currency string) string {
	unit, err := Unit(currency)
	if err != nil {
		return amt
	}
	return UnitDec(amt, unit).String()
}

// Round d to currency scale using HalfEven rounding mode.
//
// Value of d is unchanged.
func RoundUnit(d *inf.Dec, unit currency.Unit) *inf.Dec {
	return Round(d, UnitScale(unit))
}

// Round d to scale using HalfEven rounding mode.
//
// Value of d is unchanged.
func Round(d *inf.Dec, scale int) *inf.Dec {
	return new(inf.Dec).Round(d, inf.Scale(scale), inf.RoundHalfEven)
}

// Return d1 + d2.
//
// Values of d1 and d2 are unchanged.
func Add(d1 *inf.Dec, d2 *inf.Dec) *inf.Dec {
	return new(inf.Dec).Add(d1, d2)
}

// Return d1 - d2.
//
// Values of d1 and d2 are unchanged.
func Sub(d1 *inf.Dec, d2 *inf.Dec) *inf.Dec {
	return new(inf.Dec).Sub(d1, d2)
}

// Return d1 / d2 with HalfEven rounding.
//
// Values of d1 and d2 are unchanged.
func Div(d1 *inf.Dec, d2 *inf.Dec, scale int) *inf.Dec {
	return new(inf.Dec).QuoRound(d1, d2, inf.Scale(scale), inf.RoundHalfEven)
}

// Return d1 * d2.
//
// Values of d1 and d2 are unchanged.
func Mul(d1 *inf.Dec, d2 *inf.Dec) *inf.Dec {
	return new(inf.Dec).Mul(d1, d2)
}

// Amt represents arbitrary-precision decimal.
//
// Amt is a type alias of inf.Dec. Amt can be used with encoding/json and gorm.
//
// Different from inf.Dec, Amt always create new value for all math op (Add/Sub/Div/Mul)
// instead of setting the result back to the value itself.
//
// Amt always use HalfEven rounding mode.
type Amt inf.Dec

func (a *Amt) SetString(s string) error {
	d := inf.Dec(*a)
	_, b := d.SetString(s)
	if !b {
		return fmt.Errorf("invalid decimal number '%v'", s)
	}
	*a = Amt(d)
	return nil
}

func (a *Amt) String() string {
	v := inf.Dec(*a)
	return v.String()
}

func (a *Amt) Add(b *Amt) *Amt {
	aa := inf.Dec(*a)
	ba := inf.Dec(*b)
	v := Amt(*Add(&aa, &ba))
	return &v
}

func (a *Amt) Sub(b *Amt) *Amt {
	aa := inf.Dec(*a)
	ba := inf.Dec(*b)
	v := Amt(*Sub(&aa, &ba))
	return &v
}

func (a *Amt) Mul(b *Amt) *Amt {
	aa := inf.Dec(*a)
	ba := inf.Dec(*b)
	v := Amt(*Mul(&aa, &ba))
	return &v
}

func (a *Amt) Cmp(b *Amt) int {
	aa := inf.Dec(*a)
	ba := inf.Dec(*b)
	return aa.Cmp(&ba)
}

func (a *Amt) Div(b *Amt, scale int) *Amt {
	aa := inf.Dec(*a)
	ba := inf.Dec(*b)
	v := Amt(*Div(&aa, &ba, scale))
	return &v
}

func (a *Amt) Round(scale int) *Amt {
	aa := inf.Dec(*a)
	v := Amt(*Round(&aa, scale))
	return &v
}

func (a *Amt) RoundUnit(unit currency.Unit) *Amt {
	aa := inf.Dec(*a)
	v := Amt(*RoundUnit(&aa, unit))
	return &v
}

func (a *Amt) RoundCurrency(currency string) (*Amt, error) {
	u, err := Unit(currency)
	if err != nil {
		return a, err
	}
	return a.RoundUnit(u), nil
}

func (a *Amt) TryRoundCurrency(currency string) *Amt {
	cp, _ := a.RoundCurrency(currency)
	return cp
}

func (a *Amt) Abs() *Amt {
	aa := inf.Dec(*a)
	v := new(inf.Dec).Abs(&aa)
	cp := Amt(*v)
	return &cp
}

// Implements driver.Valuer in database/sql.
func (a Amt) Value() (driver.Value, error) {
	return a.String(), nil
}

// Implements encoding/json Marshaler
func (a Amt) MarshalJSON() ([]byte, error) {
	return util.UnsafeStr2Byt("\"" + a.String() + "\""), nil
}

// Implements encoding/json Unmarshaler.
func (t *Amt) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" || s == "\"\"" || len(s) < 2 {
		return nil
	}
	return t.SetString(s[1 : len(s)-1])
}

// Implements sql.Scanner in database/sql.
func (et *Amt) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		if v == "" {
			return et.SetString("0")
		}
		return et.SetString(v)
	case int64, int, uint, uint64, int32, uint32, int16, uint16, *int64, *int, *uint, *uint64, *int32, *uint32, *int16, *uint16:
		val := reflect.Indirect(reflect.ValueOf(v)).Int()
		return et.SetString(cast.ToString(val))
	case float32, float64, *float32, *float64:
		val := reflect.Indirect(reflect.ValueOf(v)).Float()
		return et.SetString(cast.ToString(val))
	case []uint8:
		s := string(v)
		if s == "" {
			return et.SetString("0")
		}
		return et.SetString(s)
	default:
		return fmt.Errorf("invalid field type '%v' for *Amt, unable to convert, %#v", reflect.TypeOf(value), v)
	}
}

func NewAmt(s string) *Amt {
	v := new(Amt)
	v.SetString(s)
	return v
}

func Zero() *Amt {
	return &Amt{}
}
