package client

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/viant/a2a-protocol/schema"
    "github.com/viant/jsonrpc"
    ssecli "github.com/viant/jsonrpc/transport/client/http/sse"
    streamcli "github.com/viant/jsonrpc/transport/client/http/streamable"
)

// UpdateHandler handles streaming update events.
type UpdateHandler interface {
    OnStatusUpdate(e *schema.TaskStatusUpdateEvent)
    OnArtifactUpdate(e *schema.TaskArtifactUpdateEvent)
}

// Operation represents a client-side operation context that can receive
// streaming updates.
type Operation interface {
    UpdateHandler
}

// A2AStreamClient wraps a JSON-RPC SSE client for A2A.
type requester interface {
    Send(ctx context.Context, r *jsonrpc.Request) (*jsonrpc.Response, error)
}

// A2AStreamClient wraps a JSON-RPC streaming-capable client (SSE or Streamable HTTP).
type A2AStreamClient struct {
    rpc requester
}

// NewStreamClient connects to an SSE endpoint and returns a client.
// headers will be attached to both the SSE handshake and message POSTs (e.g., Authorization).
// handler receives incoming streaming events.
func NewStreamClient(ctx context.Context, sseURL string, headers http.Header, op Operation) (*A2AStreamClient, error) {
	// Inject headers via custom http.Client roundtripper
	rt := withHeaders(http.DefaultTransport, headers)
	hc := &http.Client{Transport: rt, Timeout: 0}
    h := &streamHandler{UpdateHandler: op}
    cli, err := ssecli.New(ctx, sseURL,
        ssecli.WithHttpClient(hc),
        ssecli.WithMessageHttpClient(hc),
        ssecli.WithHandler(h),
        ssecli.WithHandshakeTimeout(10*time.Second),
    )
    if err != nil {
        return nil, err
    }
    return &A2AStreamClient{rpc: cli}, nil
}

// NewStreamClientStreamable connects to a Streamable HTTP endpoint (single endpoint).
func NewStreamClientStreamable(ctx context.Context, mcpURL string, headers http.Header, op Operation) (*A2AStreamClient, error) {
    rt := withHeaders(http.DefaultTransport, headers)
    hc := &http.Client{Transport: rt, Timeout: 0}
    h := &streamHandler{UpdateHandler: op}
    cli, err := streamcli.New(ctx, mcpURL,
        streamcli.WithHTTPClient(hc),
        streamcli.WithHandler(h),
    )
    if err != nil {
        return nil, err
    }
    return &A2AStreamClient{rpc: cli}, nil
}

// AutoStreamClient autodetects the streaming transport for the given streamURL.
// It attempts a JSON-RPC POST to detect a Streamable HTTP endpoint. If the server
// responds with a session header (Mcp-Session-Id), the Streamable client is used;
// otherwise it falls back to SSE.
func AutoStreamClient(ctx context.Context, streamURL string, headers http.Header, op Operation) (*A2AStreamClient, error) {
    if streamURL == "" {
        return nil, fmt.Errorf("empty streamURL")
    }
    rt := withHeaders(http.DefaultTransport, headers)
    hc := &http.Client{Transport: rt, Timeout: 0}

    // Minimal JSON-RPC request – method intentionally generic; server may return method not found.
    probe := &jsonrpc.Request{Jsonrpc: "2.0", Id: 1, Method: "ping"}
    data, _ := json.Marshal(probe)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, streamURL, bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json, text/event-stream")
    resp, err := hc.Do(req)
    if err == nil {
        _ = resp.Body.Close()
        if resp.Header.Get("Mcp-Session-Id") != "" {
            // Streamable endpoint detected
            return NewStreamClientStreamable(ctx, streamURL, headers, op)
        }
    }
    // Fallback to SSE
    return NewStreamClient(ctx, streamURL, headers, op)
}

// SendMessage sends a non-streaming message (method: message/send) using the SSE message endpoint.
func (c *A2AStreamClient) SendMessage(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, error) {
	params := map[string]interface{}{"messages": messages}
	if contextID != nil {
		params["contextId"] = contextID
	}
	if taskID != nil && *taskID != "" {
		params["taskId"] = *taskID
	}
	req, _ := jsonrpc.NewRequest("message/send", params)
    resp, err := c.rpc.Send(ctx, req)
	if err != nil {
		return nil, err
	}
	var task schema.Task
	if err := json.Unmarshal(resp.Result, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// StreamMessage starts a streaming interaction (method: message/stream) and returns the created/continued task.
func (c *A2AStreamClient) StreamMessage(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, error) {
	params := map[string]interface{}{"messages": messages}
	if contextID != nil {
		params["contextId"] = contextID
	}
	if taskID != nil && *taskID != "" {
		params["taskId"] = *taskID
	}
	req, _ := jsonrpc.NewRequest("message/stream", params)
    resp, err := c.rpc.Send(ctx, req)
	if err != nil {
		return nil, err
	}
	var task schema.Task
	if err := json.Unmarshal(resp.Result, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// Resubscribe requests server-side resume semantics for a task.
func (c *A2AStreamClient) Resubscribe(ctx context.Context, taskID string) error {
	req, _ := jsonrpc.NewRequest("tasks/resubscribe", map[string]string{"id": taskID})
    _, err := c.rpc.Send(ctx, req)
	return err
}

// streamHandler decodes notifications and forwards them.
type streamHandler struct{ UpdateHandler }

func (h *streamHandler) Serve(ctx context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	// Not used in this scenario; echo method-not-found to make it explicit
	response.Error = jsonrpc.NewMethodNotFound("method not supported on client", request.Params)
}

func (h *streamHandler) OnNotification(ctx context.Context, n *jsonrpc.Notification) {
	if h.UpdateHandler == nil || n == nil || len(n.Params) == 0 {
		return
	}
	// Try status-update
	var status schema.TaskStatusUpdateEvent
	if json.Unmarshal(n.Params, &status) == nil && status.Kind == "status-update" {
		h.UpdateHandler.OnStatusUpdate(&status)
		return
	}
	var art schema.TaskArtifactUpdateEvent
	if json.Unmarshal(n.Params, &art) == nil && art.Kind == "artifact-update" {
		h.UpdateHandler.OnArtifactUpdate(&art)
		return
	}
}

// withHeaders wraps a RoundTripper to attach default headers to each request.
func withHeaders(base http.RoundTripper, headers http.Header) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		for k, vals := range headers {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
		return base.RoundTrip(req)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
