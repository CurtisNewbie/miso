package expr

import (
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/lru"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type Expr[T any] struct {
	p *vm.Program
}

func (e *Expr[T]) Eval(env T) (any, error) {
	r, err := expr.Run(e.p, env)
	return r, errs.Wrap(err)
}

// Compile Expr expression.
//
// If T is map, use [CompileEnv] instead.
//
// The compiled *Expr can be reused concurrently.
//
// See https://expr-lang.org/docs/language-definition.
func Compile[T any](s string) (*Expr[T], error) {
	var t T
	program, err := expr.Compile(s, expr.Env(t))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return &Expr[T]{
		p: program,
	}, nil
}

// Compile Expr expression.
//
// If T is map, use [MustCompileEnv] instead.
//
// The compiled *Expr can be reused concurrently.
//
// See https://expr-lang.org/docs/language-definition.
func MustCompile[T any](s string) *Expr[T] {
	x, err := Compile[T](s)
	if err != nil {
		panic(errs.Wrapf(err, "failed to compile expr: '%v", s))
	}
	return x
}

// Compile Expr expression with Env example.
//
// The compiled *Expr can be reused concurrently.
//
// See https://expr-lang.org/docs/language-definition.
func CompileEnv[T any](s string, env T) (*Expr[T], error) {
	program, err := expr.Compile(s, expr.Env(env))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return &Expr[T]{
		p: program,
	}, nil
}

// Compile Expr expression with Env example.
//
// The compiled *Expr can be reused concurrently.
//
// See https://expr-lang.org/docs/language-definition.
func MustCompileEnv[T any](s string, env T) *Expr[T] {
	x, err := CompileEnv[T](s, env)
	if err != nil {
		panic(errs.Wrapf(err, "failed to compile expr: '%v", s))
	}
	return x
}

// Compile and Run Expr expression.
//
// See https://expr-lang.org/docs/language-definition.
func Eval(s string, t any) (any, error) {
	r, err := expr.Eval(s, t)
	return r, errs.Wrap(err)
}

type PooledExpr[T any] struct {
	threshold int
	m         lru.LRU[*Expr[T]]
}

func (e *PooledExpr[T]) Eval(s string, env T) (any, error) {
	if e.threshold > 0 && len(s) > e.threshold {
		return Eval(s, env)
	}

	ex, err := e.m.GetElse(s, func() (*Expr[T], error) {
		return Compile[T](s)
	})
	if err != nil {
		return nil, err
	}
	return ex.Eval(env)
}

// Create PooledExpr.
//
// The compiled *PooledExpr can be reused concurrently.
//
// T can be struct or map, but the overall structure must be the same (e.g., map with same kinds of keys).
//
// cacheSize: max number of *Expr in cache.
func NewPooledExpr[T any](cacheSize int) *PooledExpr[T] {
	c, err := lru.New[*Expr[T]](cacheSize)
	if err != nil {
		panic(err)
	}
	return &PooledExpr[T]{
		threshold: 1024,
		m:         c,
	}
}
