package money

import (
	"golang.org/x/text/currency"
)

// Get currency unit.
func Unit(s string) (currency.Unit, error) {
	return currency.ParseISO(s)
}

// Get scale of currency unit.
func UnitScale(u currency.Unit) int {
	n, _ := currency.Cash.Rounding(u)
	return n
}

// Return scale of currency.
func Scale(s string) (int, error) {
	c, err := Unit(s)
	if err != nil {
		return 0, err
	}
	return UnitScale(c), nil
}
