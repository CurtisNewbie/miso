package miso

import "testing"

func TestParseBearer(t *testing.T) {
	_, ok := ParseBearer("")
	if ok {
		t.Fatal("should not parse bearer")
	}

	s, ok := ParseBearer("bearer abc")
	if !ok {
		t.Fatal("should parse bearer")
	}
	if s != "abc" {
		t.Fatalf("s should be abc")
	}

	s, ok = ParseBearer("bearer  abc")
	if !ok {
		t.Fatal("should parse bearer")
	}
	if s != "abc" {
		t.Fatalf("s should be abc")
	}

	s, ok = ParseBearer("Bearer  abc")
	if !ok {
		t.Fatal("should parse bearer")
	}
	if s != "abc" {
		t.Fatalf("s should be abc")
	}

	_, ok = ParseBearer("Bearer  ")
	if ok {
		t.Fatal("should not parse bearer")
	}
}
