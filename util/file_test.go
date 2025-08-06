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

func TestFileHasSuffix(t *testing.T) {
	n := "abc.txt"
	ok := FileHasSuffix(n, "txt")
	if !ok {
		t.Fatal("should be ok")
	}
	v, ok := FileCutSuffix(n, "txt")
	if !ok {
		t.Fatal("should be ok")
	}
	if v != "abc" {
		t.Fatal(v)
	}

	v, ok = FileCutSuffix(n, "tx")
	if ok {
		t.Fatal("should not be ok")
	}
	if v != n {
		t.Fatal(v)
	}
}

func TestFileAddSuffix(t *testing.T) {
	n := "abc.txt"
	v := FileAddSuffix(n, "txt")
	if v != n {
		t.Fatal(v)
	}
	n = "abc"
	v = FileAddSuffix(n, "txt")
	if v != n+".txt" {
		t.Fatal(v)
	}
}

func TestWalkDir(t *testing.T) {
	f, err := WalkDir("../cmd/misoapi", "json", "go")
	if err != nil {
		t.Fatal(err)
	}
	for _, ff := range f {
		t.Logf("%v", ff.Path)
	}
}

func TestFileCutDotSuffix(t *testing.T) {
	v, ok := FileCutDotSuffix("abc")
	t.Logf("%v, %v", v, ok)

	v, ok = FileCutDotSuffix("abc.")
	t.Logf("%v, %v", v, ok)

	v, ok = FileCutDotSuffix("abc.txt")
	t.Logf("%v, %v", v, ok)

	v, ok = FileCutDotSuffix("")
	t.Logf("%v, %v", v, ok)

	v, ok = FileCutDotSuffix(".")
	t.Logf("%v, %v", v, ok)
}
