# Implementation Plan — #1434

## Objective

Make `wave resume` work for composition pipelines that use `aggregate` / `iterate` by (1) registering aggregate/iterate outputs as DB artifacts and (2) reusing the original run's workspace directory on resume instead of creating a fresh one at resume timestamp.

## Approach

Two narrowly-scoped fixes in the production execution path (`DefaultPipelineExecutor` in `internal/pipeline/executor.go`) plus a corresponding update to the resume CLI.

### Fix 1 — Register composition outputs

`executeAggregateInDAG` and `collectIterateOutputs` already write the output file to disk and populate `execution.ArtifactPaths`. They must also call `e.store.RegisterArtifact(runID, stepID, name, path, type, size)` so the artifact is recorded in the DB. This mirrors the pattern in `writeOutputArtifacts` (executor.go:4517-4523).

For aggregate: derive the artifact name from `filepath.Base(step.Aggregate.Into)` (without extension), same logic already used in `executeAggregateInDAG:5936`.

For iterate: register the `<stepID>-collected.json` file under name `collected-output`.

Branch outputs are not produced as a parent step artifact (they only flow through sub-pipeline merging) — out of scope for this fix.

The legacy `composition.go` paths (`executeAggregate`, `collectIterateOutputs`) are also patched for completeness so test parity holds, even though only `DefaultPipelineExecutor` runs in production.

### Fix 2 — Reuse original workspace path on resume

Root cause: resume CLI creates a new run record (`resumeRunID`) and passes it as `WithRunID`. The executor uses `e.runID` directly as the workspace dir name, so `wsRoot/<resumeRunID>/<step>` is empty.

Approach: introduce a separate `workspaceRunID` on `DefaultPipelineExecutor`. When unset, defaults to `e.runID`. Add option `WithWorkspaceRunID(string)`. The CLI's `resume.go` sets it to `opts.RunID` (the original run). All workspace path computations switch from `e.runID` → `e.workspaceRunID()` accessor that returns `e.workspaceRunID` if set else `e.runID`.

This preserves the existing two-run-record dashboard UX (resume gets its own row in `pipeline_run`) while letting the resumed step see the original workspace tree. State persistence still uses `e.runID` (the new resume run) so progress and artifacts are tagged to the resume run.

Key call sites that join `wsRoot, pipelineID`:

- `executor.go:830` (Execute root setup) — keep using `pipelineID` (resume cleans `wsRoot/<resumeRunID>/`, not the original)
- `executor.go:1268` (Execute helper) — same
- `executor.go:3852` (worktree path) — switch to `e.workspaceRunID()`
- `executor.go:3943` (step workspace creation) — switch to `e.workspaceRunID()`
- `executor.go:6146` (gate text path) — switch to `e.workspaceRunID()`
- `concurrency.go:204` (parallel agent ws) — switch to `e.workspaceRunID()`
- `matrix.go:489` (matrix worker ws) — switch to `e.workspaceRunID()`

Note: at the executor entry point on resume, `Execute` is NOT called — `ResumeWithValidation` → `ResumeManager.executeResumedPipeline` → per-step execution. So the cleanup at `executor.go:840` (`os.RemoveAll(pipelineWsPath)`) does not fire on resume. The resume code path bypasses workspace cleanup, so reading from the original workspace via `workspaceRunID` is safe — no risk of wiping it.

But `executor.go:830` does compute `pipelineWsPath := filepath.Join(wsRoot, pipelineID)` for the root setup of a fresh `Execute` call — for non-resume runs this is correct. For resume we must NOT clean. The resume path doesn't go through line 830, so we leave it alone.

When new step workspaces are created during resume (e.g. for the resumed step itself), they should land in the original workspace tree (`wsRoot/<originalRunID>/<step>/`) so subsequent steps can keep reading prior step outputs. This is what `workspaceRunID` switch achieves.

### Fix 2b — `loadResumeState` artifact paths

`resume.go:367-370` builds artifact paths via `filepath.Join(stepWorkspace, artifact.Path)`. `stepWorkspace` is discovered by scanning run dirs (line 295: glob `pipelineName-*`). With `priorRunID`, line 282-289 narrows to the specific run dir. This already works correctly. Verify no change needed.

## File Mapping

### Modified

- `internal/pipeline/executor.go`
  - Add field `workspaceRunID string` to `DefaultPipelineExecutor`.
  - Add helper `func (e *DefaultPipelineExecutor) effectiveWorkspaceRunID() string` returning `e.workspaceRunID` if non-empty else `e.runID`.
  - Switch step-workspace path computations to use the helper (lines 3852, 3943, 6146).
  - Patch `executeAggregateInDAG` (line 5896): after writing the output file, call `e.store.RegisterArtifact(execution.Status.ID, step.ID, artifactName, outputPath, "json", size)`.
  - Patch `collectIterateOutputs` (line 5823): after writing `<step.ID>-collected.json`, call `e.store.RegisterArtifact(execution.Status.ID, step.ID, "collected-output", outputPath, "json", size)`.

- `internal/pipeline/options.go` (or wherever `WithRunID` lives) — add `WithWorkspaceRunID(string) ExecutorOption`.

- `internal/pipeline/concurrency.go` (line 204) and `internal/pipeline/matrix.go` (line 489) — switch to `effectiveWorkspaceRunID()`.

- `internal/pipeline/composition.go` (legacy parity)
  - `executeAggregate`: same RegisterArtifact call (requires plumbing `runID` into `CompositionExecutor` — add field + constructor arg or per-call setter).
  - `collectIterateOutputs`: write & register `collected-output` artifact (currently only stashes in template ctx; needs to also write file + register).

- `cmd/wave/commands/resume.go` (line 220-225) — append `pipeline.WithWorkspaceRunID(opts.RunID)` so the resumed executor reads from the original workspace dir.

### Added

- Tests:
  - `internal/pipeline/executor_test.go` — `TestExecuteAggregateInDAG_RegistersArtifact`: aggregate step → assert `store.RegisterArtifact` was called with the expected (runID, stepID, name, path, type) tuple. Use the existing `state.NewStateStore(":memory:")` pattern.
  - `internal/pipeline/executor_test.go` — `TestCollectIterateOutputs_RegistersArtifact`: iterate step + 2 items → assert `collected-output` artifact in DB.
  - `internal/pipeline/resume_test.go` — `TestResume_UsesOriginalWorkspaceRunID`: simulate prior run with workspace at `<wsRoot>/<originalRun>/<step>/` containing artifact files, call `ResumeWithValidation` with `WithWorkspaceRunID(originalRun)`, assert resumed step finds artifacts (no "no workspace artifacts" error).
  - `internal/pipeline/composition_test.go` — extend `TestCompositionExecutor_Aggregate_Concat` to assert artifact registration on the mock store.

### Deleted

— None.

## Architecture Decisions

1. **Separate `workspaceRunID` from `runID`** rather than reusing the original `runID` end-to-end on resume. Reason: keeps the existing dashboard UX (resume produces its own pipeline_run row with parent_run_id linkage). Trade-off: small extra plumbing, but no schema change and no behavior change for non-resume runs.

2. **Don't add a `workspace_path` column to `pipeline_run`.** The issue suggests reading `pipeline_run.workspace_path` but that column doesn't exist. The `checkpoint` table has `workspace_path` per step, but for our purposes the run-id-derived path (`wsRoot/<runID>/...`) is sufficient — we just need to point at the right run-id.

3. **Don't symlink workspaces.** The issue floats this as an alternative; rejected because symlinks add filesystem complexity (cleanup, cross-platform behavior) for no benefit over the path-resolution fix.

4. **Register aggregate output type as `json`.** The aggregate output is always JSON in practice (concat / merge_arrays / reduce all produce JSON). If a future strategy emits non-JSON, override at the call site.

5. **Patch both `composition.go` and `executor.go`.** `composition.go` is dead code in production but is still test-covered. Keeping the two paths in parity avoids confusion when readers cross-reference.

## Risks

| Risk | Mitigation |
| --- | --- |
| `WithWorkspaceRunID` plumbing missed at one of the seven call sites → resume still creates fresh workspace at one path | Grep `filepath.Join(wsRoot, pipelineID)` and `filepath.Join(wsRoot, e.runID)` exhaustively before commit; add the resume-test that asserts the actual workspace path used by the resumed step |
| `RegisterArtifact` called with a path that disappears later (e.g. workspace cleanup wipes it before next resume) | Aggregate writes to `step.Aggregate.Into` which is typically `.agents/output/...` (workspace-relative) and survives across the run; same as existing prompt/command artifacts. No regression on cleanup semantics |
| `RegisterArtifact` UNIQUE constraint on (run_id, step_id, name) collides on rework/retry | Existing artifact registrations in `writeOutputArtifacts` already hit this code path on retries. Use `_ =` ignore-error pattern (matches executor.go:4522) so retry doesn't bubble registration errors |
| Composition tests using `&CompositionExecutor{tmplCtx: ctx}` (no store) break when `executeAggregate` calls store | Guard with `if c.store != nil` (matches the executor.go pattern at line 4517) |
| `workspaceRunID` ends up referencing a nonexistent run-id (e.g. user-provided typo) | The CLI looks up the run via `store.GetRun(opts.RunID)` before calling resume (resume.go:111). If the run doesn't exist, the CLI errors out before the executor gets `WithWorkspaceRunID` |

## Testing Strategy

### Unit

- `TestExecuteAggregateInDAG_RegistersArtifact` — happy path, asserts artifact appears in store.
- `TestCollectIterateOutputs_RegistersArtifact` — iterate parallel + sequential, both register.
- `TestEffectiveWorkspaceRunID` — defaults to runID; falls back to override when set.
- `TestExecuteAggregateInDAG_NoStore` — runs without panic when `e.store == nil` (preserves test ergonomics).

### Integration

- `TestResume_AggregateStepResumesSuccessfully` — full pipeline: prompt → aggregate → prompt. Run, kill after aggregate (simulate via persisted state). `wave resume` should pick up at the second prompt step and read the aggregate's output via `inject_artifacts`. Asserts no "required artifact ... not found" error.

- `TestResume_UsesOriginalWorkspaceRunID` — verifies the resumed step's workspace path equals `<wsRoot>/<originalRunID>/<stepID>` and that artifact files written in prior steps are readable.

### Regression

- Existing `TestCompositionExecutor_Aggregate_Concat` and `TestCompositionExecutor_IterateCollectsOutputs` keep passing. Extend the first with an assertion that registration is called.

### Manual

- Re-run the failing scenario: take a real composition pipeline (`ops-pr-respond` style), inject a forced failure after the aggregate step, run `wave resume <run-id>`, confirm completion without `--force`.

## Out of Scope

- Branch composition output registration (issue notes this in passing; not blocking the documented failure).
- Sub-pipeline output registration improvements (related #1401, but separate concern).
- WebUI render fixes for OUT pill display (#1412 covers this; this fix unblocks the data side but rendering is its own ticket).
