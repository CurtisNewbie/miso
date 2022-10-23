package test

import "testing"

func TestEqual(t *testing.T, left string, right string) bool {
	if left != right {
		t.Errorf("'%s' != '%s'", left, right)
		return false
	}
	return true
}
