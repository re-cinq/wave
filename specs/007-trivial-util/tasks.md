# Tasks

## Phase 1: Setup
- [X] Task 1.1: Confirm target repo context and ensure feature branch `007-trivial-util` is checked out.
- [X] Task 1.2: Locate `main.go` and verify package layout.

## Phase 2: Core Implementation
- [X] Task 2.1: Add `util.go` with `Clamp(n, lo, hi int) int` (doc comment, same package as `main.go`). [P]
- [X] Task 2.2: Optionally wire a demo call in `main.go` only if required for build. [P]

## Phase 3: Testing
- [X] Task 3.1: Add `util_test.go` with table-driven tests (happy, lower, upper, inverted range).
- [X] Task 3.2: Run `go build ./...` and `go test ./...`; ensure green.

## Phase 4: Polish
- [X] Task 4.1: Add doc comment on exported function; run `gofmt`/`go vet`.
- [X] Task 4.2: Commit with conventional message `feat: add Clamp utility (#7)`; push branch; open PR linking issue #7.
