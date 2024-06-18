package money

import (
	inf "gopkg.in/inf.v0"
)

// Create new signed arbitrary-precision decimal with appropriate scale for the currency and HalfEven rounding mode.
func NewInfDec(s string, currency string) (*inf.Dec, error) {
	sc, err := Scale(currency)
	if err != nil {
		return nil, err
	}
	d := new(inf.Dec)
	d.SetString(s)
	d.Round(d, inf.Scale(sc), inf.RoundHalfEven)
	return d, nil
}
