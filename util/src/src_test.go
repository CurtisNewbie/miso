package src

import "testing"

func TestUnsafeGetShortFnName(t *testing.T) {
	if v := unsafeGetShortFnName(""); v != "" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("shortFunc"); v != "shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("vvvv/pck.shortFunc"); v != "pck.shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vvvv/pck.(*abc).shortFunc"); v != "(*abc).shortFunc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vvvv/pck.(*abc).shortFunc.func2"); v != "(*abc).shortFunc.func2" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vvvv/(*abc).shortFunc.func2"); v != "(*abc).shortFunc.func2" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vvvv/pck.(*abc).shortFunc.func2.func"); v != "(*abc).shortFunc.func2.func" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("gggg/vvvv/pck.(*abc).shortFunc.funcA.funcB.do"); v != "(*a).shortFunc.funcA.funcB.do" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("[...]abc"); v != "[...]abc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("[..]abc"); v != "[..]abc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("..]abc"); v != "..]abc" {
		t.Fatal(v)
	}

	if v := unsafeGetShortFnName("abc.(*HttpProxy).AddHealthcheckFilter"); v != "(*H).AddHealthcheckFilter" {
		t.Fatal(v)
	}
}
func BenchmarkUnsafeGetShortFnName(b *testing.B) {
	/*
		goos: darwin
		goarch: arm64
		pkg: github.com/curtisnewbie/miso/miso
		cpu: Apple M3 Pro
		=== RUN   BenchmarkUnsafeGetShortFnName
		BenchmarkUnsafeGetShortFnName
		BenchmarkUnsafeGetShortFnName-11        203311693                5.792 ns/op           0 B/op          0 allocs/op
	*/
	for range b.N {
		unsafeGetShortFnName("gggg/vvvv/pck.(*abc).shortFunc")
	}
}
