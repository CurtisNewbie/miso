package json

import "testing"

func TestSWriteJson(t *testing.T) {
	type dummy struct {
		Name string `json:"name"`
	}
	d := dummy{Name: "aha"}
	s, err := SWriteJson(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)
}

func TestSWriteIndent(t *testing.T) {
	type dummy struct {
		Name string `json:"name"`
	}
	d := dummy{Name: "aha"}
	s, err := SWriteIndent(d)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)
}

func TestSParseJsonAs(t *testing.T) {
	type dummy struct {
		Name string `json:"name"`
	}
	d, err := SParseJsonAs[dummy](`{ "name": "yes" }`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", d)
}

func TestParseJsonAs(t *testing.T) {
	type dummy struct {
		Name string `json:"name"`
	}
	d, err := ParseJsonAs[dummy]([]byte(`{ "name": "yes" }`))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", d)
}
