# feat(pipeline): ops-pr-respond — full PR review-to-resolution composition

**Issue:** [re-cinq/wave#1401](https://github.com/re-cinq/wave/issues/1401)
**Author:** nextlevelshit
**State:** OPEN
**Labels:** enhancement

## Goal

Author a new composition pipeline `ops-pr-respond` that takes a PR ref, runs a parallel multi-axis review, triages findings, applies fixes per finding to the PR's head branch, verifies the result, and posts a structured response comment. Designed as the canonical **showcase pipeline** for the Wave feature surface — every primitive (typed I/O, sub-pipeline, parallel iterate, aggregate, branch, loop, agent_review, json_schema, test_suite, forge interpolation) lit up in one workflow.

## Context

Currently the fleet (post-WLP, 34 pipelines) ships:

- `ops-pr-review` — produces a verdict/comment (one-way).
- `inception-bugfix` — issue → audit → impl-issue-core → review (no PR-input shape).
- `impl-issue` / `impl-issue-core` — issue → PR (creates fresh PR).

Nothing closes the loop on `PR + review → fix → verify → response comment`. Manual hand-off is the only option today.

## Findings driving this issue

### From PR #1369 review
The reviewer surfaced 5 major issues + 11 minor observations. Each was small, deterministic, well-scoped — exactly the shape of finding `ops-pr-respond` should grind through automatically.

### From quality-review badge bug
- `quality-review` step's `agent_review` judge ran out of budget on a 15.8 KB review and produced trailing-text JSON, triggering `review_failed`.
- Webui renders any `review_failed` event as a red `fail` badge attached to the input pill, semantically misleading.
- Three-layer fix: bigger token budget on quality judge; widen judge model bracket; relocate badge in the run-detail template.

### From sub-pipeline output IN-pill missing
- `runNamedSubPipeline` propagates child artifacts into the parent's in-memory `ArtifactPaths` but never calls `e.store.RegisterArtifact` for the parent step.
- Result: `GetArtifacts(parentRunID, parentStepID)` returns empty → webui's OUT/IN pills on composition steps are blank.
- Fix: in the sub-pipeline propagation loop, also call `e.store.RegisterArtifact(...)`.

## Proposed pipeline shape

```
fetch-pr → parallel-review (6 audits via iterate parallel)
        → triage
        → resolve-each (parallel iterate of impl-finding)
        → verify
        → branch (pass → comment-back, fail → loop resolve-each max_visits: 2)
        → comment-back
```

## Wave feature matrix lit by this pipeline

Sub-pipeline composition, parallel iterate, aggregate primitive, branch primitive, loop primitive, typed I/O (ADR-010/011), input_ref.from with field nav (Rule 7), agent_review contracts (3×), json_schema contracts, test_suite contract, persona forge interpolation, concurrency caps, workspace modes (mount-readonly + worktree).

## New blocks required

- Sub-pipeline `impl-finding` — single-step craftsman, takes one finding, applies fix on existing PR branch, contracts: `source_diff` non-empty + `test_suite` pass. ~80 lines YAML.
- Schema `review-findings.schema.json` — unified findings array shape produced by aggregate.
- Schema `triaged-findings.schema.json` — `{actionable: [...], deferred: [...], rejected: [...]}`.
- (Optional) Persona `triagist` — planner tuned with agent_review focus.

## Acceptance criteria

- [ ] `ops-pr-respond` ships in `.agents/pipelines/` and embedded in `internal/defaults/pipelines/`.
- [ ] `impl-finding` ships in both locations (sub-pipeline).
- [ ] Two schemas land in `.agents/contracts/` and `internal/defaults/contracts/`.
- [ ] WLP-clean (Rules 1-7).
- [ ] Real `wave run ops-pr-respond --input "<owner>/<repo> <pr>"` against a fresh PR completes end-to-end and posts a structured comment with finding → SHA mapping.
- [ ] AGENTS.md "Pipeline Selection" table updated.
- [ ] PR includes the run log + verdict snapshot from a real validation run.

## Out of scope

- Webui badge relocation (filed separately).
- Sub-pipeline OUT-pill DB registration (filed separately).
- Quality-review token budget bump (filed separately).
