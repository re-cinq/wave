# Implementation Plan — 1411-scope-audits-pr

## 1. Objective

Reduce `ops-pr-respond` raw-finding noise from ~127 → ≤30 per 10-file PR, lift actionable share from ~15% → ≥60%, and surface design-question deferrals as structured comment output. Achieved by injecting PR scope (`pr-context` artifact + diff) into each `audit-*` sub-pipeline, inserting a deterministic pre-triage scope filter, and extending the triage schema with a `design_questions` channel.

## 2. Approach

The fix is layered: each layer independently tightens the funnel, so partial regressions still help.

1. **Inject PR scope into audits** — `parallel-review` step in `ops-pr-respond.yaml` passes `pr-context` via `config.inject` so each audit child receives `.agents/artifacts/pr-context` (typed JSON with `changed_files`, `diff_path`). Audit prompts gain a conditional preamble: "If `.agents/artifacts/pr-context` exists, restrict findings to `changed_files` and use `diff_path` as scope-of-truth."
2. **Deterministic scope filter** — new `filter-scope` shell step between `merge-findings` and `triage`. Reads `merged-findings.json` + `pr-context`, drops any finding whose `file` is not in `changed_files`, emits `scoped-findings.json` + `scope-filter-stats.json`. No LLM. Pure `jq`.
3. **Triage rewires** — `triage` step depends on `filter-scope` (not `merge-findings`); reads `scoped-findings`. Prompt loses the "drop out-of-scope" responsibility and gains "categorize deferred-with-design-question into the new `design_questions` channel". Also short-circuits when input array is empty.
4. **Schema extension** — `triaged-findings.schema.json` gains optional `design_questions[]` (each: `id`, `description`, `question`, `source_finding_id`, optional `suggested_followup`). Backward-compatible (optional field, `additionalProperties: false` already excludes others).
5. **Comment-back render** — adds "Design Questions" section, renders one bullet per `design_questions[]` entry with the question + suggested follow-up. Sanitisation rules + 16 KiB cap unchanged.
6. **Prompt-hardening** — `impl-finding` already injects `pr-context`; remove residual prompt language that suggests re-fetching `gh pr view` / `gh pr diff`. Replace with "the parent already wrote `.agents/artifacts/pr-context`; do NOT re-shell to `gh` for PR metadata."

Audit-side conditional: standalone audit runs (no `pr-context` artifact present) keep current whole-repo behaviour. Detection is a one-line file-existence check in the prompt.

## 3. File Mapping

### Modified

| File | Change |
|------|--------|
| `internal/defaults/pipelines/ops-pr-respond.yaml` | Inject `pr-context` into `parallel-review` children; add `filter-scope` step; rewire `triage.dependencies`; extend `comment-back` prompt with Design Questions section |
| `internal/defaults/pipelines/audit-security.yaml` | Scan-step prompt: PR-scope preamble + diff_path emphasis |
| `internal/defaults/pipelines/audit-architecture.yaml` | Same scope preamble |
| `internal/defaults/pipelines/audit-tests.yaml` | Same scope preamble |
| `internal/defaults/pipelines/audit-duplicates.yaml` | Same scope preamble |
| `internal/defaults/pipelines/audit-doc-scan.yaml` | Same scope preamble |
| `internal/defaults/pipelines/audit-dead-code-scan.yaml` | Same scope preamble |
| `internal/defaults/pipelines/impl-finding.yaml` | Strip redundant `gh pr view`/`gh pr diff` instructions; rely on injected `pr-context` |
| `internal/defaults/contracts/triaged-findings.schema.json` | Add optional `design_questions[]` array |
| `.agents/pipelines/ops-pr-respond.yaml` | Mirror of internal/defaults |
| `.agents/pipelines/audit-*.yaml` (×6) | Mirrors of internal/defaults |
| `.agents/pipelines/impl-finding.yaml` | Mirror |
| `.agents/contracts/triaged-findings.schema.json` | Mirror |

### Created

| File | Purpose |
|------|---------|
| `internal/defaults/contracts/scope-filter-stats.schema.json` | Schema for `filter-scope` step's stats output (per-axis kept/dropped counts, total) |
| `.agents/contracts/scope-filter-stats.schema.json` | Mirror |
| `specs/1411-scope-audits-pr/spec.md` | (already created) Issue capture |
| `specs/1411-scope-audits-pr/plan.md` | This file |
| `specs/1411-scope-audits-pr/tasks.md` | Phased breakdown |

### Deleted

None.

## 4. Architecture Decisions

### D1: Inject `pr-context` artifact, do not encode PR scope into input string
The parent could pass `pr_context.changed_files` as part of the `parallel-review.input` string. We inject the typed artifact instead. Reasons: (a) audits can stream `diff_path` instead of inlining the diff, (b) typed contract prevents drift, (c) standalone audits stay parameter-shaped.

### D2: Deterministic filter step (shell + jq), not an LLM step
Filter is a pure set-membership operation: `{f.file ∈ changed_files}`. An LLM here adds latency, cost, and non-determinism for zero benefit. Implemented as a `command` step, not a `prompt` step.

### D3: `design_questions[]` array over follow-up issues
Issue text proposes either inline section OR auto-filed follow-up issues. Choose inline. Reasons: (a) inline keeps the workflow self-contained — a reviewer reads one comment and knows everything, (b) auto-filing issues per deferral generates issue-tracker noise that the maintainer has to garbage-collect, (c) any human can promote an inline question to an issue, but auto-filed issues are hard to retract.

### D4: Conditional audit-side scoping (file presence detection)
Audit prompts read `.agents/artifacts/pr-context` if present, otherwise scan whole repo. No new pipeline `input` parameter — keeps audit pipeline contracts unchanged for standalone use. Standalone audit runs have no `pr-context` artifact, so the conditional preamble is a no-op.

### D5: Filter step drops by `file ∉ changed_files`, not by symbol-in-diff
Issue suggests two filters: file-set membership AND "cited symbol does not appear in the diff". Symbol matching needs a tokenizer / language-aware parser. File-set membership catches >95% of the noise per the empirical baseline (102/127 rejected were out-of-scope by file). Defer symbol-level filtering to a follow-up if the file filter alone doesn't hit ≤30 raw findings.

### D6: Schema extension is additive + optional
`design_questions[]` is optional. Existing consumers (everything reading `triaged-findings.json`) ignore unknown optional fields. The wave versioning policy (no backward-compat shims pre-1.0) does not block additive optional fields — it blocks rename/deprecation drift.

## 5. Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| Audit prompts reading non-existent `pr-context` file in standalone runs crash the prompt | medium | Guard with `if [ -f .agents/artifacts/pr-context ]` style conditional in the prompt body — no Bash; instruction-level guard the LLM must follow |
| Schema change breaks `triaged-findings.json` consumers | low | New field is optional; add it after existing properties; run schema validation against representative fixtures |
| `filter-scope` step strips legitimate findings whose `file` is e.g. `<NEW_FILE_NOT_IN_CHANGED>` (rare edge: changed_files derived from `gh pr view` excludes renames-only) | medium | Keep the dropped findings in the stats output for audit; if non-zero in real runs, revisit |
| Comment body ≥16 KiB after Design Questions section | low | Existing belt-and-braces `head -c 16384` enforces; sanitisation rules already drop deferred/rejected first under pressure — extend to drop design_questions before that |
| Mirroring drift between `internal/defaults/` and `.agents/` | medium | Apply same edit to both; verify with `diff -r` after edits; tests should compare both paths |
| `impl-finding` prompt change breaks resolved-PR flow already in production runs | low | Change is additive removal of redundant `gh` calls; the injected `pr-context` path is already authoritative per current code |

## 6. Testing Strategy

### Unit / contract level
- Validate `triaged-findings.schema.json` accepts examples with and without `design_questions[]`. New fixture under `internal/defaults/contracts/_fixtures/` (or wherever existing schema tests live).
- Validate `scope-filter-stats.schema.json` against a synthesised stats blob.
- Lint all touched YAMLs with the existing pipeline manifest validator (`go test ./internal/pipeline/... -run TestPipelineLoad`).

### Integration / pipeline-level
- Re-run `ops-pr-respond` against a small synthetic PR (or PR #1407 itself) and confirm:
  - `merged-findings.json` count: baseline ~127.
  - `scoped-findings.json` count: ≤30.
  - `triaged-findings.json` actionable / total ≥ 60%.
  - `design_questions[]` populated for known design-deferral cases.
  - Posted PR comment renders the Design Questions section.
- Standalone audit runs: invoke each `audit-*` pipeline alone (no `pr-context`); confirm whole-repo scan still happens and findings count is unchanged from baseline.

### Regression
- Existing pipeline-load tests, persona/skill validation, schema regex tests must remain green.
- Confirm forge-template variables still interpolate after edits (`{{ forge.cli_tool }}` etc.).
- Confirm `comment-back`'s 16 KiB cap holds with the new section appended.

### Manual / smoke
- `wave run ops-pr-respond <pr-url>` end-to-end on a real PR (the validation rule: real run, real output, not just contract pass).
- Inspect resulting comment markdown by hand.
