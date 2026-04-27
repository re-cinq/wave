# fix(pipeline): wave resume broken for composition pipelines — aggregate artifacts unregistered + new workspace path

**Issue:** [re-cinq/wave#1434](https://github.com/re-cinq/wave/issues/1434)
**Author:** nextlevelshit
**State:** OPEN
**Labels:** —

## Two layered bugs make `wave resume` unusable for any composition pipeline that uses `aggregate` or `iterate`

### Empirical baseline

`ops-pr-respond-20260427-190848-9617` on PR #1407 ran `fetch-pr` → `parallel-review` (6 audits) → `merge-findings` (aggregate) → `triage` (failed: schema compile error from Perl-only regex `(?!`, fixed separately). At failure, `triage` had already produced a valid `triaged-findings.json` (19 actionable / 3 deferred / 1 rejected) and `merge-findings` had written `merged-findings.json`.

`wave resume <run-id>` then `wave resume <run-id> --force` both failed:

```
Error: pipeline execution failed: Phase 'triage' failed: cannot resume from 'triage':
  prior step 'parallel-review' has no workspace artifacts
```

```
Error: pipeline execution failed: step "triage" failed: failed to inject artifacts:
  required artifact 'merged-findings' from step 'merge-findings' not found
```

### Bug 1 — aggregate / iterate steps don't register output artifacts

`internal/pipeline/composition.go` `executeAggregate` (legacy path) and `internal/pipeline/executor.go` `executeAggregateInDAG` (production path, line 5896) write `step.Aggregate.Into` to disk and stash bytes in `execution.ArtifactPaths` map, but never call `store.RegisterArtifact`. Compare `internal/pipeline/executor.go:4517-4523` for prompt/command steps.

DB inspection of failed run:

```
sqlite> SELECT step_id, name FROM artifact WHERE run_id='ops-pr-respond-20260427-190848-9617';
fetch-pr|pr-context
triage|triaged-findings
```

Only 2 artifacts. Should be 3 (also `merge-findings:merged-findings`). Same gap exists for `iterate` (no `collected-output` artifact registered in `collectIterateOutputs`).

Downstream consequences:
- `inject_artifacts` in any step depending on an aggregate output can't find it after restart.
- WebUI OUT pills are blank on aggregate/iterate steps (separate symptom of same bug).
- Resume preflight refuses because the dependency artifact is unregistered.

### Bug 2 — resume creates a new workspace path

The failed run wrote everything under `.agents/workspaces/ops-pr-respond-20260427-190848-9617/`. `wave resume` calls `store.CreateRun()` to register a NEW run record (`resumeRunID`), then passes it via `WithRunID` to the executor. The executor computes step workspaces as `wsRoot/<pipelineID>/<stepID>` where `pipelineID = e.runID = resumeRunID`. Result: a fresh empty workspace dir is created at the resume timestamp, and prior step outputs that live under the original workspace path are invisible to the resumed step.

Quote from resume error message:

```
Inspect workspace artifacts:
  ls .agents/workspaces/ops-pr-respond-20260427-201303-8495/
```

The path printed in the error doesn't match the persisted run-id. Old workspace `190848-9617` still on disk with `triaged-findings.json`, `pr-context.json`, etc.

## Acceptance Criteria

- `executeAggregateInDAG` registers an artifact in DB for every successful aggregate step.
- `collectIterateOutputs` registers a `collected-output` artifact for every iterate step.
- WebUI shows OUT pill for aggregate/iterate steps populated.
- `wave resume <run-id>` on a composition pipeline that failed after aggregate/iterate succeeds at finding upstream artifacts — without `--force` and without re-running prior steps.
- Regression test: simulated `ops-pr-respond` failure after `merge-findings`, `wave resume` picks up at next step reading the registered `merged-findings` artifact.

## Related

- #1401 (ops-pr-respond) — original feature, called out sub-pipeline OUT-pill DB registration as a known gap.
- #1412 (webui composition audit) — same bug class manifests as missing/incorrect OUT pills.
- #1413 (impl-finding worktree) — current validation blocked by these bugs.

## Source

Failed run `ops-pr-respond-20260427-190848-9617` on PR re-cinq/wave#1407 (2026-04-27).
