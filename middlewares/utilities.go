package middlewares

import (
	"context"

	"github.com/Arup3201/torb/api"
)

// NewContext returns a new Context that carries value u.
func NewContext(ctx context.Context, u string) context.Context {
	return context.WithValue(ctx, api.CTX_USER_KEY, u)
}

// FromContext returns the User value stored in ctx, if any.
func FromContext(ctx context.Context) (string, bool) {
	u, ok := ctx.Value(api.CTX_USER_KEY).(string)
	return u, ok
}
