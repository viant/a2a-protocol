package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/viant/a2a-protocol/schema"
	"strings"
)

// JSON-RPC 2.0 structures
type rpcRequest struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      json.RawMessage  `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  *json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Server implements A2A entry points.
type Server struct {
	tasks      *taskStore
	card       schema.AgentCard
	opsFactory NewOperationsFunc
}

// New creates a Server with an in-memory task store.
func New(card schema.AgentCard, opts ...ServerOption) *Server {
	s := &Server{tasks: newTaskStore(), card: card}
	for _, o := range opts {
		o(s)
	}
	return s
}

// RegisterJSONRPC registers a JSON-RPC handler on the given mux and path.
func (s *Server) RegisterJSONRPC(mux *http.ServeMux, path string) {
	mux.HandleFunc(path, s.handleJSONRPC)
}

// RegisterREST registers minimal REST handlers per mapping table.
func (s *Server) RegisterREST(mux *http.ServeMux) {
    // POST /v1/message:send
    mux.HandleFunc("/v1/message:send", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.NotFound(w, r)
            return
        }
        s.handleSendMessageREST(w, r)
    })
    // Consolidated handler for /v1/tasks/* routes to avoid conflicting mux patterns
    mux.HandleFunc("/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
        path := r.URL.Path
        // Push notification subroutes
        if strings.Contains(path, "/pushNotificationConfigs/") {
            switch r.Method {
            case http.MethodGet:
                s.handleGetPushConfigREST(w, r)
                return
            case http.MethodDelete:
                s.handleDeletePushConfigREST(w, r)
                return
            default:
                http.NotFound(w, r)
                return
            }
        }
        if strings.HasSuffix(path, "/pushNotificationConfigs") {
            switch r.Method {
            case http.MethodGet:
                s.handleListPushConfigsREST(w, r)
                return
            case http.MethodPost:
                s.handleCreatePushConfigREST(w, r)
                return
            default:
                http.NotFound(w, r)
                return
            }
        }
        // Subscribe: POST /v1/tasks/{id}:subscribe
        if strings.HasSuffix(path, ":subscribe") {
            if r.Method != http.MethodPost {
                http.NotFound(w, r)
                return
            }
            s.handleTaskSubscribeREST(w, r)
            return
        }
        // Cancel: POST /v1/tasks/{id}:cancel
        if strings.HasSuffix(path, ":cancel") {
            if r.Method != http.MethodPost {
                http.NotFound(w, r)
                return
            }
            s.handleTaskActionREST(w, r)
            return
        }
        // Get task: GET /v1/tasks/{id}
        if r.Method == http.MethodGet {
            s.handleGetTaskREST(w, r)
            return
        }
        http.NotFound(w, r)
    })
	// GET /v1/tasks
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		s.handleListTasksREST(w, r)
	})
}

func (s *Server) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, req.ID, -32700, "parse error", err)
		return
	}
	switch req.Method {
	case "message/send":
		s.rpcSendMessage(w, req)
	case "tasks/get":
		s.rpcGetTask(w, req)
	case "tasks/cancel":
		s.rpcCancelTask(w, req)
	case "tasks/pushNotificationConfig/set":
		s.rpcPushConfigSet(w, req)
	case "tasks/pushNotificationConfig/get":
		s.rpcPushConfigGet(w, req)
	case "tasks/pushNotificationConfig/list":
		s.rpcPushConfigList(w, req)
	case "tasks/pushNotificationConfig/delete":
		s.rpcPushConfigDelete(w, req)
	case "agent/getAuthenticatedExtendedCard":
		writeRPCResult(w, req.ID, s.card)
	default:
		writeRPCError(w, req.ID, -32601, "method not found", nil)
	}
}

// rpcSendMessage handles message/send and returns a Task.
func (s *Server) rpcSendMessage(w http.ResponseWriter, req rpcRequest) {
	type params struct {
		ContextID *string          `json:"contextId,omitempty"`
		Messages  []schema.Message `json:"messages"`
	}
	var p params
	if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || len(p.Messages) == 0 {
		writeRPCError(w, req.ID, -32602, "invalid params", errors.New("messages required"))
		return
	}
	task := s.tasks.newTask(p.ContextID)
	// Stub: immediately mark as completed with an echo artifact
	artifact := schema.Artifact{
		ID:        "a-" + task.ID,
		CreatedAt: time.Now().UTC(),
		Parts:     []schema.Part{schema.TextPart{Type: "text", Text: "ok"}},
	}
	artifact.PartsRaw, _ = schema.MarshalParts(artifact.Parts)
	task.Status = schema.TaskStatus{State: schema.TaskCompleted, UpdatedAt: time.Now().UTC()}
	task.Artifacts = []schema.Artifact{artifact}
	s.tasks.put(task)
	writeRPCResult(w, req.ID, task)
}

func (s *Server) rpcGetTask(w http.ResponseWriter, req rpcRequest) {
	type params struct {
		ID string `json:"id"`
	}
	var p params
	if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.ID == "" {
		writeRPCError(w, req.ID, -32602, "invalid params", errors.New("id required"))
		return
	}
	task, ok := s.tasks.get(p.ID)
	if !ok {
		writeRPCError(w, req.ID, -32004, "not found", nil)
		return
	}
	writeRPCResult(w, req.ID, task)
}

func (s *Server) rpcCancelTask(w http.ResponseWriter, req rpcRequest) {
	type params struct {
		ID string `json:"id"`
	}
	var p params
	if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.ID == "" {
		writeRPCError(w, req.ID, -32602, "invalid params", errors.New("id required"))
		return
	}
	if task, ok := s.tasks.get(p.ID); ok {
		task.Status.State = schema.TaskCanceled
		task.Status.UpdatedAt = time.Now().UTC()
		s.tasks.put(task)
		writeRPCResult(w, req.ID, task)
		return
	}
	writeRPCError(w, req.ID, -32004, "not found", nil)
}

func writeRPCResult(w http.ResponseWriter, id json.RawMessage, result interface{}) {
	_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
}

func writeRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string, err error) {
	e := &rpcError{Code: code, Message: message}
	if err != nil {
		e.Data = err.Error()
	}
	_ = json.NewEncoder(w).Encode(rpcResponse{JSONRPC: "2.0", ID: id, Error: e})
}

func (s *Server) rpcResubscribe(w http.ResponseWriter, req rpcRequest) {
    if !s.card.StreamingSupported() {
        writeRPCError(w, req.ID, -32002, "Streaming is not supported", nil)
        return
    }
    type params struct {
        ID string `json:"id"`
    }
	var p params
	if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.ID == "" {
		writeRPCError(w, req.ID, -32602, "invalid params", nil)
		return
	}
	if task, ok := s.tasks.get(p.ID); ok {
		writeRPCResult(w, req.ID, task)
		return
	}
	writeRPCError(w, req.ID, -32004, "not found", nil)
}

func (s *Server) pushSupported() bool {
    return s.card.PushNotificationsSupported()
}

func (s *Server) rpcPushConfigSet(w http.ResponseWriter, req rpcRequest) {
	if !s.pushSupported() {
		writeRPCError(w, req.ID, -32003, "Push Notification is not supported", nil)
		return
	}
	var p struct {
		TaskID string                        `json:"taskId"`
		Config schema.PushNotificationConfig `json:"config"`
	}
	if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.TaskID == "" {
		writeRPCError(w, req.ID, -32602, "invalid params", nil)
		return
	}
	if cfg := s.tasks.addPush(p.TaskID, &p.Config); cfg != nil {
		writeRPCResult(w, req.ID, cfg)
		return
	}
	writeRPCError(w, req.ID, -32004, "not found", nil)
}

func (s *Server) rpcPushConfigGet(w http.ResponseWriter, req rpcRequest) {
    if !s.pushSupported() {
        writeRPCError(w, req.ID, -32003, "Push Notification is not supported", nil)
        return
    }
    var p struct {
        TaskID  string `json:"taskId"`
        ConfigID string `json:"configId"`
    }
    if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.TaskID == "" || p.ConfigID == "" {
        writeRPCError(w, req.ID, -32602, "invalid params", nil)
        return
    }
	if cfg, ok := s.tasks.getPush(p.TaskID, p.ConfigID); ok {
		writeRPCResult(w, req.ID, cfg)
		return
	}
	writeRPCError(w, req.ID, -32004, "not found", nil)
}

func (s *Server) rpcPushConfigList(w http.ResponseWriter, req rpcRequest) {
	if !s.pushSupported() {
		writeRPCError(w, req.ID, -32003, "Push Notification is not supported", nil)
		return
	}
	var p struct {
		TaskID string `json:"taskId"`
	}
	if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.TaskID == "" {
		writeRPCError(w, req.ID, -32602, "invalid params", nil)
		return
	}
	if cfgs, ok := s.tasks.listPush(p.TaskID); ok {
		writeRPCResult(w, req.ID, cfgs)
		return
	}
	writeRPCError(w, req.ID, -32004, "not found", nil)
}

func (s *Server) rpcPushConfigDelete(w http.ResponseWriter, req rpcRequest) {
    if !s.pushSupported() {
        writeRPCError(w, req.ID, -32003, "Push Notification is not supported", nil)
        return
    }
    var p struct {
        TaskID  string `json:"taskId"`
        ConfigID string `json:"configId"`
    }
    if req.Params == nil || json.Unmarshal(*req.Params, &p) != nil || p.TaskID == "" || p.ConfigID == "" {
        writeRPCError(w, req.ID, -32602, "invalid params", nil)
        return
    }
	if ok := s.tasks.deletePush(p.TaskID, p.ConfigID); ok {
		writeRPCResult(w, req.ID, map[string]bool{"deleted": true})
		return
	}
	writeRPCError(w, req.ID, -32004, "not found", nil)
}
