package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/viant/a2a-protocol/schema"
	"github.com/viant/a2a-protocol/server"
	aauth "github.com/viant/a2a-protocol/server/auth"
	"github.com/viant/jsonrpc"
)

func main() {
	addr := getenv("A2A_ADDR", ":8080")
	rpcPath := getenv("A2A_JSONRPC_PATH", "/v1/jsonrpc")

    card := schema.AgentCard{
        Name:      "example-a2a-server",
        Endpoints: map[string]string{
            "jsonrpc": rpcPath,
            "rest":    "/v1",
            "sse":     "/v1/message:stream",
            "streamable": "/a2a",
        },
        Authentication: map[string]interface{}{
            "securitySchemes": map[string]map[string]string{
                "Bearer": {"type": "http", "scheme": "bearer", "bearerFormat": "JWT"},
            },
            "security": []map[string][]string{{"Bearer": {}}},
        },
    }
    // Spec-compliant capabilities object (also derives legacy list for compatibility)
    streaming, push, sth := true, true, false
    card.SetCapabilities(schema.AgentCapabilities{
        Streaming:             &streaming,
        PushNotifications:     &push,
        StateTransitionHistory: &sth,
    })

	// Build default handler with simple message/send and message/stream
	newOps := server.WithDefaultHandler(context.Background(), func(h *server.DefaultHandler) error {
		h.OnMessageSend = func(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error) {
			t := h.NewTask(contextID)
			h.CompleteText(t, "ok")
			return t, nil
		}
		h.OnMessageStream = func(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error) {
			t := h.NewTask(contextID)
			go h.StreamDemo(t)
			return t, nil
		}
		return nil
	})
	srv := server.New(card, server.WithOperations(newOps))
	// Inner mux with the actual endpoints
	inner := http.NewServeMux()
	srv.RegisterSSE(inner, "/v1")
    // Streamable endpoint (A2A): recommended base "/a2a"
    srv.RegisterStreaming(inner, "/a2a")
	srv.RegisterREST(inner)
    // Agent card is served at the well-known location only

	// Auth middleware and metadata endpoint
	policy := &aauth.Policy{Metadata: &aauth.ProtectedResourceMetadata{
		Resource:        "a2a",
		ScopesSupported: []string{"default"},
	}}
	authSvc := aauth.NewService(policy)

	// Outer mux: metadata + agent card + middleware-wrapped inner
	outer := http.NewServeMux()
	authSvc.RegisterHandlers(outer)
	outer.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(card)
	})
	outer.Handle("/", authSvc.Middleware(inner))

    log.Printf("A2A server listening on %s (SSE+JSON-RPC at /v1, Streamable at /a2a)", addr)
	log.Fatal(http.ListenAndServe(addr, outer))
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
