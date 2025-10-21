package schema

import "time"

// TaskStateTransition is a minimal record of a task state change.
type TaskStateTransition struct {
    State TaskState `json:"state"`
    At    time.Time `json:"at"`
}

