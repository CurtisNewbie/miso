package testutil

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/curtisnewbie/miso/util/osutil"
)

func FindTestdata(t *testing.T, relativePath string) string {
	p, err := FindTestdataPath(relativePath)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func FindTestdataPath(relativePath string) (string, error) {
	td := "testdata"
	wdir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wdir
	mf := "go.mod"
	for {
		cpath := path.Join(dir, td)
		ok, err := osutil.FileExists(cpath)
		if err != nil {
			return "", err
		}
		if ok {
			return path.Join(cpath, relativePath), nil
		}
		mpath := path.Join(dir, mf)
		if osutil.TryFileExists(mpath) {
			// already the top level in project directory, give up
			break
		}
		dir = path.Dir(dir) // go up one level
	}
	return "", fmt.Errorf("testdata file: '**/%v' not found", path.Join(td, relativePath))
}
