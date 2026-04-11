# fix(resume): --from-step with --detach creates 3 phantom run records

**Issue:** [re-cinq/wave#700](https://github.com/re-cinq/wave/issues/700)
**Branch:** `700-fix-detach-phantom-runs`
**Labels:** (none)
**State:** OPEN
**Author:** nextlevelshit

---

## Bug

Running `wave run --detach --from-step <step> <pipeline>` creates **3 separate pipeline_run records** instead of 1, and the run detail pages for the parent runs show empty steps/events.

## Root Cause

Two compounding bugs:

**Bug 1 — `cmd/wave/commands/run.go`**

The `--detach` path creates a run record, then spawns a subprocess with `--run <id> --from-step <step>`. The subprocess hits this condition:

```go
if opts.RunID != "" && opts.FromStep == "" {  // false when --from-step is also set
    runID = opts.RunID  // skipped
} else if store != nil {
    runID, err = store.CreateRun(...)  // creates a 2nd record
}
```

Because `--from-step` is set, the pre-created `--run` ID is ignored and a second record is created.

**Bug 2 — `internal/pipeline/resume.go`**

`ResumeFromStep()` always calls `createRunID()` which calls `store.CreateRun()`, creating a **3rd** record, even though the executor already has `e.runID` set via `WithRunID`.

## Result

- 3 runs in the dashboard for one logical operation
- The 2 parent runs show 0 steps, empty events, wrong status
- Actual work happens in the 3rd (leaf) run under a different ID than what the user sees

## Fix

- Always reuse `opts.RunID` when it is set (remove the `&& opts.FromStep == ""` guard)
- In `ResumeFromStep`, reuse `r.executor.runID` when non-empty instead of calling `createRunID()`

## Acceptance Criteria

- [ ] `wave run --detach --from-step <step> <pipeline>` creates exactly 1 `pipeline_run` record
- [ ] The single run ID is consistent across the parent process, subprocess, and resume execution
- [ ] The fix in `run.go` removes (or never contained) the `&& opts.FromStep == ""` guard
- [ ] The fix in `resume.go` reuses `r.executor.runID` when non-empty
- [ ] Unit tests verify both conditions without requiring integration build tag
- [ ] An integration test (`//go:build integration`) covers the end-to-end scenario
