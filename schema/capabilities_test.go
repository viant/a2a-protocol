package schema

import (
    "encoding/json"
    "testing"
)

func TestAgentCard_Unmarshal_Object(t *testing.T) {
    js := `{
        "name": "x",
        "capabilities": {
            "streaming": true,
            "pushNotifications": false,
            "stateTransitionHistory": true
        }
    }`
    var card AgentCard
    if err := json.Unmarshal([]byte(js), &card); err != nil {
        t.Fatalf("unmarshal object capabilities: %v", err)
    }
    if !card.StreamingSupported() {
        t.Errorf("StreamingSupported() = false, want true")
    }
    if card.PushNotificationsSupported() {
        t.Errorf("PushNotificationsSupported() = true, want false")
    }
    if !card.StateTransitionHistorySupported() {
        t.Errorf("StateTransitionHistorySupported() = false, want true")
    }
    // Derived legacy list should include only true flags
    wantSet := map[string]bool{"streaming": true, "stateTransitionHistory": true}
    if len(card.Capabilities) != len(wantSet) {
        t.Fatalf("derived legacy len = %d, want %d (%v)", len(card.Capabilities), len(wantSet), card.Capabilities)
    }
    for _, v := range card.Capabilities {
        if !wantSet[v] {
            t.Fatalf("unexpected capability in derived list: %s", v)
        }
    }
    // Marshal should prefer object form
    out, err := json.Marshal(card)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var probe map[string]json.RawMessage
    if err := json.Unmarshal(out, &probe); err != nil {
        t.Fatalf("re-unmarshal: %v", err)
    }
    raw := probe["capabilities"]
    if len(raw) == 0 || raw[0] != '{' {
        t.Fatalf("capabilities JSON shape = %s, want object", string(raw))
    }
}

func TestAgentCard_Unmarshal_Legacy(t *testing.T) {
    js := `{
        "name": "x",
        "capabilities": ["streaming", "pushNotifications"]
    }`
    var card AgentCard
    if err := json.Unmarshal([]byte(js), &card); err != nil {
        t.Fatalf("unmarshal legacy capabilities: %v", err)
    }
    if !card.StreamingSupported() {
        t.Errorf("StreamingSupported() = false, want true")
    }
    if !card.PushNotificationsSupported() {
        t.Errorf("PushNotificationsSupported() = false, want true")
    }
    if card.StateTransitionHistorySupported() {
        t.Errorf("StateTransitionHistorySupported() = true, want false")
    }
    // Marshal should emit legacy array shape when no object set
    out, err := json.Marshal(card)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    var probe map[string]json.RawMessage
    if err := json.Unmarshal(out, &probe); err != nil {
        t.Fatalf("re-unmarshal: %v", err)
    }
    raw := probe["capabilities"]
    if len(raw) == 0 || raw[0] != '[' {
        t.Fatalf("capabilities JSON shape = %s, want array", string(raw))
    }
}

