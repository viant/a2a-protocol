package server

import (
	"net/http"

	sse "github.com/viant/jsonrpc/transport/server/http/sse"
)

// RegisterSSE mounts viant/jsonrpc SSE+message endpoints under the given base path.
// Example: base="/v1" exposes:
// - GET  /v1/message:stream  (SSE)
// - POST /v1/message:send    (non-streaming JSON-RPC)
// - POST /v1/message:stream  (streaming JSON-RPC)
func (s *Server) RegisterSSE(mux *http.ServeMux, base string) {
	handler := sse.New(newA2AHandler(s),
		sse.WithURI(base+"/message:stream"),
		sse.WithMessageURI(base+"/message:send"),
	)
	mux.Handle(base+"/", handler)
	mux.Handle(base+"/message:stream", handler)
	mux.Handle(base+"/message:send", handler)
}
