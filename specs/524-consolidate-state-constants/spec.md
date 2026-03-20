# refactor: consolidate pipeline/state state constants

**Issue**: [#524](https://github.com/re-cinq/wave/issues/524)
**Author**: nextlevelshit
**Labels**: none
**State**: OPEN

## Description

State constant definitions are split across two files with inconsistent typing:

- `internal/pipeline/types.go` contains untyped string constants for step states
- `internal/state/store.go` contains typed `StepState` constants

This duplication creates maintenance burden and potential inconsistencies. Consolidate into a single canonical definition (recommend `internal/state/store.go` with `StepState` type) and update `types.go` to import from there.

## Additional Finding

A third set of duplicate state constants exists in `internal/event/emitter.go` (also untyped strings). Additionally, many hardcoded string literals (`"completed"`, `"failed"`, etc.) are used throughout `internal/pipeline/` instead of referencing constants.

## Acceptance Criteria

1. Single canonical definition of state constants in `internal/state/store.go` using the typed `StepState` type
2. `internal/pipeline/types.go` no longer defines its own state constants
3. `internal/event/emitter.go` no longer defines its own state constants
4. All references to the removed constants are updated to use `state.StateXxx`
5. Hardcoded string literals for states in production code are replaced with constants where practical
6. All tests pass (`go test ./...`)
7. No behavioral changes — pure refactor
