package lua

import (
	"testing"
)

func TestRunLua(t *testing.T) {
	script := `
printf("one two three: %v", "four")
printf("no no no: %v", "four")
printf("got, %v", myarg)
printf("table, %v", mytable)
printf("table name, %v", mytable["name"])
printf("table age, %v", mytable["age"])
printf("table age, %v", mytable["age"])
return "1234.11"
`
	res, err := Run[float64](script,
		WithGlobalStrTable("mytable", map[string]any{
			"name": "yongjie",
			"age":  100,
		}))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", res)
}
