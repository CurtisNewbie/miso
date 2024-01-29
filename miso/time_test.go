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

	t.Log(et.FormatClassicLocale())
	t.Log(et.FormatClassic())
}

func TestFuzzParseTime(t *testing.T) {
	tt, err := FuzzParseTime([]string{"2006-01-02 15:04:05", "2006/01/02 15:04:05"}, "2023/01/02 15:04:03.123192")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tt.String())
}
