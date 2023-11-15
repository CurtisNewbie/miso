package miso

import "testing"

func TestFileExists(t *testing.T) {
	ok, e := FileExists("file.go")
	if e != nil {
		t.Logf("e != nil, %v", e)
		t.FailNow()
	}
	if !ok {
		t.FailNow()
	}

	ok, e = FileExists("file_not_found")
	if e != nil {
		t.Logf("e != nil, %v", e)
		t.FailNow()
	}
	if ok {
		t.FailNow()
	}
}
