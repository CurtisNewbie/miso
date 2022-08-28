package config

import (
	"testing"
)

func TestParseProfile(t *testing.T) {

	args := make([]string, 2)
	args[0] = "profile=abc"
	args[1] = "--someflag"

	profile := ParseProfile(args)
	if profile != "abc" {
		t.Errorf("Expected abc, but got: %v", profile)
	}

	args2 := make([]string, 1)
	args2[0] = "--someflag"

	profile = ParseProfile(args2)
	if profile != "dev" {
		t.Errorf("Expected dev, but got: %v", profile)
	}
}



