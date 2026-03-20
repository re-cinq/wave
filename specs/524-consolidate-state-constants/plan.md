# Implementation Plan: Consolidate State Constants

## Objective

Eliminate duplicate state constant definitions across three packages (`pipeline`, `state`, `event`) by establishing `internal/state/store.go` with its typed `StepState` as the single canonical source, then updating all consumers.

## Approach

### Phase 1: Remove pipeline constants, import from state

The pipeline package currently defines untyped string constants (`StatePending = "pending"` etc.) in `types.go` and uses them for `execution.States map[string]string` and `execution.Status.State string`. Since `state.StepState` is `type StepState string`, its constants are assignable to `string` without explicit conversion.

**Strategy**: Delete the 7 constants from `pipeline/types.go`. Replace all bare `StateXxx` references in the pipeline package with `state.StateXxx`. The `state` package is already imported by the pipeline package (for `SaveStepState` calls), so no new import is needed.

### Phase 2: Remove event package duplicates

The event package defines its own overlapping constants (`StateRunning`, `StateCompleted`, `StateFailed`, `StateRetrying`, `StateSkipped`, `StateReworking`). These overlap with `state.StepState`. The event package also has additional non-step states (`StateStarted`, `StateStepProgress`, etc.) that are event-specific.

**Strategy**: Remove the 6 overlapping constants from `event/emitter.go`. Keep the event-specific constants (e.g., `StateStarted`, `StateStepProgress`, `StateStreamActivity`) since they have no equivalent in the state package. Add `import state` to the event package and reference `state.StateXxx` for the shared constants.

### Phase 3: Replace hardcoded string literals

Many places in `internal/pipeline/` use raw string literals (`"completed"`, `"failed"`, etc.) instead of constants. Replace these with `state.StateXxx` in production code. Test files using string literals are lower priority but should be updated where they reference step states.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/pipeline/types.go` | modify | Remove 7 state constants |
| `internal/state/store.go` | modify | Add `StatePending` (currently missing check — verify it exists) |
| `internal/event/emitter.go` | modify | Remove 6 overlapping state constants, add `state` import |
| `internal/pipeline/executor.go` | modify | Replace `StateXxx` → `state.StateXxx`, replace string literals |
| `internal/pipeline/resume.go` | modify | Replace `StateXxx` → `state.StateXxx`, replace string literals |
| `internal/pipeline/chatworkspace.go` | modify | Replace string literals with state constants |
| `internal/pipeline/composition.go` | modify | Replace `event.StateXxx` references to `state.StateXxx` for overlapping ones |
| `internal/pipeline/stepcontroller.go` | modify | Replace `"pending"` literal |
| `internal/pipeline/sequence.go` | modify | Replace string literals |
| `internal/pipeline/chatcontext.go` | modify | Replace string literals |
| `internal/pipeline/resume_test.go` | modify | Replace `StateCompleted` → `state.StateCompleted` |
| `internal/pipeline/executor_test.go` | modify | Replace `StateXxx` → `state.StateXxx` |
| `cmd/wave/commands/postmortem.go` | no change | Already uses `state.StateXxx` |

## Architecture Decisions

1. **`state.StepState` as canonical type**: It's already a named type with proper semantics. Using typed constants enables compile-time validation in function signatures that accept `StepState`.

2. **Keep `execution.States` as `map[string]string`**: Changing to `map[string]state.StepState` would be a deeper refactor. Since `StepState` is `type string`, the constants auto-convert to `string` on assignment. This is sufficient for the refactor scope.

3. **Keep event-specific constants in event package**: States like `StateStarted`, `StateStepProgress`, `StateStreamActivity` are display/monitoring concerns, not step lifecycle states. They belong in the event package.

4. **String literal replacement scope**: Replace literals in production code. In test files, replace only where using pipeline state constants (not external API response strings like `"success"`, `"failure"` from GitHub CI).

## Risks

| Risk | Mitigation |
|------|-----------|
| Import cycle (event → state) | Verified: no existing imports between packages, no cycle possible |
| `StepState` type mismatch in comparisons | `StepState` is `type string`, so `==` comparisons with string values still work |
| Missing constant (`StatePending` not in state) | Verified: it exists in `state/store.go` |
| Test string literals break | Only replace constants, leave raw assertion strings that match API responses |

## Testing Strategy

1. Run `go build ./...` after each phase to catch compile errors immediately
2. Run `go test ./...` after all changes to verify no behavioral regressions
3. Run `go vet ./...` to catch any type mismatches
4. No new tests needed — this is a pure mechanical refactor with no behavioral changes
