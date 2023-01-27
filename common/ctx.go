package common

import (
	"context"

	"github.com/sirupsen/logrus"
)

var (
	nilUser = User{}
)

// Prepared execution context
type ExecContext struct {
	Ctx  context.Context // request context
	User User            // optional, use Authenticated() first before reading this value
	Log  *logrus.Entry   // logger with tracing info
	auth bool
}

// Check whether current execution is authenticated, if so, one may read User from ExecContext
func (in *ExecContext) Authenticated() bool {
	return in.auth
}

// Create empty ExecContext
func EmptyExecContext() ExecContext {
	ctx := context.Background()

	if ctx.Value(X_SPANID) == nil {
		ctx = context.WithValue(ctx, X_SPANID, RandLowerAlphaNumeric(16))
	}

	if ctx.Value(X_TRACEID) == nil {
		ctx = context.WithValue(ctx, X_TRACEID, RandLowerAlphaNumeric(16))
	}

	return NewExecContext(ctx, nil)
}

// Create new ExecContext
func NewExecContext(ctx context.Context, user *User) ExecContext {
	hasUser := user != nil
	var u User
	if hasUser {
		u = *user
	} else {
		u = nilUser
	}
	return ExecContext{Ctx: ctx, User: u, Log: TraceLogger(ctx)}
}
