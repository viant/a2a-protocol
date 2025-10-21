package server

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/viant/a2a-protocol/schema"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
)

// opsImpl is the default Operations implementation.
type opsImpl struct {
	srv *Server
	tr  transport.Transport
}

func NewOperations(srv *Server, tr transport.Transport) Operations {
	return &opsImpl{srv: srv, tr: tr}
}

func (o *opsImpl) Implements(method string) bool {
	switch method {
	case "message/send", "message/stream",
		"tasks/get", "tasks/cancel", "tasks/resubscribe",
		"agent/getAuthenticatedExtendedCard":
		return true
	}
	return false
}

func (o *opsImpl) OnNotification(_ context.Context, _ *jsonrpc.Notification) {}

// MessageSend creates/continues a task and may set auth-required state.
func (o *opsImpl) MessageSend(ctx context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	var p struct {
		ContextID *string          `json:"contextId,omitempty"`
		TaskID    *string          `json:"taskId,omitempty"`
		Messages  []schema.Message `json:"messages"`
	}
	if err := json.Unmarshal(request.Params, &p); err != nil || len(p.Messages) == 0 {
		response.Error = jsonrpc.NewInvalidParamsError("messages required", request.Params)
		return
	}
	var task *schema.Task
	if p.TaskID != nil && *p.TaskID != "" {
		if existing, ok := o.srv.tasks.get(*p.TaskID); ok {
			if isTerminal(existing.Status.State) {
				response.Error = jsonrpc.NewError(-32006, "task is in terminal state", nil)
				return
			}
			task = existing
		}
	}
	if task == nil {
		task = o.srv.tasks.newTask(p.ContextID)
	}

	authReq := detectSecondaryAuth(p.Messages)
	if authReq.Require && strings.TrimSpace(authReq.Token) == "" {
		task.Touch(schema.TaskAuthRequired)
		task.Status.Message = buildAuthMessage(authReq)
		o.srv.tasks.put(task)
		raw, _ := json.Marshal(task)
		response.Result = raw
		return
	}

	artifact := schema.Artifact{ID: "a-" + task.ID, CreatedAt: time.Now().UTC(), Parts: []schema.Part{schema.TextPart{Type: "text", Text: "ok"}}}
	artifact.PartsRaw, _ = schema.MarshalParts(artifact.Parts)
	task.Status = schema.TaskStatus{State: schema.TaskCompleted, UpdatedAt: time.Now().UTC()}
	task.Artifacts = []schema.Artifact{artifact}
	o.srv.tasks.put(task)
	raw, _ := json.Marshal(task)
	response.Result = raw
}

// MessageStream starts streaming updates and returns the task immediately.
func (o *opsImpl) MessageStream(ctx context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
    if !o.srv.card.StreamingSupported() {
        response.Error = jsonrpc.NewError(-32002, "Streaming is not supported", nil)
        return
    }
    var p struct {
        ContextID *string          `json:"contextId,omitempty"`
        TaskID    *string          `json:"taskId,omitempty"`
        Messages  []schema.Message `json:"messages"`
    }
	if err := json.Unmarshal(request.Params, &p); err != nil || len(p.Messages) == 0 {
		response.Error = jsonrpc.NewInvalidParamsError("messages required", request.Params)
		return
	}
	var task *schema.Task
	if p.TaskID != nil && *p.TaskID != "" {
		if existing, ok := o.srv.tasks.get(*p.TaskID); ok {
			if isTerminal(existing.Status.State) {
				response.Error = jsonrpc.NewError(-32006, "task is in terminal state", nil)
				return
			}
			task = existing
		}
	}
	if task == nil {
		task = o.srv.tasks.newTask(p.ContextID)
	}

	authReq := detectSecondaryAuth(p.Messages)
	if authReq.Require && strings.TrimSpace(authReq.Token) == "" {
		task.Touch(schema.TaskAuthRequired)
		task.Status.Message = buildAuthMessage(authReq)
		o.srv.tasks.put(task)
		go func() { _ = o.sendStatus(context.Background(), task, false) }()
		raw, _ := json.Marshal(task)
		response.Result = raw
		return
	}

	raw, _ := json.Marshal(task)
	response.Result = raw
	go o.streamDemo(ctx, task)
}

func (o *opsImpl) TasksGet(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request.Params, &p); err != nil || p.ID == "" {
		response.Error = jsonrpc.NewInvalidParamsError("id required", request.Params)
		return
	}
	if task, ok := o.srv.tasks.get(p.ID); ok {
		raw, _ := json.Marshal(task)
		response.Result = raw
		return
	}
	response.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (o *opsImpl) TasksCancel(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(request.Params, &p); err != nil || p.ID == "" {
		response.Error = jsonrpc.NewInvalidParamsError("id required", request.Params)
		return
	}
	if task, ok := o.srv.tasks.get(p.ID); ok {
		task.Status.State = schema.TaskCanceled
		task.Status.UpdatedAt = time.Now().UTC()
		o.srv.tasks.put(task)
		raw, _ := json.Marshal(task)
		response.Result = raw
		return
	}
	response.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (o *opsImpl) TasksResubscribe(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
    if !o.srv.card.StreamingSupported() {
        response.Error = jsonrpc.NewError(-32002, "Streaming is not supported", nil)
        return
    }
    var p struct {
        ID string `json:"id"`
    }
	var out interface{} = map[string]string{"status": "resubscribed"}
	if err := json.Unmarshal(request.Params, &p); err == nil && p.ID != "" {
		if t, ok := o.srv.tasks.get(p.ID); ok {
			out = t
		}
	}
	b, _ := json.Marshal(out)
	response.Result = b
}

func (o *opsImpl) AgentGetCard(_ context.Context, _ *jsonrpc.Request, response *jsonrpc.Response) {
	raw, _ := json.Marshal(o.srv.card)
	response.Result = raw
}

func (o *opsImpl) pushSupported() bool { return o.srv.card.PushNotificationsSupported() }

func (o *opsImpl) TasksPushNotificationConfigSet(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	if !o.pushSupported() {
		response.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
		return
	}
	var p struct {
		TaskID string                        `json:"taskId"`
		Config schema.PushNotificationConfig `json:"config"`
	}
	if err := json.Unmarshal(request.Params, &p); err != nil || p.TaskID == "" {
		response.Error = jsonrpc.NewInvalidParamsError("taskId and config required", request.Params)
		return
	}
	if cfg := o.srv.tasks.addPush(p.TaskID, &p.Config); cfg != nil {
		raw, _ := json.Marshal(cfg)
		response.Result = raw
		return
	}
	response.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (o *opsImpl) TasksPushNotificationConfigGet(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
    if !o.pushSupported() {
        response.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
        return
    }
    var p struct {
        TaskID  string `json:"taskId"`
        ConfigID string `json:"configId"`
    }
    if err := json.Unmarshal(request.Params, &p); err != nil || p.TaskID == "" || p.ConfigID == "" {
        response.Error = jsonrpc.NewInvalidParamsError("taskId and configId required", request.Params)
        return
    }
	if cfg, ok := o.srv.tasks.getPush(p.TaskID, p.ConfigID); ok {
		raw, _ := json.Marshal(cfg)
		response.Result = raw
		return
	}
	response.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (o *opsImpl) TasksPushNotificationConfigList(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	if !o.pushSupported() {
		response.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
		return
	}
	var p struct {
		TaskID string `json:"taskId"`
	}
	if err := json.Unmarshal(request.Params, &p); err != nil || p.TaskID == "" {
		response.Error = jsonrpc.NewInvalidParamsError("taskId required", request.Params)
		return
	}
	if cfgs, ok := o.srv.tasks.listPush(p.TaskID); ok {
		raw, _ := json.Marshal(cfgs)
		response.Result = raw
		return
	}
	response.Error = jsonrpc.NewError(-32004, "not found", nil)
}

func (o *opsImpl) TasksPushNotificationConfigDelete(_ context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
    if !o.pushSupported() {
        response.Error = jsonrpc.NewError(-32003, "Push Notification is not supported", nil)
        return
    }
    var p struct {
        TaskID  string `json:"taskId"`
        ConfigID string `json:"configId"`
    }
    if err := json.Unmarshal(request.Params, &p); err != nil || p.TaskID == "" || p.ConfigID == "" {
        response.Error = jsonrpc.NewInvalidParamsError("taskId and configId required", request.Params)
        return
    }
	if ok := o.srv.tasks.deletePush(p.TaskID, p.ConfigID); ok {
		response.Result = []byte(`{"deleted":true}`)
		return
	}
	response.Error = jsonrpc.NewError(-32004, "not found", nil)
}

// --- helpers (moved from previous handler) ---

func (o *opsImpl) streamDemo(ctx context.Context, task *schema.Task) {
	task.Touch(schema.TaskRunning)
	_ = o.sendStatus(ctx, task, false)
	art := schema.Artifact{ID: "a-" + task.ID, CreatedAt: time.Now().UTC(), Parts: []schema.Part{schema.TextPart{Type: "text", Text: "processing..."}}}
	art.PartsRaw, _ = schema.MarshalParts(art.Parts)
	_ = o.sendArtifact(ctx, task, art, true, false)
	task.Touch(schema.TaskCompleted)
	task.Artifacts = append(task.Artifacts, art)
	o.srv.tasks.put(task)
	_ = o.sendStatus(ctx, task, true)
}

func (o *opsImpl) sendStatus(ctx context.Context, task *schema.Task, final bool) error {
	evt := schema.NewStatusEvent(task, final)
	return sendSSEResponse(ctx, evt)
}

func (o *opsImpl) sendArtifact(ctx context.Context, task *schema.Task, artifact schema.Artifact, append, last bool) error {
	evt := schema.NewArtifactEvent(task, artifact, append, last)
	return sendSSEResponse(ctx, evt)
}

// secondary auth helpers (copied)
type secAuth struct {
	Require          bool
	Resource         string
	Scopes           []string
	AuthorizationURI string
	Token            string
}

func detectSecondaryAuth(messages []schema.Message) secAuth {
	var out secAuth
	for _, m := range messages {
		for _, raw := range m.PartsRaw {
			var probe map[string]interface{}
			if err := json.Unmarshal(raw, &probe); err != nil {
				continue
			}
			t, _ := probe["type"].(string)
			if strings.ToLower(t) != "data" {
				continue
			}
			data, _ := probe["data"].(map[string]interface{})
			if data == nil {
				continue
			}
			if v, ok := data["requireSecondaryAuth"].(bool); ok && v {
				out.Require = true
			}
			if v, ok := data["resource"].(string); ok {
				out.Resource = v
			}
			if v, ok := data["authorization_uri"].(string); ok {
				out.AuthorizationURI = v
			}
			if v, ok := data["secondaryAuthToken"].(string); ok {
				out.Token = v
			}
			if v, ok := data["scopes"].([]interface{}); ok {
				for _, s := range v {
					if sv, ok := s.(string); ok {
						out.Scopes = append(out.Scopes, sv)
					}
				}
			}
		}
	}
	return out
}

func buildAuthMessage(a secAuth) *schema.DataPart {
	payload := map[string]interface{}{
		"auth": map[string]interface{}{
			"resource": a.Resource,
			"scopes":   a.Scopes,
		},
	}
	if a.AuthorizationURI != "" {
		payload["auth"].(map[string]interface{})["authorization_uri"] = a.AuthorizationURI
	}
	return &schema.DataPart{Type: "data", Data: payload}
}
