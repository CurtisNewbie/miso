package miso

import (
	"sync"
	"testing"
)

func TestNewSpan(t *testing.T) {
	ec := EmptyRail()
	ec.Infof("Parent Span")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		cc := ec.NextSpan()
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			cc.Infof("Child Span, j: %v", j)
		}(i)
	}

	wg.Wait()
}

func TestGetShortFnName(t *testing.T) {
	if v := getShortFnName("shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := getShortFnName("pck.shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := getShortFnName("vvvv/pck.shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := getShortFnName("gggg/vvvv/pck.shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}
}
