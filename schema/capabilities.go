package schema

// AgentCapabilities defines optional A2A features supported by an agent,
// aligning with the spec's AgentCapabilities object.
type AgentCapabilities struct {
    // Indicates if the agent supports streaming responses (e.g., SSE).
    Streaming *bool `json:"streaming,omitempty"`
    // Indicates if the agent supports push notifications for async task updates.
    PushNotifications *bool `json:"pushNotifications,omitempty"`
    // Indicates if the agent provides a history of state transitions for a task.
    StateTransitionHistory *bool `json:"stateTransitionHistory,omitempty"`
    // A list of protocol extensions supported by the agent.
    Extensions []AgentExtension `json:"extensions,omitempty"`
}

// AgentExtension declares an extension to the A2A protocol supported by the agent.
type AgentExtension struct {
    // The unique URI identifying the extension.
    URI string `json:"uri"`
    // A human-readable description of how this agent uses the extension.
    Description *string `json:"description,omitempty"`
    // If true, clients must understand and comply with the extension to interact.
    Required *bool `json:"required,omitempty"`
    // Optional, extension-specific configuration parameters.
    Params map[string]interface{} `json:"params,omitempty"`
}

