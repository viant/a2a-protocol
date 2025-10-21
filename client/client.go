package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/viant/a2a-protocol/schema"
)

// Client provides minimal A2A JSON-RPC calls.
type Client struct {
	Endpoint string
	HTTP     *http.Client
	// Optional auth headers to attach to each request
	Headers http.Header
}

func New(endpoint string) *Client {
	return &Client{Endpoint: endpoint, HTTP: http.DefaultClient, Headers: make(http.Header)}
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data,omitempty"`
	} `json:"error,omitempty"`
}

// SendMessage invokes message/send and returns a Task.
func (c *Client) SendMessage(ctx context.Context, messages []schema.Message, contextID *string) (*schema.Task, error) {
	payload := rpcRequest{JSONRPC: "2.0", ID: 1, Method: "message/send", Params: map[string]interface{}{
		"messages":  messages,
		"contextId": contextID,
	}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, vals := range c.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	var task schema.Task
	if err := json.Unmarshal(out.Result, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetTask calls tasks/get and returns a Task.
func (c *Client) GetTask(ctx context.Context, id string) (*schema.Task, error) {
	payload := rpcRequest{JSONRPC: "2.0", ID: 1, Method: "tasks/get", Params: map[string]interface{}{"id": id}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, vals := range c.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	var task schema.Task
	if err := json.Unmarshal(out.Result, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// CancelTask calls tasks/cancel and returns updated Task.
func (c *Client) CancelTask(ctx context.Context, id string) (*schema.Task, error) {
	payload := rpcRequest{JSONRPC: "2.0", ID: 1, Method: "tasks/cancel", Params: map[string]interface{}{"id": id}}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	for k, vals := range c.Headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", out.Error.Code, out.Error.Message)
	}
	var task schema.Task
	if err := json.Unmarshal(out.Result, &task); err != nil {
		return nil, err
	}
	return &task, nil
}
