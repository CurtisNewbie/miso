package miso

import (
	"os"
	"path"
	"testing"

	"github.com/curtisnewbie/miso/util"
)

// Prepare Test Environment
//
// Before calling this method, you should make sure related modules are imported in go test file, or else the dependencies may not be bootstrapped properly.
func PrepareTestEnv(t *testing.T) Rail {
	rail := EmptyRail()
	SetProp(PropAppTestEnv, true)
	cf := tryFindConfFile(rail, t)
	if cf != "" {
		err := LoadConfigFromFile(cf, rail)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := App().callConfigLoaders(rail); err != nil {
		t.Fatal(err)
	}
	if err := App().callPreServerBoot(rail); err != nil {
		t.Fatal(err)
	}
	if err := App().callBoostrapComp(rail); err != nil {
		t.Fatal(err)
	}
	if err := App().callPostServerBoot(rail); err != nil {
		t.Fatal(err)
	}

	// marked as fully bootstrapped
	App().fullyBoostrapped.Store(true)

	return rail
}

func tryFindConfFile(rail Rail, t *testing.T) string {
	cf := "conf.yml"
	wdir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wdir
	mf := "go.mod"
	for {
		cpath := path.Join(dir, cf)
		ok, err := util.FileExists(cpath)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			return cpath
		}
		mpath := path.Join(dir, mf)
		if util.TryFileExists(mpath) {
			// already the top level in project directory, give up
			break
		}
		dir = path.Dir(dir) // go up one level
	}

	rail.Warnf("Config file `%v` not found in project directory", cf)
	return ""
}
