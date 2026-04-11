# Implementation Plan: fix(resume): --from-step with --detach creates 3 phantom run records

## Objective

Ensure `wave run --detach --from-step <step> <pipeline>` creates exactly one `pipeline_run` record, not three, by fixing two guard conditions that caused phantom record creation.

## Current State

Both code fixes are already merged into `main`:

1. **`cmd/wave/commands/run.go:343-355`** — The `&& opts.FromStep == ""` guard is already removed. `opts.RunID` is always reused when set, regardless of whether `--from-step` is also present.

2. **`internal/pipeline/resume.go:159-162`** — `ResumeFromStep()` already reuses `r.executor.runID` when non-empty and only calls `createRunID()` as a fallback when running without a store.

An integration test exists at `cmd/wave/commands/run_phantom_test.go` (tagged `//go:build integration`).

## Approach

1. **Verify** the two code fixes are correct and match the issue's prescribed fix exactly.
2. **Verify** the existing integration test (`run_phantom_test.go`) correctly covers the three-record scenario.
3. **Add unit tests** (no build tag) for the specific branching logic in `run.go` run-ID selection, so normal `go test ./...` catches regressions.
4. **Add unit tests** for `ResumeFromStep` run-ID reuse in `internal/pipeline/resume_test.go`.
5. **Verify compilation** — ensure `loadManifest` helper in the integration test does not conflict with existing package-level declarations.

## File Mapping

| File | Action | Reason |
|------|--------|--------|
| `cmd/wave/commands/run.go` | verify (no change) | Fix already present at lines 343-355 |
| `internal/pipeline/resume.go` | verify (no change) | Fix already present at lines 159-162 |
| `cmd/wave/commands/run_phantom_test.go` | verify/modify | Integration test exists; may need polish |
| `cmd/wave/commands/run_test.go` | modify | Add unit tests for run-ID selection logic |
| `internal/pipeline/resume_test.go` | modify | Add unit test for runID reuse in ResumeFromStep |

## Architecture Decisions

- **Unit tests over integration tests for branching logic**: The two-line conditional guards are best verified with table-driven unit tests that don't require a full subprocess or DB, keeping normal CI green without `-tags integration`.
- **No code changes expected**: Both fixes are already in `main`. The implementation is verification + test coverage.
- **Integration test stays**: `run_phantom_test.go` covers the end-to-end scenario; it should remain under `//go:build integration` since it touches the filesystem and state DB.

## Risks

| Risk | Mitigation |
|------|-----------|
| `loadManifest` in `run_phantom_test.go` conflicts with a future helper in the same package | Check for naming collisions; rename if needed |
| Code fix was partial (only one of two bugs fixed) | Read both files and confirm both conditions explicitly |
| Integration test is broken or flaky | Run it locally with `-tags integration` before closing issue |

## Testing Strategy

- **Unit**: Table-driven tests for the `runID` selection branch in `runRun()` and the `pipelineID` selection branch in `ResumeFromStep()`.
- **Integration**: `TestPhantomRunRecords_DetachWithFromStep` in `run_phantom_test.go` — simulates the full path with a mock adapter and real state DB, asserts exactly 1 new run record.
- **Compilation**: `go build ./...` and `go vet ./...` must pass cleanly.
