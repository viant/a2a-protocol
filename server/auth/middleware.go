package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Service provides HTTP middleware and metadata endpoint.
type Service struct {
	Policy *Policy
	// Optional token sources used when Authorization header is missing.
	AccessSource AccessTokenSource
	IDSource     IDTokenSource
}

func NewService(p *Policy) *Service { return &Service{Policy: p} }

// RegisterHandlers registers metadata endpoint.
func (s *Service) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if s.Policy != nil && s.Policy.Metadata != nil {
			_ = json.NewEncoder(w).Encode(s.Policy.Metadata)
			return
		}
		_ = json.NewEncoder(w).Encode(&ProtectedResourceMetadata{Resource: "default"})
	})
}

// Middleware enforces Bearer auth for protected A2A HTTP endpoints.
    // Bypasses metadata endpoints and the agent card.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Bypass OPTIONS, agent card, and metadata endpoint
        if r.Method == http.MethodOptions || path == "/.well-known/oauth-protected-resource" || path == "/.well-known/agent-card.json" {
            next.ServeHTTP(w, r)
            return
        }
		if s.Policy != nil && s.Policy.ExcludePrefix != "" && strings.HasPrefix(path, s.Policy.ExcludePrefix) {
			next.ServeHTTP(w, r)
			return
		}

		// Require Authorization: Bearer ... for all other A2A routes
		authz := r.Header.Get("Authorization")
		if hasBearer(authz) {
			// Attach token to context for downstream use
			ctx := WithToken(r.Context(), &Token{Raw: authz})
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			return
		}

		// Attempt to acquire a token from configured sources
		if s.AccessSource != nil || s.IDSource != nil {
			resource := "a2a"
			if s.Policy != nil && s.Policy.Metadata != nil && s.Policy.Metadata.Resource != "" {
				resource = s.Policy.Metadata.Resource
			}
			var token string
			var err error
			if s.Policy != nil && s.Policy.UseIDToken && s.IDSource != nil {
				token, err = s.IDSource.IDToken(r.Context(), r, resource)
			} else if s.AccessSource != nil {
				token, err = s.AccessSource.AccessToken(r.Context(), r, resource)
			}
			if err == nil && strings.TrimSpace(token) != "" {
				authz = "Bearer " + strings.TrimSpace(token)
				r.Header.Set("Authorization", authz)
				ctx := WithToken(r.Context(), &Token{Raw: authz})
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}
		}

		// Missing/invalid creds: 401 with WWW-Authenticate and resource metadata
		w.Header().Set("WWW-Authenticate", s.wwwAuthenticateHeader(r))
		w.WriteHeader(http.StatusUnauthorized)
		// Return a small JSON message (optional)
		_, _ = w.Write([]byte(`{"error":"Unauthorized: Bearer token required"}`))
	})
}

func hasBearer(h string) bool {
	if h == "" {
		return false
	}
	h = strings.TrimSpace(h)
	return strings.HasPrefix(strings.ToLower(h), "bearer ") && len(h) > len("bearer ")
}

func (s *Service) wwwAuthenticateHeader(r *http.Request) string {
	// Build resource_metadata absolute URL
	proto := headerOrDefault(r, "X-Forwarded-Proto", "http")
	host := headerOrDefault(r, "X-Forwarded-Host", r.Host)
	metaURL := fmt.Sprintf("%s://%s/.well-known/oauth-protected-resource", proto, host)
	scope := ""
	if s.Policy != nil && s.Policy.Metadata != nil && len(s.Policy.Metadata.ScopesSupported) > 0 {
		scope = fmt.Sprintf(`, scope="%s"`, strings.Join(s.Policy.Metadata.ScopesSupported, " "))
	}
	return fmt.Sprintf(`Bearer resource_metadata="%s"%s`, metaURL, scope)
}

func headerOrDefault(r *http.Request, name, fallback string) string {
	v := r.Header.Get(name)
	if v == "" {
		return fallback
	}
	return v
}
