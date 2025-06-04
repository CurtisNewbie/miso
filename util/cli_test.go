package util

import (
	"flag"
	"os"
	"testing"
)

func TestFlagStrSlice(t *testing.T) {
	os.Args = []string{"test", "-l", "apple", "-l", "juice"}
	sf := FlagStrSlice("l", "list of strings")
	flag.Parse()
	t.Log(sf.String())
}

func TestRunEmbedPy(t *testing.T) {
	py := "python3"
	script := `
import sys
print("args:", sys.argv)
`
	out, err := RunPyScript(py, script, []string{})
	t.Log(string(out))
	if err != nil {
		t.Fatal(err)
	}
}
