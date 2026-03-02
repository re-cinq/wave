# Tasks

## Phase 1: Sentinel Error Type
- [X] Task 1.1: Add `EmptyArrayError` struct to `internal/pipeline/outcomes.go` with a `Field` string field and `Error()` method that returns a user-friendly message like `"no items in <field>"`
- [X] Task 1.2: Modify the array bounds check in `ExtractJSONPath` (line 58-59) to return `EmptyArrayError{Field: field}` when `arrayIdx == 0 && len(arr) == 0`, and keep the existing `fmt.Errorf` for all other out-of-bounds cases

## Phase 2: Executor Integration
- [X] Task 2.1: In `processStepOutcomes` (`internal/pipeline/executor.go` ~line 1789-1801), use `errors.As` to detect `EmptyArrayError` from `ExtractJSONPath`
- [X] Task 2.2: When `EmptyArrayError` is detected, format a friendly message (e.g., `"[step-id] outcome: no items in <field> — skipping %s extraction from %s"`) and add it to `deliverableTracker.AddOutcomeWarning()` but **skip** emitting the real-time `"warning"` event
- [X] Task 2.3: Ensure non-empty-array errors continue the existing behavior (both tracker warning + real-time event)

## Phase 3: Testing
- [X] Task 3.1: Add test case to `outcomes_test.go` for empty array (index 0, length 0) — verify `errors.As` returns `EmptyArrayError` with correct field name [P]
- [X] Task 3.2: Add test case to `outcomes_test.go` for non-empty array out of bounds (index 5, length 2) — verify it does NOT return `EmptyArrayError` [P]
- [X] Task 3.3: Add test to verify `processStepOutcomes` with empty-array error produces tracker warning but no real-time event [P]
- [X] Task 3.4: Run full test suite (`go test ./...`) to verify no regressions

## Phase 4: Polish
- [X] Task 4.1: Verify the friendly message reads well in context by checking the format matches the outcome summary renderer
- [X] Task 4.2: Confirm `go vet ./...` passes
