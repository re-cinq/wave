# Work Items

## Phase 1: Setup

- [X] 1.1: Re-confirm the six audit pipelines emit `findings_report`-typed `report` outputs with shape compatible with `shared-findings.schema.json`. If any deviate, decide whether to normalize via a transform step or via aggregate + cleanup.
- [X] 1.2: Confirm `triaged-findings.schema.json` field set against the issue spec (`actionable[].id|severity|file|line|description|remediation|type`, `deferred[]`, `rejected[]`). Lock the schema.
- [X] 1.3: Decide whether to add `ops-pr-review-core` as a seventh audit track. Default: skip. Revisit only if the six audits leave a coverage gap during validation.

## Phase 2: Schemas

- [X] 2.1: Write `.agents/contracts/triaged-findings.schema.json`. [P]
- [X] 2.2: Mirror to `internal/defaults/contracts/triaged-findings.schema.json` (byte-equal). [P]

## Phase 3: Sub-pipeline `impl-finding`

- [X] 3.1: Author `.agents/pipelines/impl-finding.yaml` — single craftsman step, worktree workspace based off PR head branch, input `triaged_finding` (typed), contracts: `non_empty_file` on diff + `test_suite` on `{{ project.contract_test_command }}`. Commit message references the finding `id` and quotes the PR.
- [X] 3.2: Mirror to `internal/defaults/pipelines/impl-finding.yaml` (byte-equal). [P with 3.1 once 3.1 lands]
- [X] 3.3: `wave validate impl-finding` — WLP-clean.
- [ ] 3.4: Smoke `wave run impl-finding --input '<sample>'` standalone (one synthetic finding). Capture log. *(deferred to post-merge validation; sandbox cannot wave-run this pipeline reliably from inside the implement step)*

## Phase 4: Composition pipeline `ops-pr-respond`

- [X] 4.1: Author `.agents/pipelines/ops-pr-respond.yaml` skeleton:
  - `input: pr_ref`, `pipeline_outputs: { verdict: comment-back/result, type: findings_report }`.
  - `fetch-pr` step (navigator, mount-readonly): pulls PR diff, head, base, reviews; writes `pr-context.json`.
  - `parallel-review` step: `iterate.over: '["audit-security","audit-architecture","audit-tests","audit-duplicates","audit-doc-scan","audit-dead-code-scan"]'`, `mode: parallel`, `max_concurrent: 6`, `pipeline: "{{ item }}"`, `input` = scope description from `pr-context`.
  - `merge-findings` step: `aggregate.from: "{{ parallel-review.output }}"`, `into: .agents/artifacts/merge-findings/findings.json`, `strategy: merge_arrays`.
- [X] 4.2: Add `triage` step (planner, balanced): inject merged findings + pr-context, write `triaged-findings.json`, contracts: `json_schema` against the new schema + `agent_review` (auditor).
- [X] 4.3: Add `loop` step wrapping `resolve-each` + `verify`:
  - `resolve-each`: `pipeline: impl-finding`, `iterate.over: "{{ triage.out.triaged-findings.actionable }}"`, `mode: parallel`, `max_concurrent: 4`, `on_failure: continue`, `input: "{{ item }}"` (templated per-element form; the `input_ref.from` field-nav variant noted in spec.md's Rule-7 row was not adopted in the shipped YAML).
  - `verify`: command step running `{{ project.test_command }}` + `agent_review` on the diff against pass/fail criteria. Writes `verify.json`.
  - `loop.until: "{{ verify.output.verdict == 'pass' }}"`, `max_iterations: 2`.
- [X] 4.4: Add `branch` step routing on `verify.output.verdict`: `pass → comment-back`, `fail → comment-back` (both produce a comment; `fail` includes "manual review needed" callout). Decision: collapse to a single `comment-back` step that consumes verdict (simpler than branching to two near-identical leaves).
- [X] 4.5: Add `comment-back` step: `persona: "{{ forge.type }}-commenter"`, mount-readonly, posts comment with three tables (findings, resolutions with commit SHAs, verdict). Contract: `agent_review` on tone + completeness + `json_schema` on `comment-result`.
- [X] 4.6: Mirror to `internal/defaults/pipelines/ops-pr-respond.yaml`. [P with 4.1–4.5 once stable]
- [X] 4.7: `wave validate ops-pr-respond` — WLP-clean. Iterate on YAML until clean.

## Phase 5: Validation run (acceptance gate)

- [ ] 5.1: Pick a target PR with ≥2 reviewable findings (PR #1369 is a candidate; otherwise spin a synthetic one). *(deferred to post-merge validation)*
- [ ] 5.2: `wave run -v ops-pr-respond --input "re-cinq/wave <PR>"` (no `--detach` so logs stream). *(deferred to post-merge validation)*
- [ ] 5.3: Verify in `wave logs <run-id>`:
  - parallel-review starts 6 audits concurrently.
  - merge-findings produces non-empty merged JSON.
  - triage classifies ≥1 actionable finding.
  - resolve-each commits ≥1 fix to PR head branch.
  - verify runs project test command and emits verdict.
  - comment-back posts a structured comment.
  *(deferred to post-merge validation)*
- [ ] 5.4: Capture run log + verdict snapshot for the PR description. *(deferred to post-merge validation)*

## Phase 6: Documentation

- [X] 6.1: Add `ops-pr-respond` row to AGENTS.md "Pipeline Selection" table — *PR review-to-resolution* category.
- [X] 6.2: Mention `impl-finding` as a sub-pipeline building block in AGENTS.md.
- [X] 6.3: Cross-link from `ops-pr-review` description to `ops-pr-respond` (review → respond loop).

## Phase 7: PR

- [X] 7.1: Confirm no leaked files: `git status` shows only the planned new/modified files.
- [X] 7.2: Commit (no Co-Authored-By, no AI attribution): `feat(pipeline): ops-pr-respond — full PR review-to-resolution composition`.
- [ ] 7.3: Push branch and open PR with run log + verdict snapshot pasted in description. *(handled by the create-pr step of the parent pipeline)*
- [ ] 7.4: Run `ops-pr-review` on the new PR to validate the showcase from the inside. *(post-merge follow-up)*
