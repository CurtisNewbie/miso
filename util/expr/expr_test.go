package expr

import (
	"testing"

	"github.com/curtisnewbie/miso/miso"
)

func BenchmarkPooledExpr(b *testing.B) {
	type P struct {
		A string
		B string
	}
	p := P{A: "AAA", B: "BBB"}
	pool := NewPooledExpr[P](100)

	b.Run("pool.Eval", func(b *testing.B) {
		for range b.N {
			v, err := pool.Eval(`A + B`, p)
			if err != nil {
				b.Fatal(err)
			}
			if v != "AAABBB" {
				b.Fatal("no right")
			}
		}
	})

	b.Run("Eval", func(b *testing.B) {
		for range b.N {
			_, err := Eval(`A + B`, p)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	cp, err := Compile[P](`A + B`)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("Compile.Eval", func(b *testing.B) {
		for range b.N {
			_, err := cp.Eval(p)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestPooledExpr(t *testing.T) {
	miso.SetLogLevel("debug")
	type P struct {
		A string
		B string
	}
	p := P{A: "AAA", B: "BBB"}
	pool := NewPooledExpr[P](1)
	v, err := pool.Eval(`A + B`, p)
	if err != nil {
		t.Fatal(err)
	}
	if v != "AAABBB" {
		t.Fatal("no right")
	}
	t.Log(v)

	v, err = pool.Eval(`A + B`, p)
	if err != nil {
		t.Fatal(err)
	}
	if v != "AAABBB" {
		t.Fatal("no right")
	}
	t.Log(v)

	v, err = pool.Eval(`B + A`, p)
	if err != nil {
		t.Fatal(err)
	}
	if v != "BBBAAA" {
		t.Fatal("no right")
	}
	t.Log(v)

	v, err = pool.Eval(`A + A`, p)
	if err != nil {
		t.Fatal(err)
	}
	if v != "AAAAAA" {
		t.Fatal("no right")
	}
	t.Log(v)
}
