# Implementation Plan: Human Approval Gates

## Objective

Evolve Wave's existing gate system from simple auto-approve/timeout approval into a full human-in-the-loop mechanism with interactive choices, freeform text input, multi-channel interaction (CLI, TUI, WebUI, API), and choice-based step routing.

## Approach

The existing `GateConfig` and `GateExecutor` already handle four gate types. Rather than replacing this system, we extend it:

1. **Enrich `GateConfig`** with choice definitions, freeform flag, and default choice
2. **Introduce `GateDecision`** as the result type capturing choice, text, and timestamp
3. **Define `GateHandler` interface** with channel-specific implementations (CLI, TUI, WebUI)
4. **Wire decisions into `PipelineContext`** so downstream steps see `gate.<step>.choice` etc.
5. **Add `--auto-approve` CLI flag** that sets a runtime flag propagated to the executor
6. **Implement choice routing** in the executor's main loop: `_fail` fails, named step re-queues

The key architectural insight: the `GateExecutor` currently blocks until context cancellation or timeout. The new design replaces this blocking with a `GateHandler` interface that abstracts the interaction channel. The executor calls `handler.Prompt(gate)` and gets back a `GateDecision`.

## File Mapping

### New Files
| Path | Purpose |
|------|---------|
| `internal/pipeline/gate_handler.go` | `GateHandler` interface + `GateDecision` type + `CLIGateHandler` |
| `internal/pipeline/gate_handler_test.go` | Tests for CLI handler and auto-approve logic |

### Modified Files
| Path | Change |
|------|--------|
| `internal/pipeline/types.go` | Add `Choices`, `Freeform`, `Default` fields to `GateConfig`; add `GateChoice` struct |
| `internal/pipeline/gate.go` | Refactor `executeApproval` to use `GateHandler`; add decision storage + artifact writing |
| `internal/pipeline/gate_test.go` | Update existing tests, add choice routing + freeform tests |
| `internal/pipeline/context.go` | Add `GateDecisions` map to `PipelineContext`; populate `gate.<step>.*` template vars |
| `internal/pipeline/executor.go` | Propagate `autoApprove` flag; wire gate decisions into context; handle choice routing in main loop |
| `internal/pipeline/composition.go` | Pass handler + autoApprove to gate executor |
| `cmd/wave/commands/run.go` | Add `--auto-approve` flag to `RunOptions`; propagate to executor |
| `internal/tui/gate_modal.go` | New: Bubble Tea gate approval modal component |
| `internal/webui/handlers_control.go` | Add `POST /api/runs/{id}/gates/{step}/approve` endpoint |

### Files Unchanged but Referenced
| Path | Reason |
|------|--------|
| `internal/event/emitter.go` | Already has `StateGateWaiting`, `StateGateResolved` -- sufficient |
| `internal/state/store.go` | Gate decisions can be stored as step metadata; no schema changes needed |
| `internal/manifest/manifest.go` | `GateConfig` lives in pipeline types, not manifest -- no changes |

## Architecture Decisions

### 1. GateHandler Interface over Channel Enum

Instead of a `channel: cli|tui|web` field on `GateConfig`, we define a `GateHandler` interface:

```go
type GateHandler interface {
    Prompt(ctx context.Context, gate *GateConfig) (*GateDecision, error)
}
```

The executor is injected with the appropriate handler at construction time based on the runtime mode (CLI vs TUI vs WebUI). This avoids gate config knowing about presentation and keeps the gate YAML declarative.

### 2. GateDecision as Value Object

```go
type GateDecision struct {
    Choice    string    // selected choice key
    Label     string    // human-readable label
    Text      string    // freeform text (empty if not provided)
    Timestamp time.Time
    Target    string    // resolved target step (from choice definition)
}
```

Stored in `PipelineContext.GateDecisions[stepID]` and projected as template variables.

### 3. Choice Routing via Special Return Value

Gate execution returns the decision. The caller (`executeStep` / composition executor) inspects `decision.Target`:
- `""` or matches next DAG step: normal flow continues
- `"_fail"`: pipeline fails with gate-abort error
- `"<step-id>"`: re-queue that step (reset its state to pending, re-enter main loop)

Re-queuing is conceptually similar to the existing `rework` mechanism but triggered by human choice rather than contract failure.

### 4. Auto-Approve as Executor Option

`--auto-approve` becomes `WithAutoApprove(bool)` executor option. When set, the `GateExecutor` uses an `AutoApproveHandler` that returns the gate's `Default` choice immediately. This avoids threading a flag through every gate — the handler itself encapsulates the behavior.

### 5. Freeform Text as Gate Artifact

When `Freeform: true` and the user provides text, it's written to `.wave/artifacts/gate-<step-id>-text` and registered as a step artifact. Downstream steps can inject it via `inject_artifacts`.

### 6. WebUI Gate Resolution via State Store

The WebUI handler writes the decision to the state store. The `GateExecutor` polls the store (or uses a channel) to detect the resolution. This decouples the HTTP handler from the executor goroutine.

## Risks

| Risk | Mitigation |
|------|------------|
| Re-queuing a step breaks topological ordering | Reset only the target step + its dependents; validate no cycle is created |
| CLI handler blocks in non-interactive context (detached, CI) | `--auto-approve` is required for `--detach`; fail fast if stdin is not a TTY without auto-approve |
| TUI modal conflicts with pipeline progress display | Gate modal takes focus; progress pauses during prompt |
| WebUI polling latency | Use 1s poll interval with Server-Sent Events upgrade path |
| Freeform text injection security | Sanitize through existing `security.SanitizeForPrompt()` |
| Backward compatibility of existing `gate:` YAML | New fields are all optional; existing pipelines work unchanged |

## Testing Strategy

### Unit Tests
- `gate_handler_test.go`: `AutoApproveHandler` returns default choice; `CLIGateHandler` with mocked stdin
- `gate_test.go`: Updated approval tests with choices, freeform, default fallback, timeout-to-default
- `context_test.go`: Gate decision template variable resolution (`gate.<step>.choice` etc.)
- `executor_test.go`: Choice routing (`_fail`, re-queue step), auto-approve propagation

### Integration Tests
- Full pipeline with gate step: plan -> approve -> implement, testing approve path
- Full pipeline with gate step: plan -> approve -> plan (revision loop via "Revise" choice)
- `--auto-approve` flag end-to-end with default choice
- Freeform text artifact injection into downstream step

### Existing Test Preservation
- All existing `gate_test.go` tests must continue passing (backward compat for `timer`, `pr_merge`, `ci_pass`)
