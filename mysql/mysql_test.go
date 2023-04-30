package mysql

import "testing"

func TestNewConn(t *testing.T) {
	c, en := NewConn("root", "", "", "localhost", "3306", "")
	if en != nil {
		t.Fatal(en)
	}
	if c == nil {
		t.Fatal(c)
	}
}
