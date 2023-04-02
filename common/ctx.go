package common

import (
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	Tx   *gorm.DB        // Transaction
}

// Check whether current execution is authenticated, if so, one may read User from ExecContext
func (in *ExecContext) Authenticated() bool {
	return in.auth
}

// Get username if found, else empty string
func (in *ExecContext) Username() string {
	return in.User.Username
}

// Get user id if found, else empty string
func (in *ExecContext) UserId() string {
	return in.User.UserId
}

// Get user no if found, else empty string
func (in *ExecContext) UserNo() string {
	return in.User.UserNo
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
