package schema

// StreamingSupported reports if streaming is supported either via the
// spec-compliant capabilities object or the legacy string slice.
func (a *AgentCard) StreamingSupported() bool {
    if a == nil {
        return false
    }
    if a.capObj != nil && a.capObj.Streaming != nil {
        return *a.capObj.Streaming
    }
    return contains(a.Capabilities, "streaming")
}

// PushNotificationsSupported reports if push notifications are supported.
func (a *AgentCard) PushNotificationsSupported() bool {
    if a == nil {
        return false
    }
    if a.capObj != nil && a.capObj.PushNotifications != nil {
        return *a.capObj.PushNotifications
    }
    return contains(a.Capabilities, "pushNotifications")
}

// StateTransitionHistorySupported reports if state transition history is supported.
func (a *AgentCard) StateTransitionHistorySupported() bool {
    if a == nil {
        return false
    }
    if a.capObj != nil && a.capObj.StateTransitionHistory != nil {
        return *a.capObj.StateTransitionHistory
    }
    return contains(a.Capabilities, "stateTransitionHistory")
}

func contains(list []string, v string) bool {
    for _, e := range list {
        if e == v {
            return true
        }
    }
    return false
}

// SetCapabilities sets the spec-compliant capabilities object and derives
// the legacy string list for backward compatibility.
func (a *AgentCard) SetCapabilities(c AgentCapabilities) {
    a.capObj = &c
    var list []string
    if c.Streaming != nil && *c.Streaming {
        list = append(list, "streaming")
    }
    if c.PushNotifications != nil && *c.PushNotifications {
        list = append(list, "pushNotifications")
    }
    if c.StateTransitionHistory != nil && *c.StateTransitionHistory {
        list = append(list, "stateTransitionHistory")
    }
    a.Capabilities = list
}
