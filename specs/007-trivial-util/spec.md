# validation target #7: trivial task for pipeline run

## Issue

Synthetic issue for wave pipeline validation. Safe to close/delete. Task: add a utility function or fix a trivial bug in main.go.

## Metadata

- Repository: re-cinq/wave-validation-sandbox
- Issue: #7
- URL: https://github.com/re-cinq/wave-validation-sandbox/issues/7
- Author: nextlevelshit
- State: OPEN
- Labels: (none)

## Acceptance Criteria

- New exported utility function added to `main.go` (or helper file alongside it).
- Function has clear signature, deterministic behavior, and doc comment.
- Unit test covers happy path + at least one edge case.
- `go build ./...` and `go test ./...` pass.

## Notes

- Complexity: trivial
- Skipped speckit steps: specify, clarify, checklist, analyze
- No specific bug identified — choose minimal value-add utility.
