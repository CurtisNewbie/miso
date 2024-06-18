package money

import (
	inf "gopkg.in/inf.v0"
)

// Create new signed arbitrary-precision decimal with appropriate scale for the currency and HalfEven rounding mode.
func NewInfDec(amt string, currency string) (*inf.Dec, error) {
	sc, err := Scale(currency)
	if err != nil {
		return nil, err
	}
	d := new(inf.Dec)
	d.SetString(amt)
	d.Round(d, inf.Scale(sc), inf.RoundHalfEven)
	return d, nil
}

// Format amount to appropriate rounding and scale for the currency.
func FormatAmt(amt string, currency string) string {
	dec, err := NewInfDec(amt, currency)
	if err != nil {
		return amt
	}
	return dec.String()
}
