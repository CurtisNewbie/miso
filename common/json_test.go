package common

import (
	"testing"
)

type body struct {
	Mode mode
	Mysql mysql
}

type mysql struct {
	Enabled bool
	User string
	Password string
	Host string
	Port string
}

type mode struct {
	Production bool 
}

func TestReadJsonFile(t *testing.T) {
	f := "../app-conf-dev.json"	
	var j body
	e := ReadJsonFile(f, &j)
	if e != nil {
		t.Error(e)
		return
	}

	t.Logf("json: %+v", j)
}