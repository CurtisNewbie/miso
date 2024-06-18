package encoding

import "testing"

func TestSWriteJson(t *testing.T) {
	type dummy struct {
		Name string
	}
	d := dummy{Name: "aha"}
	s, err := SWriteJson(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)
}
