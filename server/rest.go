package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/viant/a2a-protocol/schema"
)

func (s *Server) handleSendMessageREST(w http.ResponseWriter, r *http.Request) {
	// Delegate to rpcSendMessage-style handler using same payload
	var req rpcRequest
	req.ID = []byte("null")
	raw, _ := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
	}{JSONRPC: "2.0"})
	_ = raw // unused, just to remind JSON-RPC shape
	req.Method = "message/send"
	var params json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Params = &params
	s.rpcSendMessage(w, req)
}

func (s *Server) handleGetTaskREST(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	var params json.RawMessage
	_ = json.Unmarshal([]byte(`{"id":"`+id+`"}`), &params)
	s.rpcGetTask(w, rpcRequest{JSONRPC: "2.0", ID: []byte("null"), Method: "tasks/get", Params: &params})
}

func (s *Server) handleTaskActionREST(w http.ResponseWriter, r *http.Request) {
	// Expect /v1/tasks/{id}:cancel
	path := strings.TrimPrefix(r.URL.Path, "/v1/tasks:")
	// Fallback: treat entire suffix as action on id
	// This is a stub mapping for cancel only
	if strings.HasSuffix(r.URL.Path, ":cancel") {
		// Extract id between /v1/tasks/ and :cancel
		// Example: /v1/tasks/123:cancel -> id=123
		// Try common pattern
		full := r.URL.Path
		start := strings.Index(full, "/v1/tasks/")
		id := ""
		if start >= 0 {
			rest := full[start+len("/v1/tasks/"):]
			if idx := strings.Index(rest, ":cancel"); idx >= 0 {
				id = rest[:idx]
			}
		}
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var params json.RawMessage
		_ = json.Unmarshal([]byte(`{"id":"`+id+`"}`), &params)
		s.rpcCancelTask(w, rpcRequest{JSONRPC: "2.0", ID: []byte("null"), Method: "tasks/cancel", Params: &params})
		return
	}
	http.NotFound(w, r)
	_ = path
}

func (s *Server) handleTaskSubscribeREST(w http.ResponseWriter, r *http.Request) {
	// Extract id before :subscribe
	full := r.URL.Path
	start := strings.Index(full, "/v1/tasks/")
	id := ""
	if start >= 0 {
		rest := full[start+len("/v1/tasks/"):]
		if idx := strings.Index(rest, ":subscribe"); idx >= 0 {
			id = rest[:idx]
		}
	}
	if id == "" {
		http.NotFound(w, r)
		return
	}
	var params json.RawMessage
	_ = json.Unmarshal([]byte(`{"id":"`+id+`"}`), &params)
	s.rpcResubscribe(w, rpcRequest{JSONRPC: "2.0", ID: []byte("null"), Method: "tasks/resubscribe", Params: &params})
}

// List tasks: GET /v1/tasks
func (s *Server) handleListTasksREST(w http.ResponseWriter, r *http.Request) {
	tasks := s.tasks.listTasks()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tasks)
}

// Push notifications CRUD
// POST /v1/tasks/{id}/pushNotificationConfigs
func (s *Server) handleCreatePushConfigREST(w http.ResponseWriter, r *http.Request) {
    if !s.card.PushNotificationsSupported() {
        http.Error(w, "Push Notification is not supported", http.StatusNotImplemented)
        return
    }
	// Extract task id
	full := r.URL.Path
	prefix := "/v1/tasks/"
	pos := strings.Index(full, prefix)
	if pos < 0 {
		http.NotFound(w, r)
		return
	}
	rest := full[pos+len(prefix):]
	idEnd := strings.Index(rest, "/pushNotificationConfigs")
	if idEnd < 0 {
		http.NotFound(w, r)
		return
	}
	taskID := rest[:idEnd]
	var cfg schema.PushNotificationConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	created := s.tasks.addPush(taskID, &cfg)
	if created == nil {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(created)
}

// GET /v1/tasks/{id}/pushNotificationConfigs/{configId}
func (s *Server) handleGetPushConfigREST(w http.ResponseWriter, r *http.Request) {
    if !s.card.PushNotificationsSupported() {
        http.Error(w, "Push Notification is not supported", http.StatusNotImplemented)
        return
    }
	taskID, cfgID := extractTaskAndConfigID(r.URL.Path)
	if taskID == "" || cfgID == "" {
		http.NotFound(w, r)
		return
	}
	cfg, ok := s.tasks.getPush(taskID, cfgID)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfg)
}

// GET /v1/tasks/{id}/pushNotificationConfigs
func (s *Server) handleListPushConfigsREST(w http.ResponseWriter, r *http.Request) {
    if !s.card.PushNotificationsSupported() {
        http.Error(w, "Push Notification is not supported", http.StatusNotImplemented)
        return
    }
	full := r.URL.Path
	prefix := "/v1/tasks/"
	pos := strings.Index(full, prefix)
	if pos < 0 {
		http.NotFound(w, r)
		return
	}
	rest := full[pos+len(prefix):]
	idEnd := strings.Index(rest, "/pushNotificationConfigs")
	if idEnd < 0 {
		http.NotFound(w, r)
		return
	}
	taskID := rest[:idEnd]
	cfgs, ok := s.tasks.listPush(taskID)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cfgs)
}

// DELETE /v1/tasks/{id}/pushNotificationConfigs/{configId}
func (s *Server) handleDeletePushConfigREST(w http.ResponseWriter, r *http.Request) {
    if !s.card.PushNotificationsSupported() {
        http.Error(w, "Push Notification is not supported", http.StatusNotImplemented)
        return
    }
	taskID, cfgID := extractTaskAndConfigID(r.URL.Path)
	if taskID == "" || cfgID == "" {
		http.NotFound(w, r)
		return
	}
	if !s.tasks.deletePush(taskID, cfgID) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func extractTaskAndConfigID(path string) (taskID, cfgID string) {
	// Expect /v1/tasks/{taskId}/pushNotificationConfigs/{configId}
	const prefix = "/v1/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}
	rest := path[len(prefix):]
	idx := strings.Index(rest, "/pushNotificationConfigs/")
	if idx < 0 {
		return "", ""
	}
	taskID = rest[:idx]
	cfgID = rest[idx+len("/pushNotificationConfigs/"):]
	return
}
