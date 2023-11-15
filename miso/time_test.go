package miso

import (
	"testing"
	"time"
)

func TestETimeScan(t *testing.T) {
	now := time.Now()
	var et ETime
	et.Scan(now.UnixMilli())
	tt := time.Time(et)
	t.Logf("MS: %v", tt)
	if now.Unix() != tt.Unix() {
		t.Log("now.Unix != tt.Unix")
		t.FailNow()
	}

	et.Scan(now.Unix())
	tt = time.Time(et)
	t.Logf("S: %v", tt)
	if now.Unix() != tt.Unix() {
		t.Log("now.Unix != tt.Unix")
		t.FailNow()
	}
}
