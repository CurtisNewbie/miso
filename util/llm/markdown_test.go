package llm

import "testing"

func TestEscapeMarkdownLatex(t *testing.T) {
	s := `$500M \$400M $300M`
	t.Log(EscapeMarkdownLatex(s))
}
