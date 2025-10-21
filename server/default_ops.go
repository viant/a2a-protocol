package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/viant/a2a-protocol/schema"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
)

// DefaultOperations provides a thin, easy-to-implement layer for A2A servers.
// Implementers can supply simple callbacks without handling JSON-RPC directly.
type DefaultOperations struct {
	srv *Server
	tr  transport.Transport

	// Callbacks (optional). If nil, fall back to default demo behavior.
	OnMessageSend   func(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error)
	OnMessageStream func(ctx context.Context, messages []schema.Message, contextID, taskID *string) (*schema.Task, *jsonrpc.Error)
}

func NewDefaultOperations(srv *Server, tr transport.Transport, opts ...func(*DefaultOperations)) Operations {
	d := &DefaultOperations{srv: srv, tr: tr}
	for _, o := range opts {
		o(d)
	}
	return d
}

func (d *DefaultOperations) Implements(method string) bool                             { return true }
func (d *DefaultOperations) OnNotification(_ context.Context, _ *jsonrpc.Notification) {}

func (d *DefaultOperations) AgentGetCard(_ context.Context, _ *jsonrpc.Request, resp *jsonrpc.Response) {
	raw, _ := json.Marshal(d.srv.card)
	resp.Result = raw
}

func (d *DefaultOperations) pushSupported() bool {
    return d.srv.card.PushNotificationsSupported()
}

func (d *DefaultOperations) TasksPushNotificationConfigSet(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
	if !d.pushSupported() {
		resp.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
		return
	}
	var p struct {
		TaskID string                        `json:"taskId"`
		Config schema.PushNotificationConfig `json:"config"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.TaskID == "" {
		resp.Error = jsonrpc.NewInvalidParamsError("taskId and config required", req.Params)
		return
	}
	if cfg := d.srv.tasks.addPush(p.TaskID, &p.Config); cfg != nil {
		resp.Result, _ = json.Marshal(cfg)
		return
	}
	resp.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (d *DefaultOperations) TasksPushNotificationConfigGet(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
    if !d.pushSupported() {
        resp.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
        return
    }
    var p struct {
        TaskID  string `json:"taskId"`
        ConfigID string `json:"configId"`
    }
    if err := json.Unmarshal(req.Params, &p); err != nil || p.TaskID == "" || p.ConfigID == "" {
        resp.Error = jsonrpc.NewInvalidParamsError("taskId and configId required", req.Params)
        return
    }
	if cfg, ok := d.srv.tasks.getPush(p.TaskID, p.ConfigID); ok {
		resp.Result, _ = json.Marshal(cfg)
		return
	}
	resp.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (d *DefaultOperations) TasksPushNotificationConfigList(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
	if !d.pushSupported() {
		resp.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
		return
	}
	var p struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.TaskID == "" {
		resp.Error = jsonrpc.NewInvalidParamsError("taskId required", req.Params)
		return
	}
	if cfgs, ok := d.srv.tasks.listPush(p.TaskID); ok {
		resp.Result, _ = json.Marshal(cfgs)
		return
	}
	resp.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (d *DefaultOperations) TasksPushNotificationConfigDelete(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
    if !d.pushSupported() {
        resp.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
        return
    }
    var p struct {
        TaskID  string `json:"taskId"`
        ConfigID string `json:"configId"`
    }
    if err := json.Unmarshal(req.Params, &p); err != nil || p.TaskID == "" || p.ConfigID == "" {
        resp.Error = jsonrpc.NewInvalidParamsError("taskId and configId required", req.Params)
        return
    }
	if ok := d.srv.tasks.deletePush(p.TaskID, p.ConfigID); ok {
		resp.Result = []byte(`{"deleted":true}`)
		return
	}
	resp.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (d *DefaultOperations) MessageSend(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
	var p struct {
		ContextID *string          `json:"contextId,omitempty"`
		TaskID    *string          `json:"taskId,omitempty"`
		Messages  []schema.Message `json:"messages"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || len(p.Messages) == 0 {
		resp.Error = jsonrpc.NewInvalidParamsError("messages required", req.Params)
		return
	}
	if d.OnMessageSend != nil {
		if task, jerr := d.OnMessageSend(ctx, p.Messages, p.ContextID, p.TaskID); jerr != nil {
			resp.Error = jerr
		} else {
			resp.Result, _ = json.Marshal(task)
		}
		return
	}
	// default demo behavior
	// guard if continuing terminal taskId
	if p.TaskID != nil && *p.TaskID != "" {
		if existing, ok := d.srv.tasks.get(*p.TaskID); ok {
			if isTerminal(existing.Status.State) {
				resp.Error = jsonrpc.NewError(-32006, "task is in terminal state", nil)
				return
			}
		}
	}
	task := d.srv.tasks.newTask(p.ContextID)
	artifact := schema.Artifact{ID: "a-" + task.ID, CreatedAt: time.Now().UTC(), Parts: []schema.Part{schema.TextPart{Type: "text", Text: "ok"}}}
	artifact.PartsRaw, _ = schema.MarshalParts(artifact.Parts)
	task.Status = schema.TaskStatus{State: schema.TaskCompleted, UpdatedAt: time.Now().UTC()}
	task.Artifacts = []schema.Artifact{artifact}
	d.srv.tasks.put(task)
	resp.Result, _ = json.Marshal(task)
}

func (d *DefaultOperations) MessageStream(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
    if !d.srv.card.StreamingSupported() {
        resp.Error = jsonrpc.NewError(-32002, "Streaming is not supported", nil)
        return
    }
    var p struct {
        ContextID *string          `json:"contextId,omitempty"`
        TaskID    *string          `json:"taskId,omitempty"`
        Messages  []schema.Message `json:"messages"`
    }
	if err := json.Unmarshal(req.Params, &p); err != nil || len(p.Messages) == 0 {
		resp.Error = jsonrpc.NewInvalidParamsError("messages required", req.Params)
		return
	}
	if d.OnMessageStream != nil {
		if task, jerr := d.OnMessageStream(ctx, p.Messages, p.ContextID, p.TaskID); jerr != nil {
			resp.Error = jerr
		} else {
			resp.Result, _ = json.Marshal(task)
		}
		return
	}
	if p.TaskID != nil && *p.TaskID != "" {
		if existing, ok := d.srv.tasks.get(*p.TaskID); ok {
			if isTerminal(existing.Status.State) {
				resp.Error = jsonrpc.NewError(-32006, "task is in terminal state", nil)
				return
			}
		}
	}
	task := d.srv.tasks.newTask(p.ContextID)
	resp.Result, _ = json.Marshal(task)
	go d.streamDemo(ctx, task)
}

func (d *DefaultOperations) TasksGet(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" {
		resp.Error = jsonrpc.NewInvalidParamsError("id required", req.Params)
		return
	}
	if task, ok := d.srv.tasks.get(p.ID); ok {
		resp.Result, _ = json.Marshal(task)
		return
	}
	resp.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (d *DefaultOperations) TasksCancel(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" {
		resp.Error = jsonrpc.NewInvalidParamsError("id required", req.Params)
		return
	}
	if task, ok := d.srv.tasks.get(p.ID); ok {
		task.Status.State = schema.TaskCanceled
		task.Status.UpdatedAt = time.Now().UTC()
		d.srv.tasks.put(task)
		resp.Result, _ = json.Marshal(task)
		return
	}
	resp.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (d *DefaultOperations) TasksResubscribe(_ context.Context, req *jsonrpc.Request, resp *jsonrpc.Response) {
    if !d.srv.card.StreamingSupported() {
        resp.Error = jsonrpc.NewError(-32002, "Streaming is not supported", nil)
        return
    }
    var p struct {
        ID string `json:"id"`
    }
	var out interface{} = map[string]string{"status": "resubscribed"}
	if err := json.Unmarshal(req.Params, &p); err == nil && p.ID != "" {
		if t, ok := d.srv.tasks.get(p.ID); ok {
			out = t
		}
	}
	resp.Result, _ = json.Marshal(out)
}

// helpers
func (d *DefaultOperations) streamDemo(ctx context.Context, task *schema.Task) {
	task.Touch(schema.TaskRunning)
	_ = d.sendStatus(ctx, task, false)
	art := schema.Artifact{ID: "a-" + task.ID, CreatedAt: time.Now().UTC(), Parts: []schema.Part{schema.TextPart{Type: "text", Text: "processing..."}}}
	art.PartsRaw, _ = schema.MarshalParts(art.Parts)
	_ = d.sendArtifact(ctx, task, art, true, false)
	task.Touch(schema.TaskCompleted)
	task.Artifacts = append(task.Artifacts, art)
	d.srv.tasks.put(task)
	_ = d.sendStatus(ctx, task, true)
}

func (d *DefaultOperations) sendStatus(ctx context.Context, task *schema.Task, final bool) error {
	evt := schema.NewStatusEvent(task, final)
	return sendSSEResponse(ctx, evt)
}

func (d *DefaultOperations) sendArtifact(ctx context.Context, task *schema.Task, artifact schema.Artifact, append, last bool) error {
	evt := schema.NewArtifactEvent(task, artifact, append, last)
	return sendSSEResponse(ctx, evt)
}
