package auth

// ProtectedResourceMetadata describes the OAuth/OIDC metadata for a protected resource.
// This is served at /.well-known/oauth-protected-resource.
type ProtectedResourceMetadata struct {
	Resource         string   `json:"resource"`
	Issuer           string   `json:"issuer,omitempty"`
	AuthorizationURI string   `json:"authorization_uri,omitempty"`
	TokenEndpoint    string   `json:"token_endpoint,omitempty"`
	ScopesSupported  []string `json:"scopes_supported,omitempty"`
}

// Policy controls which endpoints are protected and metadata returned to clients.
type Policy struct {
	// If set, requests with path that has this prefix are bypassed (no auth).
	ExcludePrefix string
	// Metadata served for the resource.
	Metadata *ProtectedResourceMetadata
	// If true, prefer ID token from source; otherwise use access token when acquiring.
	UseIDToken bool
}
