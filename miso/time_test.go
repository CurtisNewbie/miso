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
	TestEqual(t, now.Unix(), tt.Unix())

	et.Scan(now.Unix())
	tt = time.Time(et)
	t.Logf("S: %v", tt)
	TestEqual(t, now.Unix(), tt.Unix())
}
