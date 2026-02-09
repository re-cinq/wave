# Contract: Progress Event Schema

## Event Fields (when State == "step_progress")

### Required Fields (existing)
| Field          | Type   | Description                    |
|----------------|--------|--------------------------------|
| Timestamp      | time.Time | Event timestamp             |
| PipelineID     | string | Pipeline identifier            |
| State          | string | "step_progress"                |
| TotalSteps     | int    | Total steps in pipeline        |
| CompletedSteps | int    | Steps completed so far         |
| Progress       | int    | Percentage (0-100)             |

### ETA Field (FR-011)
| Field           | Type  | Description                                      |
|-----------------|-------|--------------------------------------------------|
| EstimatedTimeMs | int64 | Estimated milliseconds remaining. 0 = no estimate|

### Invariants
- EstimatedTimeMs MUST be 0 when no historical or configured duration data is available
- EstimatedTimeMs MUST NOT be a fabricated estimate
- The field MUST be present in the event schema (not omitempty) for forward compatibility
- Future iterations MAY populate this from expected_duration YAML config or SQLite duration history
