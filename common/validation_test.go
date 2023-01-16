package common

import "testing"

type ValidatedDummy struct {
	Name       string   `json:"name" validation:"notEmpty"`
	StrNum     string   `validation:"positive"`
	PosNum     int      `validation:"positive"`
	PosZeroNum int      `validation:"positiveOrZero"`
	NegZeroNum int      `validation:"negativeOrZero"`
	NegNum     int      `validation:"negative"`
	Friends    []string `validation:"notEmpty"`
	secret     string   `validation:"notEmpty"`
}

func TestValidate(t *testing.T) {
	v := ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, NegNum: -1, Friends: []string{"nobody"}}
	e := Validate(v)
	if e != nil {
		t.Fatal(e)
	}

	v = ValidatedDummy{Name: "  ", StrNum: "1", PosNum: 1, NegNum: -1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}

	v = ValidatedDummy{Name: "", StrNum: "1", PosNum: 1, NegNum: -1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("Name's validation should fail, %v", v.Name)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "", PosNum: 1, NegNum: -1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("StrNum 's validation should fail, %v", v.StrNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "-1", PosNum: 1, NegNum: -1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("StrNum 's validation should fail, %v", v.StrNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 0, NegNum: -1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("PosNum 's validation should fail, %v", v.PosNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: -1, NegNum: -1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("PosNum 's validation should fail, %v", v.PosNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, NegNum: 1, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("NegNum 's validation should fail, %v", v.NegNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, NegNum: 0, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("NegNum 's validation should fail, %v", v.NegNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, NegNum: -1}
	if e = Validate(v); e == nil {
		t.Fatalf("Friends 's validation should fail, %v", v.Friends)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, PosZeroNum: -1, NegNum: 0, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("PosZeroNum 's validation should fail, %v", v.PosZeroNum)
	}

	v = ValidatedDummy{Name: "abc", StrNum: "1", PosNum: 1, PosZeroNum: 0, NegZeroNum: 1, NegNum: 0, Friends: []string{"nobody"}}
	if e = Validate(v); e == nil {
		t.Fatalf("NegZeroNum 's validation should fail, %v", v.NegZeroNum)
	}
}
