# Phase 1 Data Model: Farewell Function

This feature has no persistent data. A single in-memory value is produced
per call.

## Entities

### FarewellMessage (transient string)

| Field     | Type     | Description                                                                                     |
| --------- | -------- | ----------------------------------------------------------------------------------------------- |
| `text`    | `string` | Rendered human-readable farewell line; non-empty; no trailing newline (newline added on write). |

**Invariants**:

- When input name is empty/whitespace: `text == "Farewell — see you next wave."`
- When input name is non-empty: `text == "Farewell, " + trimmed(name) + " — see you next wave."`
- Deterministic: same input → same output within a build (SC-003).
- No persistence, no ID, no lifecycle. Constructed and discarded per call.

## State Transitions

None.

## Relationships

None.
