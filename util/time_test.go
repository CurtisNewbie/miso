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

func TestTimeAddSub(t *testing.T) {
	n := Now()
	t.Logf("now: %+v", n)
	v := n.Add(-time.Hour)
	t.Logf("v: %+v", v)
	if n.Sub(v) != time.Hour {
		t.Fatal("diff is not an hour")
	}
}

func TestTimeAddDate(t *testing.T) {
	n := Now()
	t.Logf("now: %+v", n)
	v := n.AddDate(1, 0, 0)
	t.Logf("n: %v, v: %v", n, v)

	v = n.AddDate(0, 1, 0)
	t.Logf("n: %v, v: %v", n, v)

	v = n.AddDate(0, 0, 1)
	t.Logf("n: %v, v: %v", n, v)

	if n.After(v) {
		t.Fatal("n should not be after v")
	}

	if v.Before(n) {
		t.Fatal("v should not be before n")
	}
}
