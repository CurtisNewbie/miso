package redis

import (
	"testing"
	"time"

	"github.com/curtisnewbie/gocommon/common"
)

func TestRCache(t *testing.T) {
	common.LoadConfigFromFile("../app-conf-dev.yml")
	InitRedisFromProp()

	keypre := "test:rcache:key:"
	exp := 60 * time.Second

	rcache := NewRCache(exp)
	e := rcache.Put(keypre + "1", "2")
	if e != nil {
		t.Fatal(e)
	}

	val, e := rcache.GetElse(keypre + "1", nil)
	if e != nil {
		t.Fatal(e)
	}
	if val != "2" {
		t.Fatalf("val '%v' != \"2\"", val)
	}
}
