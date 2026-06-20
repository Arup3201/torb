package middlewares

import (
	"context"
)

const (
	CTX_USER_KEY = "USER_ID"
)

// NewContext returns a new Context that carries value u.
func NewContext(ctx context.Context, u string) context.Context {
	return context.WithValue(ctx, CTX_USER_KEY, u)
}

// FromContext returns the User value stored in ctx, if any.
func FromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(CTX_USER_KEY).(string)
	return u, ok
}
