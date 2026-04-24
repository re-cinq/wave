# Tasks

## Phase 1: Setup
- [X] Task 1.1: Confirm on branch `001-hello-smoke` at repo root
- [X] Task 1.2: Verify no existing `hello.go` (or back up if present)

## Phase 2: Core Implementation
- [X] Task 2.1: Write `hello.go` with exact snippet from issue
- [X] Task 2.2: Run `gofmt -w hello.go`

## Phase 3: Testing
- [X] Task 3.1: Run `go vet ./...`
- [X] Task 3.2: Run `go build ./...`

## Phase 4: Polish
- [X] Task 4.1: Stage `hello.go` + `specs/001-hello-smoke/`
- [X] Task 4.2: Commit with conventional message `feat: add hello() function`
- [ ] Task 4.3: Open PR referencing issue #1 (handled by next pipeline step)
