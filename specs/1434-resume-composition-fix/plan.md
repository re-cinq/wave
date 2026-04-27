# Implementation Plan — 1434 resume composition fix

## Objective

Make `wave resume` work for composition pipelines (`aggregate`/`iterate`) by (1) registering aggregate/iterate output artifacts in the DB so resume can rediscover them, and (2) preserving the original run's workspace path on resume so prior step outputs remain visible.

## Approach

Two independent fixes that compose:

**Bug 1 — artifact registration**: Wire `e.store.RegisterArtifact` into `executeAggregateInDAG` and `collectIterateOutputs` (the DAG-executor paths used in production). Mirror the call site in `executor.go:4517-4523` (prompt/command step path). Also have `loadResumeState` in `resume.go` query DB artifacts via `store.GetArtifacts(originalRunID)` and merge them into `state.ArtifactPaths` so composition step outputs are recovered alongside declared `OutputArtifacts`.

**Bug 2 — workspace path preservation**: Add a `WithWorkspaceOverride(path string)` executor option. When set, `Execute()` and `executeResumedPipeline` use the override as `pipelineWsPath` and skip the `os.RemoveAll`/`os.MkdirAll` cleanup. Resume CLI computes `originalWsPath := filepath.Join(wsRoot, opts.RunID)` and passes it via the new option.

The legacy `composition.go` `CompositionExecutor` is dead code in production (only invoked from tests per `Grep CompositionExecutor`). Apply parallel registration there for correctness, but the production fix lives in `executor.go`.

## File Mapping

### Modify

- `internal/pipeline/executor.go`
  - Add `WithWorkspaceOverride(string) ExecutorOption` and `workspaceOverride string` field on `DefaultPipelineExecutor`.
  - In `Execute()` (~L825-851): when `e.workspaceOverride != ""`, use it as `pipelineWsPath` and skip `RemoveAll`/`MkdirAll`. Also gate the second `pipelineWsPath` site at L1268-1272.
  - In `executeAggregateInDAG` (~L5950): after `os.WriteFile`, derive `artifactName` (already done at L5936) and call `e.store.RegisterArtifact(execution.Status.ID, step.ID, artifactName, outputPath, "json", size)`.
  - In `collectIterateOutputs` (~L5890): after writing `<stepID>-collected.json`, call `e.store.RegisterArtifact(execution.Status.ID, step.ID, "collected-output", outputPath, "json", size)`.

- `internal/pipeline/resume.go`
  - In `loadResumeState`: when `resolvedRunID != ""` and `r.executor.store != nil`, call `store.GetArtifacts(resolvedRunID, "")` and merge each record into `state.ArtifactPaths` keyed as `record.StepID + ":" + record.Name`. This recovers composition outputs that have no declared `OutputArtifacts` to walk.

- `cmd/wave/commands/resume.go`
  - After resolving `wsRoot` (~L198-201), compute `originalWsPath := filepath.Join(wsRoot, opts.RunID)` and append `pipeline.WithWorkspaceOverride(originalWsPath)` to `execOpts`.

- `internal/pipeline/composition.go` (parallel correctness, even though dead in production)
  - Add `runID string` field on `CompositionExecutor` and a constructor parameter (or `SetRunID` setter).
  - In `executeAggregate` and `collectIterateOutputs`: if `c.store != nil && c.runID != ""`, call `c.store.RegisterArtifact(...)` with derived artifact names.

### Add

- `internal/pipeline/executor_resume_composition_test.go` (or extend `executor_test.go`)
  - `TestExecuteAggregateInDAG_RegistersArtifact` — mock store, run a synthetic aggregate step, assert `RegisterArtifact` was called with expected `(stepID, name, path, "json")`.
  - `TestCollectIterateOutputs_RegistersCollectedOutput` — same shape, asserts `name == "collected-output"`.
  - `TestWithWorkspaceOverride_PreservesPath` — set override on a fresh dir containing a marker file; run `Execute`; assert marker still present (no `RemoveAll`).

- Extend `resume_test.go`
  - `TestLoadResumeState_MergesDBArtifacts` — pre-populate store with `RegisterArtifact("run", "merge-findings", "merged-findings", path, "json", n)`; run `loadResumeState`; assert `state.ArtifactPaths["merge-findings:merged-findings"] == path`.
  - `TestResume_CompositionPipeline_UsesOriginalWorkspace` — full integration covering both fixes: pre-stage an original workspace with prior step outputs + DB artifact records, resume with new run ID, assert downstream step's `inject_artifacts` resolves.

## Architecture Decisions

**1. Production fix lives in `executor.go`, not `composition.go`.**
The DAG executor (`DefaultPipelineExecutor.executeAggregateInDAG` / `executeIterateInDAG`) is the actual code path. `CompositionExecutor` in `composition.go` is currently only used in tests (verified by Grep). Apply registration in both for hygiene, but the bug bites in `executor.go`.

**2. Workspace preservation via executor option, not DB schema change.**
Adding a `workspace_path` column to `pipeline_run` would require a migration. The original workspace path is trivially `filepath.Join(wsRoot, opts.RunID)` in resume CLI — no persistence needed. An `ExecutorOption` keeps the change local and avoids schema churn.

**3. `loadResumeState` falls back to DB artifacts.**
Walking `step.OutputArtifacts` works for declarative steps but composition steps don't declare them. Reading the `artifact` table via `store.GetArtifacts(originalRunID)` gives a uniform recovery path that doesn't depend on knowing which step types produce which outputs.

**4. No `--force` workaround.**
The fix must work without `--force`. `--force` skips phase validation entirely; we want phase validation to *succeed* because the artifacts are now visible.

**5. Skip cleanup when override is set.**
Resume must not `RemoveAll` the original workspace. `e.preserveWorkspace` already gates cleanup; treat `workspaceOverride != ""` as implying preserve. The override flag explicitly opts in.

## Risks

- **Risk**: `RegisterArtifact` insert collides with existing record on resume.
  **Mitigation**: SQLite `INSERT INTO artifact` will `INSERT` a new row each call (autoincrement PK). Multiple registrations of the same `(run_id, step_id, name)` tuple just create duplicate rows — `GetArtifacts` would return both. Acceptable for now (issue calls for registration, not dedup). If problematic, switch to `INSERT OR REPLACE` keyed on `(run_id, step_id, name)` in a follow-up.
- **Risk**: Original workspace contains stale state from a prior failed step.
  **Mitigation**: Resume already starts from the failed step (`fromStep`), and that step's per-workspace cleanup happens at step level (`workspace.Type` handling). Top-level workspace dir is just a parent; preserving it preserves *prior* step outputs which is the goal.
- **Risk**: Tests assume `os.RemoveAll(pipelineWsPath)` runs at start of `Execute`.
  **Mitigation**: Existing `--preserve-workspace` tests already cover the no-cleanup branch. Mirror that pattern; don't change existing behaviour when override is empty.
- **Risk**: `composition.go` parallel fix breaks tests that construct `CompositionExecutor{tmplCtx: ...}` directly.
  **Mitigation**: Add `runID` as an optional field with zero-value semantics — tests that don't set it skip registration (guarded by `c.store != nil && c.runID != ""`).

## Testing Strategy

1. **Unit**: New tests for `executeAggregateInDAG` and `collectIterateOutputs` with a mock store asserting `RegisterArtifact` is called.
2. **Unit**: `loadResumeState` test with pre-populated DB artifacts asserting they merge into `ArtifactPaths`.
3. **Unit**: `WithWorkspaceOverride` test asserting workspace preservation.
4. **Integration** (resume_test.go pattern): Stage a `.agents/workspaces/<original-id>/` with prior step outputs and DB artifact records; call `ResumeFromStep` with a new run ID and `WithWorkspaceOverride(originalPath)`; assert the downstream step finds its injected artifact.
5. **Regression**: All existing `resume_test.go`, `executor_test.go`, `composition_test.go` continue to pass.
6. **Manual / pipeline validation**: After landing, re-trigger an `ops-pr-respond` run on a real PR, kill after `merge-findings`, `wave resume <run-id>` — must pick up at `triage` without `--force`.
