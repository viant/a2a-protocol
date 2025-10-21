package server

import (
	"context"
	"github.com/viant/a2a-protocol/schema"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
	"time"
)

// Option configures DefaultHandler.
type Option func(h *DefaultHandler) error

// DefaultHandler is a convenience wrapper around DefaultOperations with
// simple registration helpers for common A2A server behaviors.
type DefaultHandler struct {
	*DefaultOperations
}

// NewDefaultHandler creates a DefaultHandler.
func NewDefaultHandler(srv *Server, tr transport.Transport) *DefaultHandler {
	return &DefaultHandler{DefaultOperations: NewDefaultOperations(srv, tr).(*DefaultOperations)}
}

// WithDefaultHandler returns an Operations factory that builds a DefaultHandler
// and applies provided options, similar to viant/mcp-protocol's pattern.
func WithDefaultHandler(ctx context.Context, options ...Option) NewOperationsFunc {
	return func(srv *Server, tr transport.Transport) Operations {
		implementer := NewDefaultHandler(srv, tr)
		for _, opt := range options {
			_ = opt(implementer)
		}
		return implementer
	}
}

// RegisterMessageSend registers a simple handler for message/send.
func RegisterMessageSend(fn func(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error)) Option {
	return func(h *DefaultHandler) error {
		h.OnMessageSend = fn
		return nil
	}
}

// RegisterMessageStream registers a simple handler for message/stream.
func RegisterMessageStream(fn func(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error)) Option {
	return func(h *DefaultHandler) error {
		h.OnMessageStream = fn
		return nil
	}
}

// Helper: NewTask creates a new task with the given context ID.
func (h *DefaultHandler) NewTask(contextID *string) *schema.Task {
	return h.DefaultOperations.srv.tasks.newTask(contextID)
}

// Helper: CompleteText sets a single text artifact and marks task completed.
func (h *DefaultHandler) CompleteText(task *schema.Task, text string) {
	art := schema.Artifact{ID: "a-" + task.ID, CreatedAt: time.Now().UTC(), Parts: []schema.Part{schema.TextPart{Type: "text", Text: text}}}
	art.PartsRaw, _ = schema.MarshalParts(art.Parts)
	task.Status = schema.TaskStatus{State: schema.TaskCompleted, UpdatedAt: time.Now().UTC()}
	task.Artifacts = []schema.Artifact{art}
	h.DefaultOperations.srv.tasks.put(task)
}

// Helper: StreamDemo streams a basic running→artifact→completed sequence.
func (h *DefaultHandler) StreamDemo(task *schema.Task) {
	task.Touch(schema.TaskRunning)
	_ = h.DefaultOperations.sendStatus(context.Background(), task, false)
	art := schema.Artifact{ID: "a-" + task.ID, CreatedAt: time.Now().UTC(), Parts: []schema.Part{schema.TextPart{Type: "text", Text: "processing..."}}}
	art.PartsRaw, _ = schema.MarshalParts(art.Parts)
	_ = h.DefaultOperations.sendArtifact(context.Background(), task, art, true, false)
	task.Touch(schema.TaskCompleted)
	task.Artifacts = append(task.Artifacts, art)
	h.DefaultOperations.srv.tasks.put(task)
	_ = h.DefaultOperations.sendStatus(context.Background(), task, true)
}
