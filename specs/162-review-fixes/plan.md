# Implementation Plan: PR #162 Code Review Fixes

## Objective

Fix a mutex deadlock in `BubbleTeaProgressDisplay.updateFromEvent` and add missing test coverage for the `stream_activity` guard in `BasicProgressDisplay`.

## Approach

### Issue 1: Mutex Deadlock (HIGH)

**Root cause**: `EmitProgress` acquires `btpd.mu.Lock()` at `bubbletea_progress.go:122`, then calls `updateFromEvent` at `:126`, which calls `btpd.AddStep()` at `:216`. `AddStep` tries to acquire `btpd.mu.Lock()` at `:141` — deadlock on Go's non-reentrant mutex.

**Fix strategy**: Extract the step-creation logic from `AddStep` into an unexported `addStepLocked` helper that assumes the caller already holds the mutex. Then:
- `AddStep` acquires the lock and delegates to `addStepLocked`
- `updateFromEvent` calls `addStepLocked` directly (since it's already called under lock from `EmitProgress`)

This is the cleanest approach because:
- It preserves the public `AddStep` API unchanged
- It minimizes code duplication (the step-creation logic lives in one place)
- The `Locked` suffix convention is idiomatic in Go for mutex-guarded helpers

### Issue 2: Missing BasicProgressDisplay Test (MEDIUM)

**Gap**: `bubbletea_progress_resume_test.go` has `TestUpdateFromEvent_StreamActivityGuard` testing the BubbleTea variant, but `progress_test.go` has no equivalent for `BasicProgressDisplay.EmitProgress`'s `stream_activity` guard at line 647.

**Fix strategy**: Add a table-driven test `TestBasicProgressDisplay_StreamActivityGuard` following the same pattern as the BubbleTea test. The test will:
1. Create a `BasicProgressDisplay` with verbose mode and a `bytes.Buffer` writer
2. Pre-set `stepStates` to simulate running/completed/not-started states
3. Emit `stream_activity` events for each state
4. Assert output is only produced for the "running" state

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/display/bubbletea_progress.go` | modify | Extract `addStepLocked` from `AddStep`; call `addStepLocked` in `updateFromEvent` |
| `internal/display/progress_test.go` | modify | Add `TestBasicProgressDisplay_StreamActivityGuard` table-driven test |

## Architecture Decisions

1. **`addStepLocked` naming**: Following Go convention for internal helpers that require the caller to hold the mutex (e.g., `sync.Pool` internals, `net/http` internals). The `Locked` suffix signals "caller must hold the lock."

2. **No interface change**: `AddStep` remains the public API. External callers are unaffected.

3. **No structural refactor**: The fix is minimal and surgical — only the call site in `updateFromEvent` changes from `btpd.AddStep(...)` to `btpd.addStepLocked(...)`.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Introducing a new deadlock path via `addStepLocked` | Low | High | `addStepLocked` never acquires the mutex; `go test -race` validates |
| Test flakiness in `BasicProgressDisplay` test | Low | Low | Use `bytes.Buffer` for deterministic output capture |
| Breaking existing `AddStep` callers | None | — | Public API unchanged; only internal dispatch changes |

## Testing Strategy

1. **Existing tests**: `go test ./internal/display/...` must pass unchanged
2. **Race detector**: `go test -race ./internal/display/...` must pass (validates the deadlock fix)
3. **New test**: `TestBasicProgressDisplay_StreamActivityGuard` — table-driven test with 3 sub-cases:
   - `stream_activity` for running step → output produced
   - `stream_activity` for completed step → no output
   - `stream_activity` for not-started step → no output
4. **Deadlock regression**: The existing `TestUpdateFromEvent_SyntheticCompletionMarksStepDone` test calls `d.AddStep()` then `d.updateFromEvent()` — after the fix, `updateFromEvent` will use `addStepLocked` for auto-creation, which is exercised by this test path
