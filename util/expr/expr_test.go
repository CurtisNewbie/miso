package expr

import (
	"testing"
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
			_, err := pool.Eval(`A + B`, p)
			if err != nil {
				b.Fatal(err)
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
