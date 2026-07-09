package atom

import (
	"testing"
	"time"
)

func TestETimeScan(t *testing.T) {
	now := time.Now()
	t.Logf("now: %v", now)

	var et Time
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
	tt, err := FuzzParseTime([]string{SQLDateTimeFormatWithT}, "2023-01-02T15:04:03")
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

func TestUnmarshalJSON(t *testing.T) {
	var et Time
	err := et.UnmarshalJSON([]byte("2025-04-09 09:40:10.123"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", et)

	err = et.UnmarshalJSON([]byte("2025-04-09"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", et)

	err = et.UnmarshalJSON([]byte("1744251041206"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", et)
}

func TestUnmarshalJSONUnixSeconds(t *testing.T) {
	// Dify API (and similar) return created_at as Unix seconds, not milliseconds.
	// e.g. 1737000000 = 2025-01-16, not 1970-01-21.
	unixSec := int64(1737000000)
	expected := time.Unix(unixSec, 0)

	var et Time
	if err := et.UnmarshalJSON([]byte("1737000000")); err != nil {
		t.Fatal(err)
	}
	if et.Unix() != expected.Unix() {
		t.Fatalf("Unix-seconds JSON: got %v (unix=%d), want %v (unix=%d)",
			et, et.Unix(), expected, expected.Unix())
	}
	t.Logf("Unix-sec 1737000000 → %v ✓", et)

	// Milliseconds above threshold should still work.
	unixMilli := int64(1744251041206)
	expectedMilli := time.UnixMilli(unixMilli)
	if err := et.UnmarshalJSON([]byte("1744251041206")); err != nil {
		t.Fatal(err)
	}
	if et.Unix() != expectedMilli.Unix() {
		t.Fatalf("Unix-milli JSON: got %v, want %v", et, expectedMilli)
	}
	t.Logf("Unix-milli 1744251041206 → %v ✓", et)
}

func TestEndOfDay(t *testing.T) {
	var et Time
	err := et.UnmarshalJSON([]byte("2025-04-09 09:40:10.123"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", et)
	t.Logf("%v", et.EndOfDay())
}

func TestTimeGoStringer(t *testing.T) {
	type dummy struct {
		Time Time
	}

	d := dummy{Time: Now()}
	t.Logf("%#v", d)
}

func TestLastWeekday(t *testing.T) {
	now := Now().StartOfDay()
	for k := range 7 {
		for i := range 7 {
			d := now.AddDate(0, 0, -i)
			m := d.LastWeekday(time.Weekday(k))
			t.Logf("%v (%v), %v (%v)", d, d.Weekday(), m, m.Weekday())
		}
	}
}

func TestNextWeekday(t *testing.T) {
	now := Now().StartOfDay()
	for k := range 7 {
		for i := range 7 {
			d := now.AddDate(0, 0, i)
			m := d.NextWeekday(time.Weekday(k))
			t.Logf("%v (%v), %v (%v)", d, d.Weekday(), m, m.Weekday())
		}
	}
}

func TestStartEndOfMonth(t *testing.T) {
	now := Now()
	t.Log(now)
	t.Log(now.StartOfMonth())
	t.Log(now.EndOfMonth())
	t.Log(now.StartOfHour())
	t.Log(now.EndOfHour())
	t.Log(now.StartOfMin())
	t.Log(now.EndOfMin())
	t.Log(now.StartOfWeek(time.Monday))
	t.Log(now.EndOfWeek(time.Monday))
	t.Log(now.StartOfWeek(time.Sunday))
	t.Log(now.EndOfWeek(time.Sunday))
}
