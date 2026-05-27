package llm

import "testing"

func TestParseLLMJsonAs(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	tab := []struct {
		in  string
		exp payload
	}{
		{"{}", payload{}},
		{`{"name":"alice"}`, payload{"alice"}},
		{"```json\n{\"name\":\"bob\"}\n```", payload{"bob"}},
		{"```\n{\"name\":\"carol\"}\n```", payload{"carol"}},
		{"```json\n```", payload{}},
	}
	for _, v := range tab {
		act, err := ParseLLMJsonAs[payload](v.in)
		if err != nil {
			t.Fatalf("in: %q, unexpected error: %v", v.in, err)
		}
		if act != v.exp {
			t.Fatalf("in: %q, exp: %+v, act: %+v", v.in, v.exp, act)
		}
	}
}
