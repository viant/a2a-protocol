package server

import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/viant/a2a-protocol/schema"
)

// local copy to decode REST JSON-RPC responses
type rpcResp struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      json.RawMessage `json:"id,omitempty"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *struct {
        Code    int             `json:"code"`
        Message string          `json:"message"`
        Data    json.RawMessage `json:"data,omitempty"`
    } `json:"error,omitempty"`
}

func newTestServer(streaming, push bool) (*Server, *http.ServeMux) {
    card := schema.AgentCard{Name: "test"}
    card.Endpoints = map[string]string{"rest": "/v1"}
    card.SetCapabilities(schema.AgentCapabilities{Streaming: &streaming, PushNotifications: &push})
    srv := New(card)
    mux := http.NewServeMux()
    srv.RegisterJSONRPC(mux, "/rpc")
    return srv, mux
}

func rpcCall(t *testing.T, ts *httptest.Server, method string, params interface{}) rpcResp {
    t.Helper()
    payload := map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": method}
    if params != nil {
        b, _ := json.Marshal(params)
        var raw json.RawMessage = b
        payload["params"] = &raw
    }
    b, _ := json.Marshal(payload)
    resp, err := http.Post(ts.URL+"/rpc", "application/json", bytes.NewReader(b))
    if err != nil {
        t.Fatalf("rpc post: %v", err)
    }
    defer resp.Body.Close()
    var out rpcResp
    body, _ := io.ReadAll(resp.Body)
    if err := json.Unmarshal(body, &out); err != nil {
        t.Fatalf("decode rpc: %v body=%s", err, string(body))
    }
    return out
}

func TestRPC_MessageSend_And_Get(t *testing.T) {
    _, mux := newTestServer(true, false)
    ts := httptest.NewServer(mux)
    defer ts.Close()

    // Send message via JSON-RPC
    rpc := rpcCall(t, ts, "message/send", map[string]interface{}{
        "messages": []map[string]interface{}{{"role": "user", "parts": []map[string]interface{}{{"type": "text", "text": "hi"}}}},
    })
    if rpc.Error != nil {
        t.Fatalf("unexpected error: %+v", rpc.Error)
    }
    // Extract task id
    var task schema.Task
    if err := json.Unmarshal(rpc.Result, &task); err != nil {
        t.Fatalf("decode task: %v", err)
    }
    if task.Status.State != schema.TaskCompleted {
        t.Fatalf("state=%s want completed", task.Status.State)
    }

    // Get task via JSON-RPC
    rpc2 := rpcCall(t, ts, "tasks/get", map[string]string{"id": task.ID})
    if rpc2.Error != nil {
        t.Fatalf("unexpected error on get: %+v", rpc2.Error)
    }
}

// Note: resubscribe is validated in gating_test via direct call; JSON-RPC handler
// does not expose tasks/resubscribe in this minimal mapping.

func TestRPC_PushEndpoints(t *testing.T) {
    // streaming true, push true
    _, mux := newTestServer(true, true)
    ts := httptest.NewServer(mux)
    defer ts.Close()

    // Create a task first
    rpc := rpcCall(t, ts, "message/send", map[string]interface{}{
        "messages": []map[string]interface{}{{"role": "user", "parts": []map[string]interface{}{{"type": "text", "text": "hi"}}}},
    })
    var task schema.Task
    _ = json.Unmarshal(rpc.Result, &task)

    // Set config
    rpc2 := rpcCall(t, ts, "tasks/pushNotificationConfig/set", map[string]interface{}{
        "taskId": task.ID,
        "config": map[string]interface{}{"id": "c1", "url": "https://example.com/webhook"},
    })
    if rpc2.Error != nil { t.Fatalf("set error: %+v", rpc2.Error) }

    // List configs
    rpc3 := rpcCall(t, ts, "tasks/pushNotificationConfig/list", map[string]interface{}{"taskId": task.ID})
    if rpc3.Error != nil { t.Fatalf("list error: %+v", rpc3.Error) }

    // Get single
    rpc4 := rpcCall(t, ts, "tasks/pushNotificationConfig/get", map[string]interface{}{"taskId": task.ID, "configId": "c1"})
    if rpc4.Error != nil { t.Fatalf("get error: %+v", rpc4.Error) }

    // Delete
    rpc5 := rpcCall(t, ts, "tasks/pushNotificationConfig/delete", map[string]interface{}{"taskId": task.ID, "configId": "c1"})
    if rpc5.Error != nil { t.Fatalf("delete error: %+v", rpc5.Error) }
}

func TestRPC_Card_Capabilities_Object(t *testing.T) {
    _, mux := newTestServer(true, false)
    ts := httptest.NewServer(mux)
    defer ts.Close()

    rpc := rpcCall(t, ts, "agent/getAuthenticatedExtendedCard", nil)
    if rpc.Error != nil { t.Fatalf("card error: %+v", rpc.Error) }
    var obj map[string]json.RawMessage
    if err := json.Unmarshal(rpc.Result, &obj); err != nil {
        t.Fatalf("decode card: %v", err)
    }
    capRaw := obj["capabilities"]
    if len(capRaw) == 0 || capRaw[0] != '{' {
        t.Fatalf("capabilities shape = %s, want object", string(capRaw))
    }
}
