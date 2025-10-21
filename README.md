# Agent2Agent (A2A) Protocol – Go Implementation

Go implementation of the Agent2Agent (A2A) protocol. This repository provides the core schema, a production-friendly server with streaming transports, and a minimal client for interacting with A2A agents.

- Schema: Core types (AgentCard, Task, Message, Parts, Artifact)
- Server: JSON-RPC entry points with SSE and Streamable HTTP, plus REST helpers
- Client: Minimal client for `message/send`, `message/stream`, and task operations
- Auth: Simple hooks/middleware for transport-layer auth (HTTP headers)

Status: SSE and Streamable HTTP transports supported. See below for endpoints, client usage, and migration notes.

## Overview

This monorepo tracks the protocol types and a reference server/client in one place. The implementation follows the spec under `schema/spec.txt`, including capability advertisement via `AgentCard` and a well-known discovery endpoint.

## Requirements

- Go 1.23+ (module sets `go 1.23.4`)

## Install

- Library (schema, server, client):
  - `go get github.com/viant/a2a-protocol@latest`
- Example server binary:
  - `go run ./cmd/a2a-server`

## Quick Start

Start a simple A2A server with SSE and Streamable HTTP endpoints:

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"

    "github.com/viant/a2a-protocol/schema"
    "github.com/viant/a2a-protocol/server"
    "github.com/viant/jsonrpc"
)

func main() {
    card := schema.AgentCard{Name: "example-a2a-server"}
    streaming := true
    card.SetCapabilities(schema.AgentCapabilities{Streaming: &streaming})

    ops := server.WithDefaultHandler(context.Background(), func(h *server.DefaultHandler) error {
        h.OnMessageSend = func(ctx context.Context, msgs []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error) {
            t := h.NewTask(contextID)
            h.CompleteText(t, "ok")
            return t, nil
        }
        return nil
    })

    srv := server.New(card, server.WithOperations(ops))
    mux := http.NewServeMux()
    srv.RegisterSSE(mux, "/v1")
    srv.RegisterStreaming(mux, "/a2a")

    // Serve well-known Agent Card
    http.HandleFunc("/.well-known/agent-card.json", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(card)
    })
    http.Handle("/", mux)

    log.Println("listening on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Minimal client that autodetects streaming transport (Streamable HTTP or SSE):

```go
ctx := context.Background()
headers := http.Header{"Authorization": {"Bearer <token>"}}
cli, _ := client.AutoStreamClient(ctx, "http://localhost:8080/a2a", headers, nil)
task, _ := cli.StreamMessage(ctx, []schema.Message{{Role: "user", Parts: []schema.Part{schema.TextPart{Type: "text", Text: "hello"}}}}, nil, nil)
_ = task
```

## Capabilities (AgentCard)

The `AgentCard.capabilities` now follows the spec-compliant object shape (AgentCapabilities):

- `streaming?: boolean`
- `pushNotifications?: boolean`
- `stateTransitionHistory?: boolean`
- `extensions?: AgentExtension[]`

Backward compatibility: the server accepts the legacy `[]string` shape on input and will still populate `AgentCard.Capabilities []string`. When an object is present, JSON marshaling prefers the object form.

Gating: server features are enabled based on capabilities:

- Streaming methods (`message/stream`, `tasks/resubscribe`) require `capabilities.streaming: true`.
- Push notification endpoints require `capabilities.pushNotifications: true`.

To set capabilities in code use `schema.AgentCard.SetCapabilities(...)`, which also derives the legacy list for compatibility.

Extensions: you can declare protocol extensions via `capabilities.extensions` to advertise non-core features or requirements.

Example:

```go
card := schema.AgentCard{Name: "example-a2a-server"}
streaming := true
ext := []schema.AgentExtension{{
    URI:         "https://developers.google.com/identity/protocols/oauth2",
    Description: ptr("Google OAuth 2.0 authentication"),
    Required:    ptr(false),
    Params:      map[string]interface{}{"audience": "example"},
}}
card.SetCapabilities(schema.AgentCapabilities{
    Streaming:  &streaming,
    Extensions: ext,
})
```

Note: Extensions are metadata for discovery/negotiation; behavior depends on the extension semantics in your agent.

## Transports and Endpoints

By default the server exposes both streaming transports and a single Agent Card:

- SSE (JSON-RPC over SSE):
  - `GET  /v1/message:stream` – opens SSE stream
  - `POST /v1/message:send`   – non-streaming JSON-RPC
  - `POST /v1/message:stream` – request that may stream updates

- Streamable HTTP (single endpoint):
  - `POST/GET /a2a` (recommended base for A2A)

- Agent Card (spec-compliant discovery):
  - `GET /.well-known/agent-card.json`

You can mount either or both via the server helpers:

```go
srv := server.New(card, server.WithOperations(newOps))
inner := http.NewServeMux()
srv.RegisterSSE(inner, "/v1")          // SSE endpoints under /v1
srv.RegisterStreaming(inner, "/a2a")    // Streamable HTTP at /a2a
```

### Example: Spec-compliant AgentCard capabilities

```go
card := schema.AgentCard{Name: "example-a2a-server"}
streaming, push, sth := true, true, false
card.SetCapabilities(schema.AgentCapabilities{
    Streaming:             &streaming,
    PushNotifications:     &push,
    StateTransitionHistory: &sth,
})
```

## Client Usage

### SSE client

```go
stream, _ := client.NewStreamClient(ctx, "http://localhost:8080/v1/message:stream", headers, handler)
task, _ := stream.StreamMessage(ctx, msgs, nil, nil)
```

### Streamable HTTP client

```go
// Explicit streamable
stream, _ := client.NewStreamClientStreamable(ctx, "http://localhost:8080/a2a", headers, handler)
// Or autodetect (tries Streamable, then falls back to SSE)
stream, _ = client.AutoStreamClient(ctx, "http://localhost:8080/a2a", headers, handler)
task, _ := stream.StreamMessage(ctx, msgs, nil, nil)
```

### Agent Card

The canonical discovery endpoint is `/.well-known/agent-card.json`. The helper `/v1/card` has been removed to align with the spec.

## Migration

See MIGRATION.md for details on moving from the legacy `capabilities: []string` to the spec-compliant `capabilities` object and how the server maintains backward compatibility.

## Contributing

- Issues: Use GitHub Issues to report bugs and request features.
- Pull Requests: Keep changes focused. Include tests where it makes sense.
- Style: Follow existing patterns; avoid broad refactors in unrelated areas.

## Development

- Run tests: `go test ./...`
- Example server: `go run ./cmd/a2a-server`

## License

Licensed under the Apache License, Version 2.0. See `LICENSE` for details.
