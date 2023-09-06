package miso

import "testing"

func TestPadNum(t *testing.T) {
	var res string
	var expected string

	res = PadNum(11, 4)
	expected = "0011"
	if res != expected {
		t.Fatalf("actual: %v, expected: %v", res, expected)
	}

	res = PadNum(0, 4)
	expected = "0000"
	if res != expected {
		t.Fatalf("actual: %v, expected: %v", res, expected)
	}

	res = PadNum(12345, 4)
	expected = "12345"
	if res != expected {
		t.Fatalf("actual: %v, expected: %v", res, expected)
	}
}

func TestMaxLenStr(t *testing.T) {
	s := "123456"
	TestEqual(t, MaxLenStr(s, 3), "123")

	s = "12"
	TestEqual(t, MaxLenStr(s, 3), "12")
}
