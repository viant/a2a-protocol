package server

import (
    "context"
    "encoding/json"
    "net/http/httptest"
    "testing"

    "github.com/viant/a2a-protocol/schema"
    "github.com/viant/jsonrpc"
)

func TestStreamingGating(t *testing.T) {
    // card with streaming=false
    card := schema.AgentCard{Name: "test"}
    sFalse, pFalse := false, false
    card.SetCapabilities(schema.AgentCapabilities{Streaming: &sFalse, PushNotifications: &pFalse})
    srv := New(card)

    // rpcResubscribe should return error -32002
    rr := httptest.NewRecorder()
    params := json.RawMessage(`{"id":"t1"}`)
    srv.rpcResubscribe(rr, rpcRequest{JSONRPC: "2.0", ID: []byte("1"), Method: "tasks/resubscribe", Params: &params})
    var resp rpcResponse
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("decode response: %v", err)
    }
    if resp.Error == nil || resp.Error.Code != -32002 {
        t.Fatalf("expected streaming not supported (-32002), got: %+v", resp.Error)
    }

    // opsImpl.MessageStream should also reject when disabled
    ops := NewOperations(srv, nil)
    req := &jsonrpc.Request{Method: "message/stream", Params: json.RawMessage(`{"messages":[{"role":"user","parts":[{"type":"text","text":"hi"}]}]}`)}
    var r jsonrpc.Response
    ops.MessageStream(context.Background(), req, &r)
    if r.Error == nil || r.Error.Code != -32002 {
        t.Fatalf("expected streaming not supported (-32002) via ops, got: %+v", r.Error)
    }

    // Enable streaming and ensure success path (no error)
    sTrue := true
    card2 := schema.AgentCard{Name: "test2"}
    card2.SetCapabilities(schema.AgentCapabilities{Streaming: &sTrue})
    srv2 := New(card2)
    ops2 := NewOperations(srv2, nil)
    req2 := &jsonrpc.Request{Method: "message/stream", Params: json.RawMessage(`{"messages":[{"role":"user","parts":[{"type":"text","text":"hi"}]}]}`)}
    var r2 jsonrpc.Response
    ops2.MessageStream(context.Background(), req2, &r2)
    if r2.Error != nil {
        t.Fatalf("unexpected error with streaming enabled: %+v", r2.Error)
    }
}

func TestPushNotificationGating(t *testing.T) {
    card := schema.AgentCard{Name: "test"}
    sFalse, pFalse := false, false
    card.SetCapabilities(schema.AgentCapabilities{Streaming: &sFalse, PushNotifications: &pFalse})
    srv := New(card)

    // rpcPushConfigSet should return -32003 when push not supported
    rr := httptest.NewRecorder()
    params := json.RawMessage(`{"taskId":"t1","config":{"id":"c1","url":"https://example"}}`)
    srv.rpcPushConfigSet(rr, rpcRequest{JSONRPC: "2.0", ID: []byte("1"), Method: "tasks/pushNotificationConfig/set", Params: &params})
    var resp rpcResponse
    if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
        t.Fatalf("decode response: %v", err)
    }
    if resp.Error == nil || resp.Error.Code != -32003 {
        t.Fatalf("expected push not supported (-32003), got: %+v", resp.Error)
    }
}

