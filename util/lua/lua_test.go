package lua

import (
	"testing"

	"github.com/curtisnewbie/miso/util"
)

func TestRunLua(t *testing.T) {
	script := `
infof("one two three: %v", "four")
errorf("no no no: %v", "four")
infof("got, %v", myarg)
infof("table, %v", mytable)
infof("table name, %v", mytable["name"])
infof("table age, %v", mytable["age"])
infof("table age, %v", mytable["age"])
return 123.22
`
	res, err := Run[float64](script,
		WithGlobalStrTable("mytable", map[string]any{
			"name": "yongjie",
			"age":  100,
		}))
	util.Must(err)
	t.Logf("%#v", res)
}
