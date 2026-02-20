# Data Model: Optional Pipeline Steps

**Feature**: 118-optional-pipeline-steps
**Date**: 2026-02-20

## Entity Changes

### 1. Step (Modified)

**File**: `internal/pipeline/types.go`

```go
type Step struct {
    ID              string           `yaml:"id"`
    Persona         string           `yaml:"persona"`
    Dependencies    []string         `yaml:"dependencies,omitempty"`
    Optional        bool             `yaml:"optional,omitempty"`     // NEW — FR-001
    Memory          MemoryConfig     `yaml:"memory"`
    Workspace       WorkspaceConfig  `yaml:"workspace"`
    Exec            ExecConfig       `yaml:"exec"`
    OutputArtifacts []ArtifactDef    `yaml:"output_artifacts,omitempty"`
    Handover        HandoverConfig   `yaml:"handover,omitempty"`
    Strategy        *MatrixStrategy  `yaml:"strategy,omitempty"`
    Validation      []ValidationRule `yaml:"validation,omitempty"`
}
```

**Changes**:
- Add `Optional bool` field with `yaml:"optional,omitempty"` tag
- Default: `false` (Go zero value for bool)
- Position: after `Dependencies`, before `Memory` (logical grouping with step-level behavioral flags)

**Validation**: No explicit validation needed — `yaml.v3` rejects non-boolean values at parse time. The `omitempty` tag means the field is absent from serialized output when `false`.

---

### 2. Step State Constants (Modified)

**File**: `internal/pipeline/types.go`

```go
const (
    StatePending        = "pending"
    StateRunning        = "running"
    StateCompleted      = "completed"
    StateFailed         = "failed"
    StateRetrying       = "retrying"
    StateFailedOptional = "failed_optional"  // NEW — FR-004
)
```

**File**: `internal/state/store.go`

```go
const (
    StatePending        StepState = "pending"
    StateRunning        StepState = "running"
    StateCompleted      StepState = "completed"
    StateFailed         StepState = "failed"
    StateRetrying       StepState = "retrying"
    StateFailedOptional StepState = "failed_optional"  // NEW — FR-004
)
```

**Changes**:
- Add `StateFailedOptional` constant in both packages
- This is a terminal state — no transitions out of it
- Treated as "completed-like" for resume logic (step is not re-executed)

**State Transitions** (updated):
```
Pending → Running → Completed          (success)
Pending → Running → Failed             (required step failure)
Pending → Running → Retrying → Running (retry loop)
Pending → Running → FailedOptional     (optional step failure, after retries exhausted)
Pending → Skipped                      (dependency on failed optional step's artifacts)
```

---

### 3. Event (Modified)

**File**: `internal/event/emitter.go`

```go
type Event struct {
    // ... existing fields ...

    // Optional step tracking (FR-006)
    Optional bool `json:"optional,omitempty"` // NEW — true for events related to optional steps
}
```

```go
const (
    // ... existing constants ...

    // Optional step states (FR-006)
    StateFailedOptional = "failed_optional" // NEW — optional step failed (non-blocking)
)
```

**Changes**:
- Add `Optional bool` field to Event struct with `json:"optional,omitempty"`
- Add `StateFailedOptional` event state constant
- `Optional` is set on all events for optional steps: `running`, `step_progress`, `failed_optional`, `stream_activity`
- The `omitempty` tag ensures no overhead in serialized events for non-optional steps

---

### 4. Display Types (Modified)

**File**: `internal/display/types.go`

```go
const (
    StateNotStarted    ProgressState = "not_started"
    StateRunning       ProgressState = "running"
    StateCompleted     ProgressState = "completed"
    StateFailed        ProgressState = "failed"
    StateSkipped       ProgressState = "skipped"       // Already exists — reused for dependency skipping
    StateCancelled     ProgressState = "cancelled"
    StateFailedOptional ProgressState = "failed_optional" // NEW — FR-008
)
```

**Changes**:
- Add `StateFailedOptional` to the display `ProgressState` enum
- Used for rendering optional failures with distinct icon/color in the progress display
- `StateSkipped` (already exists) is used for downstream steps skipped due to missing artifacts from optional failures

---

### 5. PipelineExecution (Modified)

**File**: `internal/pipeline/executor.go`

No struct changes needed. The existing `States map[string]string` field naturally accommodates the new `"failed_optional"` state value. The execution loop changes are behavioral, not structural.

---

### 6. PipelineStatus (Modified)

**File**: `internal/pipeline/executor.go`

```go
type PipelineStatus struct {
    ID                  string
    PipelineName        string
    State               string
    CurrentStep         string
    CompletedSteps      []string
    FailedSteps         []string
    FailedOptionalSteps []string  // NEW — FR-008
    SkippedSteps        []string  // NEW — FR-009
    StartedAt           time.Time
    CompletedAt         *time.Time
}
```

**Changes**:
- Add `FailedOptionalSteps []string` — tracks step IDs that failed with optional status
- Add `SkippedSteps []string` — tracks step IDs skipped due to dependency on failed optional steps
- These provide the pipeline summary data needed for distinct display (FR-008)

---

## State Machine Diagram

```
                    ┌─────────┐
                    │ Pending │
                    └────┬────┘
                         │
                    ┌────▼────┐
              ┌────▶│ Running │◀────┐
              │     └────┬────┘     │
              │          │          │
              │    ┌─────┼─────┐   │
              │    │     │     │   │
         ┌────┴──┐│┌────▼───┐ │┌──┴────┐
         │Retrying│││Completed│ ││ Failed │  (required step → halts pipeline)
         └────────┘│└────────┘ │└───────┘
                   │           │
                   │  ┌────────▼──────┐
                   │  │FailedOptional │  (optional step → continues pipeline)
                   │  └───────────────┘
                   │
              ┌────▼───┐
              │ Skipped │  (dependency on failed optional → skipped)
              └────────┘
```

## YAML Schema Example

```yaml
steps:
  - id: navigate
    persona: navigator
    # ... required step (default)

  - id: lint
    persona: linter
    optional: true           # Non-critical — pipeline continues if this fails
    dependencies: [navigate]
    # ...

  - id: implement
    persona: implementer
    dependencies: [navigate]  # Ordering dependency on navigate only
    memory:
      strategy: fresh
      inject_artifacts:
        - step: navigate
          artifact: nav-context
          as: navigation.md
    # ...

  - id: format-check
    persona: formatter
    optional: true
    dependencies: [implement]
    memory:
      strategy: fresh
      inject_artifacts:
        - step: implement
          artifact: code-output
          as: code.tar
    # ...

  - id: review
    persona: reviewer
    dependencies: [implement, format-check]  # Ordering deps
    memory:
      strategy: fresh
      inject_artifacts:
        - step: implement      # If implement succeeded, inject its artifact
          artifact: code-output
          as: code.tar
        # format-check artifact NOT injected — review doesn't need it
    # ...
```

**Behavior in example**:
- If `lint` fails → marked `failed_optional`, pipeline continues to `implement`
- If `format-check` fails → marked `failed_optional`, `review` still runs (no artifact injection from `format-check`)
- If `format-check` fails and `review` also injected `format-check`'s artifact → `review` would be skipped

## Database Schema Impact

No schema migration required. The `step_state.state` column is `TEXT` and already stores arbitrary state strings. The new `"failed_optional"` value is just another string value stored in the same column. The `"skipped"` state also needs no schema change since `display.StateSkipped` already exists as a display concept.

The `SaveStepState` method in `store.go:263-289` needs a minor update to handle `StateFailedOptional` in the `completedAt` assignment:

```go
if state == StateCompleted || state == StateFailed || state == StateFailedOptional {
    completedAt = &now
}
```
