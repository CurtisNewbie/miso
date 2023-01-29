package common

import "testing"

type ValidatedDummy struct {
	Name       string        `json:"name" validation:"maxLen  : 10 , notEmpty"`
	StrNum     string        `validation:"positive"`
	PosNum     int           `validation:"positive"`
	PosZeroNum int           `validation:"positiveOrZero"`
	NegZeroNum int           `validation:"negativeOrZero"`
	NegNum     int           `validation:"negative"`
	Friends    []string      `validation:"notEmpty"`
	secret     string        `validation:"notEmpty"` //lint:ignore U1000 for testing
	Another    AnotherDummy  `validation:"validated"`
	DummyPtr   *AnotherDummy `validation:"notNil,validated"`
}

type AnotherDummy struct {
	Favourite string `validation:"notEmpty"`
}

func validDummy() ValidatedDummy {
	return ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, NegNum: -1, Friends: []string{"nobody"}, Another: AnotherDummy{Favourite: "apple"}, DummyPtr: &AnotherDummy{Favourite: "juice"}}
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
	v.Name = "aaaaaaaaaaaaaaaa" // exceeds max len
	e = Validate(v)
	if e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}
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
}
