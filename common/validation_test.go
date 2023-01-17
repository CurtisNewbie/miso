package common

import "testing"

type ValidatedDummy struct {
	Name       string        `json:"name" validation:"notEmpty"`
	StrNum     string        `validation:"positive"`
	PosNum     int           `validation:"positive"`
	PosZeroNum int           `validation:"positiveOrZero"`
	NegZeroNum int           `validation:"negativeOrZero"`
	NegNum     int           `validation:"negative"`
	Friends    []string      `validation:"notEmpty"`
	secret     string        `validation:"notEmpty"`
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
	if ve, _ := e.(*ValidationError); ve.Field != "Name" {
		t.Fatalf("Validation should fail because of Name")
	}

	v = validDummy()
	v.Name = ""
	e = Validate(v)
	if e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Name" {
		t.Fatalf("Validation should fail because of Name")
	}

	v = validDummy()
	v.StrNum = ""
	e = Validate(v)
	if e == nil {
		t.Fatalf("StrNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "StrNum" {
		t.Fatalf("Validation should fail because of StrNum")
	}

	v = validDummy()
	v.StrNum = "-1"
	e = Validate(v)
	if e == nil {
		t.Fatalf("StrNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "StrNum" {
		t.Fatalf("Validation should fail because of StrNum")
	}

	v = validDummy()
	v.StrNum = "0"
	e = Validate(v)
	if e == nil {
		t.Fatalf("StrNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "StrNum" {
		t.Fatalf("Validation should fail because of StrNum")
	}

	v = validDummy()
	v.PosNum = 0
	e = Validate(v)
	if e == nil {
		t.Fatalf("PosNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "PosNum" {
		t.Fatalf("Validation should fail because of PosNum")
	}

	v = validDummy()
	v.PosNum = -1
	e = Validate(v)
	if e == nil {
		t.Fatalf("PosNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "PosNum" {
		t.Fatalf("Validation should fail because of PosNum")
	}

	v = validDummy()
	v.NegNum = 1
	e = Validate(v)
	if e == nil {
		t.Fatalf("NegNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "NegNum" {
		t.Fatalf("Validation should fail because of NegNum")
	}

	v = validDummy()
	v.NegNum = 0
	e = Validate(v)
	if e == nil {
		t.Fatalf("NegNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "NegNum" {
		t.Fatalf("Validation should fail because of NegNum")
	}

	v = validDummy()
	v.Friends = []string{}
	e = Validate(v)
	if e == nil {
		t.Fatalf("Friends's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Friends" {
		t.Fatalf("Validation should fail because of Friends")
	}

	v = validDummy()
	v.PosZeroNum = -1
	e = Validate(v)
	if e == nil {
		t.Fatalf("PosZeroNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "PosZeroNum" {
		t.Fatalf("Validation should fail because of PosZeroNum")
	}

	v = validDummy()
	v.NegZeroNum = 1
	e = Validate(v)
	if e == nil {
		t.Fatalf("NegZeroNum's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "NegZeroNum" {
		t.Fatalf("Validation should fail because of NegZeroNum")
	}

	v = validDummy()
	v.Another = AnotherDummy{}
	e = Validate(v)
	if e == nil {
		t.Fatalf("Another.Favourite 's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "Another.Favourite" {
		t.Fatalf("Validation should fail because of Another.Favourite")
	}

	v = validDummy()
	v.DummyPtr = nil
	e = Validate(v)
	if e == nil {
		t.Fatalf("DummyPtr 's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "DummyPtr" {
		t.Fatalf("Validation should fail because of DummyPtr")
	}

	v = validDummy()
	v.DummyPtr = &AnotherDummy{}
	e = Validate(v)
	if e == nil {
		t.Fatalf("DummyPtr.Favourite 's validation should fail, %v", v.StrNum)
	}
	if ve, _ := e.(*ValidationError); ve.Field != "DummyPtr.Favourite" {
		t.Fatalf("Validation should fail because of DummyPtr.Favourite")
	}
}
