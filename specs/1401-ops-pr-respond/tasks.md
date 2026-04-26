# Work Items

## Phase 1: Setup

- [X] 1.1: Confirm `audit-doc-scan`, `audit-duplicates`, `audit-architecture`, `audit-security`, `audit-tests`, `ops-pr-review-core` all exist and are WLP-clean (each is a sub-pipeline-callable building block).
- [X] 1.2: Confirm aggregate primitive supports `merge_arrays` with cross-source findings (precedent: `ops-parallel-audit.merge-findings`).
- [X] 1.3: Confirm `branch` and `loop` primitives are available in the executor (per memory: composition primitives merged 2026-03-30, issue #677 closed). Decision: linear composition is cleaner for this pipeline; `branch` would require additional micro sub-pipelines and `loop`'s `until` would require a custom verdict-extraction step because `ops-pr-review-core`'s primary output is the diff, not the verdict. impl-finding's contract-level rework_step covers per-finding test-failure retry — the actual feedback loop the issue cares about. Documented inline in `ops-pr-respond.yaml` header.

## Phase 2: Core Implementation

- [X] 2.1: Authored `.agents/contracts/review-findings.schema.json` — array of `{source, severity, file, line, title, description, recommendation}` objects with `additionalProperties: false`. Wrapped in `{findings, total, summary}` envelope. Replaced an orphaned schema of the same name from a prior session (no pipeline referenced it).
- [X] 2.2: Authored `.agents/contracts/triaged-findings.schema.json` — `{actionable: [Finding], deferred: [Finding], rejected: [RejectedFinding+reason], summary}`. Finding shape duplicated inline rather than `$ref`'d to keep cross-schema validation simple.
- [X] 2.3: Authored `.agents/pipelines/impl-finding.yaml` — single craftsman step on a fresh worktree that switches to the PR head branch via `gh pr checkout`, applies one fix, commits with `fix(<scope>): <title> [refs: <source>/<severity>]`, and pushes. Contracts: `non_empty_file` on fix-diff artifact + `test_suite` must_pass on `{{ project.contract_test_command }}` with `rework_step: fix-finding-rework`.
- [X] 2.4: Authored `.agents/pipelines/ops-pr-respond.yaml` with steps:
    - `parallel-review` — iterate parallel `max_concurrent: 3` over six reviewer pipelines.
    - `aggregate-findings` — aggregate `merge_arrays` of `parallel-review.output` into `.agents/output/aggregated-findings.json`.
    - `normalize` — summarizer persona, reshapes aggregated array into `review-findings.schema.json` shape (json_schema must_pass).
    - `triage` — summarizer persona, dual contract: `json_schema` must_pass + `agent_review` warn against `triage-review-criteria.md`.
    - `resolve-each` — iterate **sequential** over `triage.output.actionable` (Rule 7 field-nav), calling `impl-finding` with input `'{"pr_ref": "{{ input }}", "finding": {{ item }}}'`. Sequential mode prevents racing commits on the same PR head branch.
    - `verify` — sub-pipeline `ops-pr-review-core` re-run on the patched PR.
    - `comment-back` — `{{ forge.type }}-commenter` persona, structured PR comment with finding → SHA mapping; contract `json_schema` against `gh-pr-comment-result.schema.json`.
    Also created `.agents/contracts/triage-review-criteria.md` for the agent_review judge.
- [X] 2.5: Mirrored to `internal/defaults/pipelines/ops-pr-respond.yaml`, `internal/defaults/pipelines/impl-finding.yaml`, `internal/defaults/contracts/review-findings.schema.json`, `internal/defaults/contracts/triaged-findings.schema.json`, `internal/defaults/contracts/triage-review-criteria.md` (byte-for-byte parity).

## Phase 3: Validation

- [X] 3.1: `go build ./cmd/wave` — clean.
- [X] 3.2: `go test ./internal/pipeline/...` — pass (29s). `TestAllShippedPipelinesLoad` confirms both new pipelines parse and pass typed-IO validation in both `.agents/pipelines/` and `internal/defaults/pipelines/`.
- [X] 3.3: `go test ./...` — full project suite green.
- [X] 3.4: Defaults parity diff — `diff -q` clean for all four mirrored files (verified inline during 2.5).
- [ ] 3.5: **Real run validation** (mandatory per acceptance criteria): pipeline authoring is complete; live `wave run ops-pr-respond` against a re-cinq/wave PR is the operator's call to schedule with appropriate scope and adapter selection. Real-run remains the gate for shipping per project policy.
- [ ] 3.6: Verify the real run posted a structured comment with finding → SHA mapping. Pending 3.5.

## Phase 4: Polish

- [X] 4.1: AGENTS.md Pipeline Selection table — added `ops-pr-respond` (canonical PR review-to-resolution showcase) and `impl-finding` (sub-pipeline; not standalone).
- [X] 4.2: AGENTS.md preamble — bumped fleet count 34 → 36.
- [ ] 4.3: PR description includes the captured run log + verdict snapshot from task 3.5 — pending.
- [ ] 4.4: PR description explicitly notes deferred fixes per issue's "out of scope" section — pending PR composition.
