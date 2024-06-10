package util

import (
	"bytes"
	"os"
	"testing"
)

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

func TestMkdirParentAll(t *testing.T) {
	f := "test/abc/yo"
	p := "test"
	err := MkdirParentAll(f)
	if err != nil {
		t.Fatal(err)
	}
	os.RemoveAll(p)
}

func TestSaveTmpFile(t *testing.T) {
	buf := bytes.NewReader([]byte("oh"))
	p, err := SaveTmpFile("/tmp", buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(p)
}
