# fix(pipeline): wave resume broken for composition pipelines — aggregate artifacts unregistered + new workspace path

**Issue**: [re-cinq/wave#1434](https://github.com/re-cinq/wave/issues/1434)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: (none)
**Branch**: `1434-resume-composition-fix`

## Summary

Two layered bugs make `wave resume` unusable for any composition pipeline that uses `aggregate` or `iterate`.

## Empirical Baseline

`ops-pr-respond-20260427-190848-9617` on PR #1407 ran `fetch-pr` → `parallel-review` (6 audits) → `merge-findings` (aggregate) → `triage` (failed). At failure, `triage` had already produced a valid `triaged-findings.json` and `merge-findings` had written `merged-findings.json`.

`wave resume <run-id>` then `wave resume <run-id> --force` both failed:

```
Error: pipeline execution failed: ❌ Phase 'triage' failed: cannot resume from 'triage':
  prior step 'parallel-review' has no workspace artifacts
```

```
Error: pipeline execution failed: step "triage" failed: failed to inject artifacts:
  required artifact 'merged-findings' from step 'merge-findings' not found
```

## Bug 1 — Aggregate / Iterate Steps Don't Register Output Artifacts

`internal/pipeline/composition.go` `executeAggregate` (L450-492) writes `step.Aggregate.Into` to disk and stashes bytes in `tmplCtx.SetStepOutput`, but never calls `store.RegisterArtifact`. Same gap in `executeIterate`/`collectIterateOutputs`.

The DAG-path equivalents in `internal/pipeline/executor.go`:
- `executeAggregateInDAG` (L5896-5964) sets `execution.ArtifactPaths[step.ID+":"+name]` but never calls `e.store.RegisterArtifact`.
- `collectIterateOutputs` (L5823-5894) writes `<stepID>-collected.json`, populates `execution.ArtifactPaths[step.ID+":collected-output"]`, but never registers.

Compare with `internal/pipeline/executor.go:4517-4523` (prompt/command step path) which does call `RegisterArtifact`.

DB inspection of failed run:

```
sqlite> SELECT step_id, name FROM artifact WHERE run_id='ops-pr-respond-20260427-190848-9617';
fetch-pr|pr-context
triage|triaged-findings
```

Only 2 of 3 artifacts. `merge-findings:merged-findings` is missing.

Downstream consequences:
- `inject_artifacts` in steps depending on aggregate output can't resolve via DB after restart.
- WebUI OUT pills are blank on aggregate/iterate steps.
- Resume `loadResumeState` only walks declared `step.OutputArtifacts` — composition steps lack those, so `state.ArtifactPaths` never gets populated for them, and the resume preflight refuses.

## Bug 2 — Resume Creates a New Workspace Path

The failed run wrote everything under `.agents/workspaces/ops-pr-respond-20260427-190848-9617/`. `wave resume` keeps the original run-id parameter (`opts.RunID`) but in `cmd/wave/commands/resume.go:171` calls `store.CreateRun(...)` to mint a new `resumeRunID` for dashboard visibility, then passes that new ID via `pipeline.WithRunID(resumeRunID)`.

The executor uses `e.runID` to compute `pipelineWsPath := filepath.Join(wsRoot, pipelineID)` (executor.go:830, 1268), so the resumed run runs inside `.agents/workspaces/ops-pr-respond-20260427-201303-8495/` — a fresh empty dir. The first step run inside that new workspace can't see prior step outputs that live under the original path.

Quote from the resume error message:

```
Inspect workspace artifacts:
  ls .agents/workspaces/ops-pr-respond-20260427-201303-8495/
```

The path printed in the error doesn't match the persisted run-id. Old workspace `190848-9617` still on disk with `triaged-findings.json`, `pr-context.json`, etc.

## Acceptance Criteria

- `executeAggregateInDAG` registers an artifact in the DB for every successful aggregate step.
- `collectIterateOutputs` registers a `collected-output` artifact for iterate steps.
- WebUI shows OUT pill populated for aggregate/iterate steps.
- `wave resume <run-id>` on a composition pipeline that failed after aggregate/iterate succeeds at finding upstream artifacts — without `--force` and without re-running prior steps.
- The resumed pipeline runs inside the original workspace path, not a freshly minted one.
- Regression test: `ops-pr-respond` on a representative PR, kill after `merge-findings`, `wave resume` — pipeline picks up at `triage` reading the registered `merged-findings` artifact.

## Related

- #1401 (ops-pr-respond) — original feature; called out sub-pipeline OUT-pill DB registration as a known gap.
- #1412 (webui composition audit) — same bug class manifests as missing/incorrect OUT pills.
- #1413 (impl-finding worktree) — current validation blocked by these bugs.

## Source

Failed run `ops-pr-respond-20260427-190848-9617` on PR re-cinq/wave#1407 (2026-04-27).
