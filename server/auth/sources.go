package auth

import (
	"context"
	"net/http"
)

// AccessTokenSource returns an OAuth2 access token for a protected resource.
type AccessTokenSource interface {
	AccessToken(ctx context.Context, r *http.Request, resource string) (string, error)
}

// IDTokenSource returns an OIDC ID token for a protected resource/user.
type IDTokenSource interface {
	IDToken(ctx context.Context, r *http.Request, resource string) (string, error)
}
