package common

import (
	"testing"
	"time"
)

func TestWTime(t *testing.T) {
	now := time.Now()
	casted := WTime(now)
	t.Logf("before: %v, after: %v", now, casted)

	_, e := casted.MarshalJSON()
	if e != nil {
		t.Error(e)
	}
}

func TestTTime(t *testing.T) {
	now := time.Now()
	casted := TTime(now)
	t.Logf("before: %v, after: %v", now, casted)

	m, e := casted.MarshalJSON()
	if e != nil {
		t.Error(e)
	}
	t.Log(string(m))

	var wp TTime
	e = wp.UnmarshalJSON(m)
	if e != nil {
		t.Error(e)
	}
	t.Log(wp)
}

func TestTranslateFormatPerf(t *testing.T) {
	before := "yyyy-MM-dd HH:mm:ss"

	start := time.Now().UnixMilli()
	total := 1_000_000

	for i := 0; i < total; i++ {
		after := TranslateFormat(before)
		if after != "2006-01-02 15:04:05" {
			t.Error(after)
			return
		}
	}

	end := time.Now().UnixMilli()
	t.Logf("time: %dms, total: %d, perf: %.5fms each", end-start, total, float64(end-start)/float64(total))
}

func TestTranslateFormatOne(t *testing.T) {
	before := "yyyy-MM-dd HH:mm:ss"
	after := TranslateFormat(before)
	if after != "2006-01-02 15:04:05" {
		t.Error(after)
		return
	}
}

func TestTranslateFormatTwo(t *testing.T) {
	before := "yyyy/MM/dd HH:mm"
	after := TranslateFormat(before)
	if after != "2006/01/02 15:04" {
		t.Error(after)
		return
	}
}

func TestTranslateFormatThree(t *testing.T) {
	before := "yyyy-MM-dd HH:mm:ss"
	after := TranslateFormat(before)
	t.Logf("Format: %s, %v", after, time.Now().Format(after))

	before = "yyyy-MM-dd HH:mm"
	after = TranslateFormat(before)
	t.Logf("Format: %s, %v", after, time.Now().Format(after))

	before = "yyyy/MM/dd HH:mm:ss"
	after = TranslateFormat(before)
	t.Logf("Format: %s, %v", after, time.Now().Format(after))

	before = "yyyy/MM/dd HH:mm"
	after = TranslateFormat(before)
	t.Logf("Format: %s, %v", after, time.Now().Format(after))

	before = "yyyy/MM/dd"
	after = TranslateFormat(before)
	t.Logf("Format: %s, %v", after, time.Now().Format(after))
}

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
