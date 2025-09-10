package expr

import (
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type Expr[T any] struct {
	p *vm.Program
}

func (e *Expr[T]) Eval(env T) (any, error) {
	return expr.Run(e.p, env)
}

// Compile Expr expression.
//
// The compiled *Expr can be reused concurrently.
//
// See https://expr-lang.org/docs/language-definition.
func Compile[T any](s string) (*Expr[T], error) {
	var t T
	program, err := expr.Compile(s, expr.Env(t))
	if err != nil {
		return nil, errs.WrapErr(err)
	}
	return &Expr[T]{
		p: program,
	}, nil
}

// Compile Expr expression.
//
// The compiled *Expr can be reused concurrently.
//
// See https://expr-lang.org/docs/language-definition.
func MustCompile[T any](s string) *Expr[T] {
	x, err := Compile[T](s)
	if err != nil {
		panic(errs.WrapErrf(err, "failed to compile expr: '%v", s))
	}
	return x
}
