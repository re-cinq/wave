# Implementation Plan: Outcome Warning Noise Fix

## Objective

Reduce user confusion by replacing raw Go error messages with friendly, contextual messages when outcome extraction encounters empty arrays — a normal "no results" condition, not an error.

## Approach

The fix has three parts:

1. **Introduce a sentinel error type** in `outcomes.go` for the specific case of indexing into an empty array (index 0, length 0). This lets callers distinguish "no items" from genuine extraction failures.
2. **In `executor.go`**, when `ExtractJSONPath` returns the empty-array sentinel, format a user-friendly message (e.g., `"no items in enhanced_issues"`) and add it to the tracker as an outcome warning but **skip** emitting the real-time warning event. This prevents the noisy `⚠` line during execution.
3. **No changes to display code** — `progress.go` and `outcome.go` already render whatever messages they receive. The fix is at the source: better messages and fewer events.

## File Mapping

| File | Action | Purpose |
|------|--------|---------|
| `internal/pipeline/outcomes.go` | modify | Add `ErrEmptyArray` sentinel error type; return it when array index 0 on length 0 |
| `internal/pipeline/executor.go` | modify | Detect `ErrEmptyArray` in `processStepOutcomes`, format friendly message, skip real-time warning event |
| `internal/pipeline/outcomes_test.go` | modify | Add tests for empty-array sentinel error detection |
| `internal/pipeline/executor_test.go` | modify | Add test for `processStepOutcomes` empty-array handling (friendly message, no real-time event) |

## Architecture Decisions

1. **Sentinel error type over string matching**: Using `errors.As` with a typed error is idiomatic Go and avoids brittle string matching on error messages. The `EmptyArrayError` struct carries the field name so the friendly message can reference it.

2. **Suppress real-time event, keep summary warning**: The issue reports dual output (⚠ during execution + ! in summary). Suppressing the real-time event for empty-array cases removes the first occurrence while keeping the summary line — which uses the now-friendly message.

3. **Only index 0 on length 0 is treated as "no results"**: Index 5 on length 2, or index 0 on length 0 where the path *doesn't* use index 0 — these remain genuine warnings with the existing technical message. The heuristic is intentionally narrow.

## Risks

| Risk | Mitigation |
|------|------------|
| Other callers of `ExtractJSONPath` may not expect a typed error | The new type satisfies `error` interface; callers that don't check `errors.As` see the same behavior as before (just with a different message string) |
| Suppressing real-time warnings could hide genuine issues | Only suppressed for the narrow empty-array case (index 0, length 0); all other warnings still emit in real-time |
| The friendly message may lose debugging context | The full technical detail is still available in debug/trace logs; the user-facing message is simplified |

## Testing Strategy

1. **Unit tests for `ExtractJSONPath`**: Verify that index 0 on an empty array returns an `EmptyArrayError` with the correct field name. Verify that index 5 on length 2 still returns a regular error.
2. **Unit tests for `processStepOutcomes`**: Verify that empty-array errors produce a friendly warning message in the tracker but do not emit a real-time warning event. Verify that other errors still emit both.
3. **Existing tests must pass unchanged**: The existing `outcomes_test.go` tests for `wantErr: true` on array-out-of-bounds still pass because `EmptyArrayError` satisfies the `error` interface.
