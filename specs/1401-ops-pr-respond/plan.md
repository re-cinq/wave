# Plan: ops-pr-respond Composition Pipeline

## Objective

Ship a forge-agnostic composition pipeline `ops-pr-respond` that closes the `PR + review → triage → fix → verify → response comment` loop, plus a sub-pipeline `impl-finding` that fixes a single triaged finding on the PR's head branch. Lights up every Wave primitive (sub-pipeline, parallel iterate, aggregate, branch, loop, typed I/O, agent_review, json_schema, test_suite, forge interpolation) in a single canonical showcase workflow.

## Approach

Build composition layer only — reuse existing audit-* sub-pipelines and `ops-pr-review-core` for the parallel-review fanout. New craftsman sub-pipeline `impl-finding` does the per-finding fix. Two new JSON schemas type the aggregate output and the triage gate. Triage step decides `actionable/deferred/rejected` buckets; only `actionable` items feed the parallel resolve-each iterate. A `loop` wraps resolve-each + verify with `max_visits: 2` so a failed test pass triggers one re-fix attempt before bailing. The terminal `comment-back` step posts a structured response with finding → SHA mapping.

WLP rules enforced: typed I/O on every step (`pr_ref`, `findings`, `triaged_findings`, `fix_result`, `verdict`), explicit `output_artifacts.type`, deterministic contracts (`json_schema` or `agent_review` or `test_suite`), canonical artifact paths under `.agents/output/`, sub-pipeline composition where blocks already exist, iterate over typed collections via `input_ref.from` with field navigation (Rule 7).

## File Mapping

### Created

- `.agents/pipelines/ops-pr-respond.yaml` — composition pipeline (~250 lines).
- `internal/defaults/pipelines/ops-pr-respond.yaml` — embedded copy (parity).
- `.agents/pipelines/impl-finding.yaml` — single-step craftsman sub-pipeline (~100 lines).
- `internal/defaults/pipelines/impl-finding.yaml` — embedded copy.
- `.agents/contracts/review-findings.schema.json` — unified findings array shape produced by aggregate.
- `internal/defaults/contracts/review-findings.schema.json` — embedded copy.
- `.agents/contracts/triaged-findings.schema.json` — `{actionable, deferred, rejected}` shape.
- `internal/defaults/contracts/triaged-findings.schema.json` — embedded copy.
- `specs/1401-ops-pr-respond/{spec.md,plan.md,tasks.md}` — planning artifacts.

### Modified

- `AGENTS.md` — Pipeline Selection table: add `ops-pr-respond` row + `impl-finding` row.

### Not modified (out of scope per issue)

- `internal/webui/templates/run-detail.*` — badge relocation filed separately.
- `internal/pipeline/sub_pipeline*.go` — `RegisterArtifact` fix filed separately.
- `internal/agent/judge.go` — quality-review budget bump filed separately.

### Optional / decided against

- Custom `triagist` persona — issue marks optional. Plan to reuse existing `summarizer` or `navigator` persona with focused prompt + `agent_review` contract instead. Simpler. Avoids defaults-bloat. Skip unless validation run reveals the existing personas can't hit the triage quality bar.

## Architecture Decisions

### AD-1: Reuse `ops-pr-review-core`, do not re-run audits

`ops-pr-review-core` already runs three parallel reviewers (security, quality, slop) plus a synthesis verdict. Calling it inside the parallel-review iterate gets us 3 of the 6 axes for free. Add audit-doc-scan, audit-duplicates, audit-architecture as the other 3 axes. Total: 6 parallel sub-pipelines, all using existing blocks.

### AD-2: Aggregate strategy = `merge_arrays` plus schema-validated reshape

Use the aggregate primitive (`strategy: merge_arrays`) as in `ops-parallel-audit`. Then a thin persona step normalizes shape to match `review-findings.schema.json` (each item: `{source, severity, file, line, title, description, recommendation}`). Aggregate alone won't enforce shape — the normalize step ensures the triage step has a clean contract.

### AD-3: Triage gate = `agent_review` + `json_schema`

Triage step uses `summarizer` persona, prompt-only, with dual contract: `json_schema` against `triaged-findings.schema.json` (must_pass) and `agent_review` (warn). The judge persona is `navigator`, model `cheapest`, token_budget 6000. Outputs three buckets: `actionable` (auto-fixable), `deferred` (filed as follow-up issue notes), `rejected` (false positive / out of scope).

### AD-4: resolve-each = parallel iterate over `actionable[]` calling `impl-finding`

Use `iterate` with `mode: parallel`, `max_concurrent: 3` (matches WLP guidance). The `over` source uses `input_ref.from` with field navigation (Rule 7) to read `triage.output.actionable`. Each iteration spawns `impl-finding` with the finding object as input.

### AD-5: `impl-finding` = single craftsman step on PR worktree

Reuses pattern from `impl-issue-core.implement`: `craftsman` persona, `worktree` workspace branched from PR head ref. Two contracts:
- `non_empty_file` on a per-finding diff artifact (`source_diff` semantic — name it `fix-diff`).
- `test_suite` running `{{ project.contract_test_command }}`, must_pass, on_failure: rework, rework_step: fix-finding (a paired rework_only step).

`impl-finding` writes one commit per finding with a `fix(<scope>): <finding-title> [refs: finding-N]` message. The verify step downstream maps finding → SHA from these commits.

### AD-6: Verify = run `ops-pr-review-core` again on the modified branch

After resolve-each, verify by re-running `ops-pr-review-core` against the same PR (now with new commits). Inspect verdict — if APPROVE or COMMENT, branch into comment-back. If REQUEST_CHANGES or REJECT, branch into the loop body (re-triage + re-resolve, capped at `max_visits: 2`).

### AD-7: Branch + loop primitives wire pass/fail paths

Use `branch` primitive on the verify step's verdict. `pass` route → `comment-back`. `fail` route → re-enters loop (`max_visits: 2`). Loop wraps `triage → resolve-each → verify` so re-triage can incorporate the verify findings.

### AD-8: comment-back = structured response with finding → SHA mapping

Final step: `summarizer` persona with `{{ forge.cli_tool }} {{ forge.pr_command }} comment` to post a single comment listing each addressed finding, the commit SHA that fixed it, and any deferred/rejected findings with reasons. Output JSON matches `gh-pr-comment-result.schema.json` (already in fleet).

### AD-9: Embedded defaults parity

`internal/defaults/pipelines/` and `internal/defaults/contracts/` mirror `.agents/` versions byte-for-byte. Per memory `feedback_defaults_agnostic.md`, `internal/defaults/` is the language-agnostic shipped fleet — these new pipelines are forge-agnostic via `{{ forge.* }}` interpolation, so they belong in defaults.

## Risks

### R1: Loop re-entry causes test thrashing
Mitigation: `max_visits: 2` cap from issue spec. Each re-entry runs full `ops-pr-review-core` then full resolve-each — bounded but expensive. Document expected cost in pipeline metadata (~6× core review cost worst case).

### R2: Aggregate output may have inconsistent finding shapes across 6 sources
Mitigation: AD-2 normalize step. Schema-validate before triage. If a source produces malformed findings, normalize step flags them, triage drops them into `rejected`.

### R3: `impl-finding` parallel fixes can produce merge conflicts on shared files
Mitigation: `max_concurrent: 3` matches existing WLP fanout; each iteration commits its own SHA on the same branch, sequentially via the executor's commit lock. Worst case = late commits land on stale tree, test fails, rework_step takes one shot. If it still fails, loop re-entry handles it.

### R4: Real validation run may reveal latent gaps in `ops-pr-review-core` (token budget, model bracket)
Mitigation: Issue explicitly defers the token-budget fix. If validation hits the cap, document the failure and file a follow-up — do not patch in this PR.

### R5: Sub-pipeline OUT-pill empty in webui (per issue context)
Mitigation: Issue explicitly defers this fix. Webui pills will be empty for this pipeline's composition steps until the sibling fix lands. Note in PR description.

### R6: Forge agnosticism — `gh pr comment` vs other forges
Mitigation: Use `{{ forge.cli_tool }} {{ forge.pr_command }}` interpolation throughout, matching `ops-pr-review.publish`. No hardcoded `gh` calls.

## Testing Strategy

### Pipeline-level (real run, per acceptance criteria)
- Pick a fresh test PR in re-cinq/wave with multiple known findings.
- Run `wave run ops-pr-respond --input "re-cinq/wave <PR#>" --adapter claude --model cheapest`.
- Verify: comment posted with finding → SHA mapping; loop respected `max_visits: 2`; all primitives fired (check run log for iterate, aggregate, branch, loop events).
- Capture run log + verdict snapshot. Attach to PR per acceptance criteria.

### Schema validation (deterministic, fast)
- Lint `review-findings.schema.json` and `triaged-findings.schema.json` with the existing JSON-schema validator path (consistent with how other contracts are validated in CI).

### Pipeline contract validation (existing test infra)
- Run `go test ./internal/pipeline/...` — covers WLP rule conformance via the existing pipeline-contract validators.
- Verify the new pipelines parse, all `step.input_ref.from` field-nav references resolve, all `iterate.over` typed collections type-check.

### Defaults parity check
- `diff -r .agents/pipelines/ops-pr-respond.yaml internal/defaults/pipelines/ops-pr-respond.yaml` → empty.
- Same for `impl-finding.yaml` and both schemas.

### No-mock policy
Per memory `feedback_real_verification.md`: validation = drive the real CLI surface (`wave run`) against a real PR and read real output. Schema lint + Go tests are necessary but **not sufficient** — only a real `wave run` producing the declared typed output counts as pipeline validation (per `feedback_pipeline_validation_means_run.md`).
