package auth

import "context"

type tokenKeyType string

const tokenKey tokenKeyType = "a2a-auth-token"

// Token represents a bearer token (raw header value).
type Token struct {
	// Raw is the full Authorization header value, e.g. "Bearer <token>".
	Raw string
}

// WithToken attaches a token to context.
func WithToken(ctx context.Context, t *Token) context.Context {
	return context.WithValue(ctx, tokenKey, t)
}

// FromContext extracts token if present.
func FromContext(ctx context.Context) (*Token, bool) {
	v := ctx.Value(tokenKey)
	if v == nil {
		return nil, false
	}
	t, ok := v.(*Token)
	return t, ok
}
