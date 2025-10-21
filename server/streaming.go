package server

import (
    streamable "github.com/viant/jsonrpc/transport/server/http/streamable"
    "net/http"
)

// RegisterStreaming mounts the JSON-RPC Streamable-HTTP handler.
// Recommended base is "/a2a" for A2A servers.
func (s *Server) RegisterStreaming(mux *http.ServeMux, base string) {
    if base == "" {
        base = "/a2a"
    }
    h := streamable.New(newA2AHandler(s), streamable.WithURI(base))
    mux.Handle(base, h)
}
