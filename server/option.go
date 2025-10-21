package server

import (
	"github.com/viant/jsonrpc/transport"
)

// NewOperationsFunc constructs Operations for a given server and notifier.
type NewOperationsFunc func(srv *Server, transport transport.Transport) Operations

// Option configures the Server.
type ServerOption func(*Server)

// WithOperations sets a custom Operations factory for the server.
func WithOperations(factory NewOperationsFunc) ServerOption {
	return func(s *Server) { s.opsFactory = factory }
}
