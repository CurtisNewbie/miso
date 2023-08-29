package core

import "testing"

func TestFileExists(t *testing.T) {
	ok, e := FileExists("file.go")
	TestTrue(t, ok)
	TestIsNil(t, e)

	ok, e = FileExists("file_not_found")
	TestFalse(t, ok)
	TestIsNil(t, e)
}
