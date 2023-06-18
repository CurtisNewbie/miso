package common

import (
	"context"
	"strconv"

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
	auth bool            // is authenticated
}

// Check whether current execution is authenticated, if so, one may read User from ExecContext
func (c *ExecContext) Authenticated() bool {
	return c.auth
}

// Get username if found, else empty string
func (c *ExecContext) Username() string {
	return c.User.Username
}

// Get user id if found, else empty string
func (c *ExecContext) UserId() string {
	return c.User.UserId
}

// Get user no if found, else empty string
func (c *ExecContext) UserNo() string {
	return c.User.UserNo
}

// Get user's id as int if found, else 0
//
// Basically the same as UserIdI except that error is ignored
func (c *ExecContext) UserIdInt() int {
	i, _ := c.UserIdI()
	return i
}

// Get user's id as int if found, else 0
func (c *ExecContext) UserIdI() (int, error) {
	if c.User.UserId == "" {
		return 0, nil
	}

	return strconv.Atoi(c.User.UserId)
}

// Create a new ExecContext with a new SpanId
func (c *ExecContext) NextSpan() ExecContext {
	// X_TRACE_ID is propagated as parent context, we only need to create a new X_SPAN_ID
	ctx := context.WithValue(c.Ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	return NewExecContext(ctx, &c.User)
}

// Create empty ExecContext
func EmptyExecContext() ExecContext {
	ctx := context.Background()

	if ctx.Value(X_SPANID) == nil {
		ctx = context.WithValue(ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	if ctx.Value(X_TRACEID) == nil {
		ctx = context.WithValue(ctx, X_TRACEID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
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

	if ctx.Value(X_SPANID) == nil {
		ctx = context.WithValue(ctx, X_SPANID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	if ctx.Value(X_TRACEID) == nil {
		ctx = context.WithValue(ctx, X_TRACEID, RandLowerAlphaNumeric(16)) //lint:ignore SA1029 keys must be exposed for user to use
	}

	return ExecContext{Ctx: ctx, User: u, Log: TraceLogger(ctx), auth: user != nil}
}

func GetCtxStr(ctx context.Context, key string) string {
	v := ctx.Value(key)
	if v == nil {
		return ""
	}
	sv, ok := v.(string)
	if ok {
		return sv
	}
	return ""
}

func SelectExecContext(cs ...ExecContext) ExecContext {
	if len(cs) > 0 {
		return cs[0]
	} else {
		return EmptyExecContext()
	}
}
