package miso

import (
	"testing"
)

type ValidatedDummy struct {
	Name       string        `json:"name" valid:"maxLen  : 10 , notEmpty"`
	StrNum     string        `valid:"positive"`
	PosNum     int           `valid:"positive"`
	PosZeroNum int           `valid:"positiveOrZero"`
	NegZeroNum int           `valid:"negativeOrZero"`
	NegNum     int           `valid:"negative"`
	Friends    []string      `valid:"notEmpty"`
	Type       string        `valid:"member:PUBLIC|PROTECTED"`
	secret     string        `valid:"notEmpty"` //lint:ignore U1000 for testing
	Another    AnotherDummy  `valid:"validated"`
	DummyPtr   *AnotherDummy `valid:"notNil,validated"`
}

type AnotherDummy struct {
	Favourite string `validation:"notEmpty"`
}

func validDummy() ValidatedDummy {
	return ValidatedDummy{Name: "abc",
		StrNum:   "1",
		PosNum:   1,
		NegNum:   -1,
		Friends:  []string{"nobody"},
		Another:  AnotherDummy{Favourite: "apple"},
		DummyPtr: &AnotherDummy{Favourite: "juice"},
		Type:     "PUBLIC",
	}
}

func TestValidate(t *testing.T) {
	v := validDummy()
	e := Validate(v)
	if e != nil {
		t.Fatal(e)
	}

	v = validDummy()
	v.Name = "    "
	e = Validate(v)
	if e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Name" || ve.Rule != "notEmpty" {
		t.Fatalf("Validation should fail because of Name/notEmpty")
	}

	v = validDummy()
	v.Name = ""
	e = Validate(v)
	if e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Name" || ve.Rule != "notEmpty" {
		t.Fatalf("Validation should fail because of Name/notEmpty")
	}

	v = validDummy()
	v.Name = "イタリアイタリアイタリア" // exceeds max len
	e = Validate(v)
	if e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}
	// t.Logf("%v", e)
	if ve, _ := e.(*ValidationError); ve.Field != "Name" || ve.Rule != "maxLen" {
		t.Fatalf("Validation should fail because of Name/maxLen")
	}

	v = validDummy()
	v.StrNum = ""
	e = Validate(v)
	if e == nil {
		t.Fatalf("StrNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "StrNum" || ve.Rule != "positive" {
		t.Fatalf("Validation should fail because of StrNum/positive")
	}

	v = validDummy()
	v.StrNum = "-1"
	e = Validate(v)
	if e == nil {
		t.Fatalf("StrNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "StrNum" || ve.Rule != "positive" {
		t.Fatalf("Validation should fail because of StrNum/positive")
	}

	v = validDummy()
	v.StrNum = "0"
	e = Validate(v)
	if e == nil {
		t.Fatalf("StrNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "StrNum" || ve.Rule != "positive" {
		t.Fatalf("Validation should fail because of StrNum/positive")
	}

	v = validDummy()
	v.PosNum = 0
	e = Validate(v)
	if e == nil {
		t.Fatalf("PosNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "PosNum" || ve.Rule != "positive" {
		t.Fatalf("Validation should fail because of PosNum/positive")
	}

	v = validDummy()
	v.PosNum = -1
	e = Validate(v)
	if e == nil {
		t.Fatalf("PosNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "PosNum" || ve.Rule != "positive" {
		t.Fatalf("Validation should fail because of PosNum/positive")
	}

	v = validDummy()
	v.NegNum = 1
	e = Validate(v)
	if e == nil {
		t.Fatalf("NegNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "NegNum" || ve.Rule != "negative" {
		t.Fatalf("Validation should fail because of NegNum/negative")
	}

	v = validDummy()
	v.NegNum = 0
	e = Validate(v)
	if e == nil {
		t.Fatalf("NegNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "NegNum" || ve.Rule != "negative" {
		t.Fatalf("Validation should fail because of NegNum/negative")
	}

	v = validDummy()
	v.Friends = []string{}
	e = Validate(v)
	if e == nil {
		t.Fatalf("Friends's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Friends" || ve.Rule != "notEmpty" {
		t.Fatalf("Validation should fail because of Friends/notEmpty")
	}

	v = validDummy()
	v.PosZeroNum = -1
	e = Validate(v)
	if e == nil {
		t.Fatalf("PosZeroNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "PosZeroNum" || ve.Rule != "positiveOrZero" {
		t.Fatalf("Validation should fail because of PosZeroNum/positiveOrZero")
	}

	v = validDummy()
	v.NegZeroNum = 1
	e = Validate(v)
	if e == nil {
		t.Fatalf("NegZeroNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "NegZeroNum" || ve.Rule != "negativeOrZero" {
		t.Fatalf("Validation should fail because of NegZeroNum/negativeOrZero")
	}

	v = validDummy()
	v.Another = AnotherDummy{}
	e = Validate(v)
	if e == nil {
		t.Fatalf("Another.Favourite 's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Another.Favourite" || ve.Rule != "notEmpty" {
		t.Fatalf("Validation should fail because of Another.Favourite/notEmpty")
	}

	v = validDummy()
	v.DummyPtr = nil
	e = Validate(v)
	if e == nil {
		t.Fatalf("DummyPtr 's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "DummyPtr" || ve.Rule != "notNil" {
		t.Fatalf("Validation should fail because of DummyPtr/notNil")
	}

	v = validDummy()
	v.DummyPtr = &AnotherDummy{}
	e = Validate(v)
	if e == nil {
		t.Fatalf("DummyPtr.Favourite 's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "DummyPtr.Favourite" || ve.Rule != "notEmpty" {
		t.Fatalf("Validation should fail because of DummyPtr.Favourite/notEmpty")
	}

	v = validDummy()
	v.Type = "wrong type"
	e = Validate(v)
	if e == nil {
		t.Fatalf("Type validation should fail, %v", v.Type)
	} else {
		t.Log(e)
	}

}

func TestParsedValidRules(t *testing.T) {
	r := "maxLen  : 10 , notEmpty"
	rules := parseValidRules(r)
	if len(rules) != 2 {
		t.Fatal()
	}
	t.Logf("%#v", rules)

	r = "member:PUBLIC|PROTECTED"
	rules = parseValidRules(r)
	if len(rules) != 1 {
		t.Fatal()
	}
	t.Logf("%#v", rules)

	r = "notNil,validated"
	rules = parseValidRules(r)
	if len(rules) != 2 {
		t.Fatal()
	}
	t.Logf("%#v", rules)
}
