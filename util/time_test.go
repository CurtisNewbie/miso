package util

import (
	"testing"
	"time"
)

func TestETimeScan(t *testing.T) {
	now := time.Now()
	t.Logf("now: %v", now)

	var et ETime
	et.Scan(now.UnixMilli())
	t.Logf("MS: %v", et)
	if now.Unix() != et.Unix() {
		t.Log("now.Unix != tt.Unix")
		t.FailNow()
	}

	et.Scan(now.Unix())
	t.Logf("S: %v", et)
	if now.Unix() != et.Unix() {
		t.Log("now.Unix != tt.Unix")
		t.FailNow()
	}

	t.Log(et.FormatClassicLocale())
	t.Log(et.FormatClassic())

	et.Scan(now.UnixMilli() - 100_000)
	t.Logf("et after sub: %v", et)
}

func TestFuzzParseTime(t *testing.T) {
	tt, err := FuzzParseTime([]string{"2006-01-02 15:04:05", "2006/01/02 15:04:05"}, "2023/01/02 15:04:03.123192")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tt.String())
}

func TestTimeAdd(t *testing.T) {
	n := Now()
	t.Logf("now: %+v", n)
	v := n.Add(-time.Hour)
	t.Logf("v: %+v", v)
	if n.ToTime().Sub(v) != time.Hour {
		t.Fatal("diff is not an hour")
	}
}
