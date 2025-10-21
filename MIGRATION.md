Migration Guide: Capabilities Shape

Overview
- Prior: `AgentCard.capabilities` was a `[]string` (e.g., ["streaming", "pushNotifications"]).
- Now: `AgentCard.capabilities` follows the spec-compliant object shape (AgentCapabilities).
- Compatibility: The implementation accepts both shapes on input and prefers emitting the object when available.

What Changed
- Added types: `schema.AgentCapabilities`, `schema.AgentExtension`.
- `schema.AgentCard` implements custom JSON marshal/unmarshal:
  - Unmarshal accepts either object or legacy array.
  - Marshal prefers the object if present; otherwise emits the legacy array.
- Added helpers on `AgentCard`:
  - `StreamingSupported()`, `PushNotificationsSupported()`, `StateTransitionHistorySupported()`
  - `SetCapabilities(AgentCapabilities)` to set the object and derive the legacy list.
- Server gating now checks these booleans:
  - Streaming methods require `capabilities.streaming: true`.
  - Push notification endpoints require `capabilities.pushNotifications: true`.

How to Migrate
1) Prefer setting capabilities using the object API:
   ```go
   card := schema.AgentCard{Name: "your-agent"}
   s, p := true, false
   card.SetCapabilities(schema.AgentCapabilities{Streaming: &s, PushNotifications: &p})
   ```

2) If you receive AgentCards from external sources:
   - No changes needed. The code accepts both object and array shapes.
   - Use the helper methods to query support rather than string matching.

3) If you previously relied on string matching:
   - Replace checks like `contains(card.Capabilities, "pushNotifications")` with `card.PushNotificationsSupported()`.

4) If you need to continue emitting the legacy array:
   - Avoid calling `SetCapabilities`; populate `card.Capabilities` directly. The server will emit the array shape.

Notes
- The sample server now emits the object shape by default and includes legacy derivation for interoperability.
- The implementation does not remove `AgentCard.Capabilities []string` to keep public API stable during transition.
