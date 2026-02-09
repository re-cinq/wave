# Contract: Step-Start Event Schema

## Event Fields (when State == "running" and emitted at step start)

### Required Fields (existing)
| Field       | Type     | Description                    |
|-------------|----------|--------------------------------|
| Timestamp   | time.Time| Event timestamp                |
| PipelineID  | string   | Pipeline identifier            |
| StepID      | string   | Step identifier                |
| State       | string   | "running"                      |
| Persona     | string   | Persona name for this step     |
| Message     | string   | Human-readable description     |

### New Fields (FR-010)
| Field    | Type   | Description                           |
|----------|--------|---------------------------------------|
| Model    | string | Model name (e.g., "claude-sonnet-4-20250514") |
| Adapter  | string | Adapter type (e.g., "claude")         |

### Invariants
- Model and Adapter MUST be populated from the resolved persona/adapter configuration
- Both fields use `json:"...,omitempty"` — absent when empty (backward compatible)
- Non-streaming adapters still include these fields (they describe the adapter, not the streaming capability)
