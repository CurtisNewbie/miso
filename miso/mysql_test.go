package miso

import "testing"

func TestNewConn(t *testing.T) {
	c, en := NewMySQLConn("root", "", "", "localhost", "3306", "")
	if en != nil {
		t.Fatal(en)
	}
	if c == nil {
		t.Fatal(c)
	}
}
