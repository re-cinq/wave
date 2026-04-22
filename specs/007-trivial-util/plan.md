# Implementation Plan: 007-trivial-util

## Objective

Add a small, well-tested utility function to `main.go` of the validation sandbox to exercise the wave implement pipeline end-to-end.

## Approach

Introduce one pure utility (e.g. `Reverse(s string) string` or `Clamp(n, lo, hi int) int`) in a sibling file `util.go` next to `main.go`. Keep `main.go` untouched beyond an optional call-site demo. Add `util_test.go` with table-driven tests.

## File Mapping

- CREATE `util.go` — exported utility function with doc comment.
- CREATE `util_test.go` — table-driven unit tests.
- MODIFY `main.go` — optional: invoke utility to demonstrate (only if needed for build).
- No deletions.

## Architecture Decisions

- Pure function, no deps, stdlib only — keeps diff trivial and reviewable.
- Separate file to avoid noise in `main.go`.
- Table-driven tests per Go idiom.

## Risks

- Worktree runs in `wave` repo, not sandbox repo. Mitigation: implement step targets sandbox repo via gh CLI / correct repo context; plan is portable.
- Ambiguity of "utility function" — pick `Clamp` as deterministic, obviously useful.
- No acceptance criteria in issue — mitigation: apply minimal quality bar (doc, test, build).

## Testing Strategy

- Unit tests in `util_test.go`: happy path, lower bound, upper bound, inverted range.
- `go build ./...` must succeed.
- `go test ./...` must pass.
- No integration tests required for trivial util.
