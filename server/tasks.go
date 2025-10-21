package server

import (
	"sync"
	"time"

	"github.com/viant/a2a-protocol/schema"
)

type taskStore struct {
    mu    sync.RWMutex
    seq   int64
    items map[string]*schema.Task
    // push notification configs per task
    push map[string]map[string]*schema.PushNotificationConfig
    pseq int64
    // state transition history per task (internal stub)
    hist map[string][]schema.TaskStateTransition
}

func newTaskStore() *taskStore {
    return &taskStore{
        items: map[string]*schema.Task{},
        push:  map[string]map[string]*schema.PushNotificationConfig{},
        hist:  map[string][]schema.TaskStateTransition{},
    }
}

func (t *taskStore) newTask(contextID *string) *schema.Task {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.seq++
    id := formatID(t.seq)
    task := &schema.Task{
        ID:        id,
        ContextID: contextID,
        Status: schema.TaskStatus{
            State:     schema.TaskRunning,
            UpdatedAt: time.Now().UTC(),
        },
    }
    t.items[id] = task
    // record initial state
    t.hist[id] = append(t.hist[id], schema.TaskStateTransition{State: task.Status.State, At: task.Status.UpdatedAt})
    return task
}

func (t *taskStore) get(id string) (*schema.Task, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	v, ok := t.items[id]
	return v, ok
}

func (t *taskStore) put(task *schema.Task) {
    t.mu.Lock()
    defer t.mu.Unlock()
    if prev, ok := t.items[task.ID]; ok {
        if prev.Status.State != task.Status.State {
            at := task.Status.UpdatedAt
            if at.IsZero() {
                at = time.Now().UTC()
            }
            t.hist[task.ID] = append(t.hist[task.ID], schema.TaskStateTransition{State: task.Status.State, At: at})
        }
    } else {
        // first write, ensure history initialized
        t.hist[task.ID] = append(t.hist[task.ID], schema.TaskStateTransition{State: task.Status.State, At: task.Status.UpdatedAt})
    }
    t.items[task.ID] = task
}

func formatID(seq int64) string {
	return "t-" + itoa(seq)
}

func itoa(v int64) string {
	// small, dependency-free base-10
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// isTerminal returns true if the task state is terminal per spec.
func isTerminal(state schema.TaskState) bool {
	switch state {
	case schema.TaskCompleted, schema.TaskFailed, schema.TaskCanceled:
		return true
	}
	return false
}

// push notification config helpers
func (t *taskStore) addPush(taskID string, cfg *schema.PushNotificationConfig) *schema.PushNotificationConfig {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.items[taskID]; !ok {
		return nil
	}
	t.pseq++
	if cfg.ID == "" {
		cfg.ID = "pc-" + itoa(t.pseq)
	}
	m, ok := t.push[taskID]
	if !ok {
		m = map[string]*schema.PushNotificationConfig{}
		t.push[taskID] = m
	}
	m[cfg.ID] = cfg
	return cfg
}

func (t *taskStore) getPush(taskID, cfgID string) (*schema.PushNotificationConfig, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	m := t.push[taskID]
	if m == nil {
		return nil, false
	}
	v, ok := m[cfgID]
	return v, ok
}

func (t *taskStore) listPush(taskID string) ([]*schema.PushNotificationConfig, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if _, ok := t.items[taskID]; !ok {
		return nil, false
	}
	m := t.push[taskID]
	if m == nil {
		return []*schema.PushNotificationConfig{}, true
	}
	out := make([]*schema.PushNotificationConfig, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out, true
}

func (t *taskStore) deletePush(taskID, cfgID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	m := t.push[taskID]
	if m == nil {
		return false
	}
	if _, ok := m[cfgID]; !ok {
		return false
	}
	delete(m, cfgID)
	return true
}

func (t *taskStore) listTasks() []*schema.Task {
    t.mu.RLock()
    defer t.mu.RUnlock()
    out := make([]*schema.Task, 0, len(t.items))
    for _, v := range t.items {
        out = append(out, v)
    }
    return out
}

// getHistory returns the recorded state transitions for a task, if any.
func (t *taskStore) getHistory(taskID string) ([]schema.TaskStateTransition, bool) {
    t.mu.RLock()
    defer t.mu.RUnlock()
    h, ok := t.hist[taskID]
    // return a copy to avoid external mutation
    if !ok {
        return nil, false
    }
    out := make([]schema.TaskStateTransition, len(h))
    copy(out, h)
    return out, true
}
