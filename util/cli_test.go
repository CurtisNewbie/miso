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
