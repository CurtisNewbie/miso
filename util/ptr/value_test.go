package ptr

import "testing"

func TestPtrVal(t *testing.T) {
	s := "123"
	var p *string = &s
	r := StrVal(p)
	if r != "123" {
		t.Fatal("should be 123")
	}
	r = StrVal(nil)
	if r != "" {
		t.Fatal("should be emtpy")
	}
	p = nil
	r = StrVal(p)
	if r != "" {
		t.Fatal("should be emtpy")
	}
}
