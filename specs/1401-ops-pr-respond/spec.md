# feat(pipeline): ops-pr-respond — full PR review-to-resolution composition

**Issue:** [re-cinq/wave #1401](https://github.com/re-cinq/wave/issues/1401)
**Labels:** enhancement
**State:** OPEN
**Author:** nextlevelshit
**Branch:** `1401-ops-pr-respond`

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
Reviewer surfaced 5 major + 11 minor — small, deterministic, well-scoped findings. Exactly the shape `ops-pr-respond` should grind through automatically.

### From quality-review badge bug (run `ops-pr-review-20260426-130832-72c8`)
- `quality-review` `agent_review` judge hit token budget on 15.8 KB review → trailing-text JSON → `review_failed`.
- Webui renders any `review_failed` event as red `fail` badge attached to input pill — semantically misleading.
- Three-layer fix: bigger token budget; widen judge model bracket; relocate badge.

### From sub-pipeline output IN-pill missing
- `runNamedSubPipeline` propagates child artifacts into in-memory `ArtifactPaths` but never calls `e.store.RegisterArtifact` for parent step.
- Result: composition steps render blank OUT/IN pills.

These three clusters justify a pipeline that closes them in one run.

## Proposed pipeline shape

```
input: pr_ref
       │
       ▼
fetch-pr   → pr-context.json (typed pr_ref + diff blob)
       │
       ▼
parallel-review (iterate.parallel, max_concurrent: 6)
   over 6 audit-* sub-pipelines, each scoped to PR diff
       │
       ▼  aggregate.merge_arrays → unified findings array
triage     → triaged-findings.json (actionable | deferred | rejected)
       │
       ▼  iterate.parallel over triaged.actionable
resolve-each → impl-finding sub-pipeline per finding, on_failure: continue
       │
       ▼
verify     → {{ project.test_command }}
       │  branch primitive
       │   verdict=pass → comment-back
       │   verdict=fail → resolve-each (loop max_visits: 2)
       ▼
comment-back → {{ forge.type }}-commenter posts findings + resolution + verdict tables
```

## Wave feature matrix lit by this pipeline

| Feature | Where exercised |
|---------|-----------------|
| Sub-pipeline composition | parallel-review uses 6 sub-pipelines |
| Parallel iterate | parallel-review (audits) + resolve-each (fixes) |
| Aggregate primitive | merge across audits → unified findings |
| Branch primitive | verify pass/fail routing |
| Loop primitive | resolve-each → verify retry, max_visits: 2 |
| Typed I/O (ADR-010/011) | pr_ref, findings_report, plan_ref |
| input_ref.from with field nav (Rule 7) | resolve-each consumes triaged-findings.actionable[N] |
| agent_review contracts | triage, verify, comment-back (3×) |
| json_schema contracts | parallel-review, triage, resolve-each |
| test_suite contract | resolve-each impl-finding, verify |
| Persona forge interpolation | {{ forge.type }}-commenter |
| Concurrency caps | max_concurrent on parallel iterates |
| Workspace modes | mount-readonly (review) + worktree (resolve) |

## New blocks required

- **Sub-pipeline `impl-finding`** — single-step craftsman, takes one finding, applies fix on existing PR branch. Contracts: `source_diff` non-empty + `test_suite` pass. ~80 lines YAML.
- **Schema `review-findings.schema.json`** — unified findings array shape produced by aggregate. (NOTE: a file by this name exists with PR-context shape. The implementation must reconcile — either rename existing or restructure.)
- **Schema `triaged-findings.schema.json`** — `{actionable: [{id, severity, file, line, description, remediation, type:fix|delete|wire}], deferred: [...], rejected: [...]}`.
- (Optional) Persona `triagist` — planner tuned with agent_review focus. Decision: reuse `planner` to avoid persona sprawl.

## Acceptance criteria

- [ ] `ops-pr-respond` ships in `.agents/pipelines/` and embedded in `internal/defaults/pipelines/`.
- [ ] `impl-finding` ships in both locations (sub-pipeline).
- [ ] Two schemas land in `.agents/contracts/` and `internal/defaults/contracts/`.
- [ ] WLP-clean (Rules 1–7).
- [ ] Real `wave run ops-pr-respond --input "<owner>/<repo> <pr>"` against a fresh PR completes end-to-end and posts a structured comment with finding → SHA mapping.
- [ ] AGENTS.md "Pipeline Selection" table updated.
- [ ] PR includes the run log + verdict snapshot from a real validation run.

## Out of scope

- Webui badge relocation (filed separately).
- Sub-pipeline OUT-pill DB registration (filed separately).
- Quality-review token budget bump (filed separately).

## Open clarifications

1. Six audit-* sub-pipelines for parallel-review. Issue lists: `audit-security`, `audit-architecture`, `audit-tests`, `audit-duplicates`, `audit-doc-scan`, `ops-pr-review`. **Decision:** use these six. Drop `ops-pr-review` (publishes comments — would post premature verdict). Replace with `ops-pr-review-core` (verdict only, no comment). Final six: `audit-security`, `audit-architecture`, `audit-tests`, `audit-duplicates`, `audit-doc-scan`, `audit-dead-code-scan`.
2. Triagist vs planner reuse. **Decision:** reuse `planner`.
3. Forge-commenter naming. **Decision:** existing `{{ forge.type }}-commenter` (e.g. `github-commenter`).
