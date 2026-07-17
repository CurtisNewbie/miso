package llm

import (
	"testing"
)

func TestEscapeMarkdownLatex(t *testing.T) {
	s := `$500M \$400M $300M`
	t.Log(EscapeMarkdownLatex(s))
}

func TestStripMarkdownFence(t *testing.T) {
	tab := []struct {
		in  string
		exp string
	}{
		{"```json\n{}\n```", "{}"},
		{"```\n{}\n```", "{}"},
		{"```json\n```", ""},
		{"```\n```", ""},
		{"{}", "{}"},
		{"  ```json\n{\"a\":1}\n```  ", `{"a":1}`},
	}
	for _, v := range tab {
		act := StripMarkdownFence(v.in)
		if act != v.exp {
			t.Fatalf("in: %q, exp: %q, act: %q", v.in, v.exp, act)
		}
	}
}

func TestTagExtractor(t *testing.T) {
	tx, err := TagExtractor("test")
	if err != nil {
		t.Fatal(err)
	}
	tab := [][]string{
		{"<test", ""},
		{"<test>", ""},
		{"<test>a", "a"},
		{"<test>ab", "ab"},
		{"<test>ab\n", "ab\n"},
		{"<test>ab<", "ab"},
		{"<test>ab</", "ab"},
		{"<test>ab</t", "ab"},
		{"<test>ab</te", "ab"},
		{"<test>ab</tes", "ab"},
		{"<test>ab</test", "ab"},
		{"<test>ab</test>", "ab"},
		{"<test>ab\nc</test>", "ab\nc"},
		{"<test>ab<bbbb", "ab<bbbb"},
		{"<test>ab<bbbb>", "ab<bbbb>"},
		{"<test>1 < 2", "1 < 2"},
		{"<test>1 < 2</test>", "1 < 2"},
		{"mentions `<test>` tag\n\n<test>real</test>", "real"},
		{"<test>a</test> then <test>b</test>", "b"},
	}
	for _, v := range tab {
		r := tx.Content(v[0])
		if r != v[1] {
			t.Fatalf("ori: %v, exp: '%v', act: '%v'", v[0], v[1], r)
		}
		t.Logf("ori: %v, exp: '%v', act: '%v'", v[0], v[1], r)
	}
}
