package money

import (
	"reflect"
	"testing"
)

func TestAdd(t *testing.T) {
	unit, _ := Unit("CNY")
	d1 := UnitDec("1.1", unit)
	d2 := UnitDec("1.12", unit)
	v := Add(d1, d2)
	t.Logf("d1: %v, d2: %v, v: %v", d1, d2, v)

	sv := d1.Add(d1, d2)
	t.Logf("d1: %v, d2: %v, sv: %v", d1, d2, sv)
}

func TestSub(t *testing.T) {
	unit, _ := Unit("CNY")
	d1 := UnitDec("1.3", unit)
	d2 := UnitDec("1.4", unit)
	v := Sub(d1, d2)
	t.Logf("d1: %v, d2: %v, v: %v", d1, d2, v)
}

func TestMul(t *testing.T) {
	unit, _ := Unit("CNY")
	d1 := UnitDec("1.3", unit)
	d2 := UnitDec("1.4", unit)
	v := Mul(d1, d2)
	t.Logf("d1: %v, d2: %v, v: %v", d1, d2, v)
}

func TestDiv(t *testing.T) {
	unit, _ := Unit("CNY")
	d1 := UnitDec("1.3", unit)
	d2 := UnitDec("1.4", unit)
	v := Div(d1, d2, 4)
	t.Logf("d1: %v, d2: %v, v: %v", d1, d2, v)
}

func TestAmt(t *testing.T) {
	amt := new(Amt)
	t.Logf("1. amt: %v", amt)

	amt.SetString("1.64213")
	t.Logf("2. amt: %v", amt)

	r := amt.Add(NewAmt("1.2"))
	t.Logf("3. amt: %v, r: %v", amt, r)

	r = amt.Div(NewAmt("1.2"), 4)
	t.Logf("4. amt: %v, r: %v", amt, r)

	r = amt.Mul(NewAmt("1.2"))
	t.Logf("5. amt: %v, r: %v", amt, r)

	r = amt.Round(2)
	t.Logf("6. amt: %v, r: %v", amt, r)

	r = Zero().Add(NewAmt("123"))
	t.Logf("7. r: %v", r)

	v, err := amt.Value()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("8. amt: %v, v: %v, type: %v", amt, v, reflect.TypeOf(v))

	vs, err := amt.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("9. amt: %v, vs: %v", amt, string(vs))

	err = amt.UnmarshalJSON([]byte("\"1.64214\""))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("10. amt: %v", amt)

	err = amt.UnmarshalJSON([]byte("1.64214"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("11. amt: %v", amt)

	err = amt.Scan("1.64213")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("12. amt: %v", amt)

	err = amt.Scan(uint(1))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("13. amt: %v", amt)

	err = amt.Scan("")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("14. amt: %v", amt)

	err = amt.Scan(1.64213)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("15. amt: %v", amt)

	err = amt.Scan(1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("16. amt: %v", amt)

	err = amt.Scan(-1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("17. amt: %v", amt)

	abs := amt.Abs()
	t.Logf("18. amt: %v, abs: %v", amt, abs)

	si := amt.Sign()
	t.Logf("19. amt: %v, si: %v", amt, si)
}
