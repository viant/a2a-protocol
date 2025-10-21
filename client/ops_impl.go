package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/viant/a2a-protocol/schema"
)

type ops struct {
	rpc      *Client
	sse      *A2AStreamClient
	restBase string
	headers  http.Header
	http     *http.Client
}

func (o *ops) MessageSend(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, error) {
	// Use rpc JSON-RPC client; extend with taskID by passing as part of messages data
	// If taskID is provided and RPC server supports it, you could add dedicated method; here we ignore.
	return o.rpc.SendMessage(ctx, messages, contextID)
}

func (o *ops) MessageStream(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, error) {
	if o.sse != nil {
		return o.sse.StreamMessage(ctx, messages, contextID, taskID)
	}
	// Fallback to non-streaming
	return o.MessageSend(ctx, messages, contextID, taskID)
}

func (o *ops) TasksGet(ctx context.Context, id string) (*schema.Task, error) {
	return o.rpc.GetTask(ctx, id)
}

func (o *ops) TasksCancel(ctx context.Context, id string) (*schema.Task, error) {
	return o.rpc.CancelTask(ctx, id)
}

func (o *ops) TasksResubscribe(ctx context.Context, id string) error {
	if o.sse == nil {
		return fmt.Errorf("streaming client not configured")
	}
	return o.sse.Resubscribe(ctx, id)
}

func (o *ops) TasksList(ctx context.Context) ([]*schema.Task, error) {
	if o.restBase == "" {
		return nil, fmt.Errorf("rest base not configured")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(o.restBase, "/")+"/tasks", nil)
	for k, vals := range o.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var out []*schema.Task
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (o *ops) AgentGetCard(ctx context.Context) (*schema.AgentCard, error) {
	// Prefer well-known
	url := "/.well-known/agent-card.json"
	// If restBase present and absolute, use it as base to construct URL
	if o.restBase != "" {
		if strings.HasPrefix(o.restBase, "http") {
			url = strings.TrimRight(o.restBase, "/") + "/../.well-known/agent-card.json"
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	for k, vals := range o.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := o.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	var card schema.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, err
	}
	return &card, nil
}
