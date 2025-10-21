package schema

import "time"

// TaskStatusUpdateEvent represents a streaming status update.
type TaskStatusUpdateEvent struct {
	TaskID    string                 `json:"taskId"`
	ContextID string                 `json:"contextId,omitempty"`
	Kind      string                 `json:"kind"` // "status-update"
	Status    TaskStatus             `json:"status"`
	Final     bool                   `json:"final"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewStatusEvent creates a status update event.
func NewStatusEvent(task *Task, final bool) *TaskStatusUpdateEvent {
	ctx := ""
	if task.ContextID != nil {
		ctx = *task.ContextID
	}
	return &TaskStatusUpdateEvent{
		TaskID:    task.ID,
		ContextID: ctx,
		Kind:      "status-update",
		Status:    task.Status,
		Final:     final,
	}
}

// TaskArtifactUpdateEvent represents a streaming artifact update.
type TaskArtifactUpdateEvent struct {
	TaskID    string                 `json:"taskId"`
	ContextID string                 `json:"contextId,omitempty"`
	Kind      string                 `json:"kind"` // "artifact-update"
	Artifact  Artifact               `json:"artifact"`
	Append    bool                   `json:"append,omitempty"`
	LastChunk bool                   `json:"lastChunk,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func NewArtifactEvent(task *Task, art Artifact, append, last bool) *TaskArtifactUpdateEvent {
	ctx := ""
	if task.ContextID != nil {
		ctx = *task.ContextID
	}
	return &TaskArtifactUpdateEvent{
		TaskID:    task.ID,
		ContextID: ctx,
		Kind:      "artifact-update",
		Artifact:  art,
		Append:    append,
		LastChunk: last,
	}
}

// Helper to update task status timestamps
func (t *Task) Touch(state TaskState) {
	t.Status.State = state
	t.Status.UpdatedAt = time.Now().UTC()
}
