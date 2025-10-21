package testutil

import (
	"os"
	"path"
	"testing"

	"github.com/curtisnewbie/miso/util/osutil"
)

func FindTestdata(t *testing.T, relativePath string) string {
	td := "testdata"
	wdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wdir
	mf := "go.mod"
	for {
		cpath := path.Join(dir, td)
		ok, err := osutil.FileExists(cpath)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			return path.Join(cpath, relativePath)
		}
		mpath := path.Join(dir, mf)
		if osutil.TryFileExists(mpath) {
			// already the top level in project directory, give up
			break
		}
		dir = path.Dir(dir) // go up one level
	}
	t.Fatalf("testdata file: '**/%v' not found", path.Join(td, relativePath))
	return ""
}
