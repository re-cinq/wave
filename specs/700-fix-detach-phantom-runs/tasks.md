# Tasks

## Phase 1: Verification

- [X] Task 1.1: Confirm fix in `cmd/wave/commands/run.go` — verify lines 343-355 have no `&& opts.FromStep == ""` guard and `opts.RunID` is always reused when set
- [X] Task 1.2: Confirm fix in `internal/pipeline/resume.go` — verify lines 159-162 reuse `r.executor.runID` when non-empty and only fall back to `createRunID()` when it is empty
- [X] Task 1.3: Verify `cmd/wave/commands/run_phantom_test.go` compiles cleanly (check `loadManifest` helper for naming collisions in the `commands` package)

## Phase 2: Unit Tests

- [X] Task 2.1: Add unit test in `cmd/wave/commands/run_runid_test.go` — `TestRunID_ReusedWhenRunIDAndFromStepBothSet` verifies that when `opts.RunID != ""` and `opts.FromStep != ""`, the existing run ID is reused (not a new `CreateRun` call) [P]
- [X] Task 2.2: Add unit test in `internal/pipeline/resume_test.go` — `TestResumeFromStep_ReusesExecutorRunID` verifies that `ResumeFromStep` does not call `store.CreateRun` when `executor.runID` is already set [P]

## Phase 3: Integration Test Validation

- [X] Task 3.1: Run `go build ./...` and `go vet ./...` to confirm no compilation errors
- [X] Task 3.2: Run `go test ./cmd/wave/commands/... ./internal/pipeline/...` (unit tests) to confirm new tests pass
- [ ] Task 3.3: Run `go test -tags integration ./cmd/wave/commands/...` to confirm `TestPhantomRunRecords_DetachWithFromStep` passes

## Phase 4: Polish

- [X] Task 4.1: Ensure the integration test in `run_phantom_test.go` uses `t.TempDir()` or `t.Cleanup()` rather than manual `os.RemoveAll` to avoid leftover state on test failure
- [X] Task 4.2: Final review — confirm issue #700 acceptance criteria are all met and PR is ready
