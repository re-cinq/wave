# Tasks

## Phase 1: Remove pipeline constants and update references

- [X] Task 1.1: Replace the 7 state constants in `internal/pipeline/types.go` with re-exports from `state.StepState`
- [X] Task 1.2: Update `internal/pipeline/executor.go` — replace string literal state references with constants
- [X] Task 1.3: Update `internal/pipeline/resume.go` — replace string literals with constants
- [X] Task 1.4: Update `internal/pipeline/chatworkspace.go` — replace `"pending"` and `"failed"` literals with constants
- [X] Task 1.5: Update `internal/pipeline/stepcontroller.go` — replace `"pending"` literal with `StatePending`
- [X] Task 1.6: Update `internal/pipeline/chatcontext.go` — replace `"failed"`, `"completed"` literals with constants
- [X] Task 1.7: Update `internal/pipeline/sequence.go` — replace `"completed"`, `"failed"` literals with constants
- [X] Task 1.8: Update `internal/pipeline/composition_state.go` — comment-only references, no code change needed
- [X] Task 1.9: Run `go build ./...` to verify compilation

## Phase 2: Remove event package duplicates

- [X] Task 2.1: Replace overlapping constants in `internal/event/emitter.go` with re-exports from `state.StepState`
- [X] Task 2.2: Add `state` import to `internal/event/emitter.go`
- [X] Task 2.3: Existing `event.StateXxx` references continue to work via re-exported constants (no changes needed in consumers)
- [X] Task 2.4: Run `go build ./...` to verify compilation

## Phase 3: Update test files

- [X] Task 3.1: Test files using bare `StateCompleted`/`StateFailed` continue to work via re-exported constants
- [X] Task 3.2: Test files using `event.StateXxx` continue to work via re-exported constants
- [X] Task 3.3: No test file changes needed — all references resolve through re-exports

## Phase 4: Validation

- [X] Task 4.1: Run `go test -race ./...` — full test suite passes
- [X] Task 4.2: Run `go vet ./...` — no type warnings
- [X] Task 4.3: Verified pipeline constants derive from `state.StepState` (no hardcoded duplicates)
- [X] Task 4.4: Verified single canonical source in `state/store.go`, re-exports in `pipeline/types.go` and `event/emitter.go`
