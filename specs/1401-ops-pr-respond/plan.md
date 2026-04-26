# Implementation Plan: ops-pr-respond

## 1. Objective

Author a new composition pipeline `ops-pr-respond` that takes a PR ref, fans out a parallel multi-axis review, triages findings, applies per-finding fixes to the PR's head branch, verifies, and posts a structured response comment. Pipeline doubles as the canonical Wave-feature showcase (every primitive lit up).

## 2. Approach

Compose the pipeline from existing building blocks plus one new sub-pipeline (`impl-finding`) and two new schemas. No new personas. Reuse `ops-pr-review-core` for the PR-aware verdict track; reuse five `audit-*` pipelines for breadth; aggregate via the `aggregate` primitive; triage via a `planner` step; iterate-parallel `impl-finding` over actionable findings; verify via `command + agent_review`; route via `branch`; loop back through `loop` (`max_visits: 2`) on verify-fail; post via `{{ forge.type }}-commenter`.

WLP-clean from day one: typed `pipeline_outputs`, canonical `.agents/artifacts/<step>/<output>.<ext>` paths, `input_ref:` for sub-pipeline handovers where typed, deterministic `on_failure` values, no `retry` contract outcomes, no path templating in contracts.

## 3. File Mapping

### Created

- `.agents/pipelines/ops-pr-respond.yaml` — the composition pipeline (~535 lines; ~2× the original ~250-line estimate because per-step prompt bodies grew during contract design).
- `.agents/pipelines/impl-finding.yaml` — single-step craftsman sub-pipeline that fixes one finding on an existing PR branch (~80 lines).
- `.agents/contracts/triaged-findings.schema.json` — `{actionable, deferred, rejected}` schema.
- `.agents/contracts/pr-review-findings.schema.json` — unified findings array shape produced by aggregate. (Avoids name collision with existing `review-findings.schema.json` which describes a PR-review IO bundle, not a flat findings array.)
- `internal/defaults/pipelines/ops-pr-respond.yaml` — embedded twin.
- `internal/defaults/pipelines/impl-finding.yaml` — embedded twin.
- `internal/defaults/contracts/triaged-findings.schema.json` — embedded twin.
- `internal/defaults/contracts/pr-review-findings.schema.json` — embedded twin.
- `specs/1401-ops-pr-respond/{spec,plan,tasks}.md` — planning artifacts (this directory).

### Modified

- `AGENTS.md` — append entry for `ops-pr-respond` to the **Pipeline Selection** table; mention `impl-finding` as a building block.

### Not modified

- No persona changes (decision: reuse `planner` for triage; reuse existing `{{ forge.type }}-commenter`).
- No executor / type-system changes — every primitive (`iterate`, `aggregate`, `branch`, `loop`) already exists.
- No schema migrations — only additive schema files.

## 4. Architecture Decisions

### A. Six audit fan-out track
Use `audit-security`, `audit-architecture`, `audit-tests`, `audit-duplicates`, `audit-doc-scan`, `audit-dead-code-scan`. Drop `ops-pr-review` from the parallel set (it publishes comments — would emit a premature verdict to the PR). Wire `ops-pr-review-core` as the seventh audit if PR-aware multi-track review is needed; otherwise the six audits cover the surface and `ops-pr-review-core` runs separately as a dedicated PR track. **Decision:** start with six audit-* pipelines (matches `findings_report` shape uniformly via `shared-findings.schema.json`); add `ops-pr-review-core` only if validation surfaces a coverage gap.

### B. Schema naming
Existing `review-findings.schema.json` already describes a PR-review-input bundle (`pr_number`, `head_branch`, `findings`, `fix_plan_summary`). New schema for the aggregate output is a flat findings array — name it `pr-review-findings.schema.json` to avoid clobbering. The aggregate output conforms to (a superset of) `shared-findings.schema.json` — option B: reuse `shared-findings.schema.json` directly. **Decision:** reuse `shared-findings.schema.json` for the aggregate output; only add `triaged-findings.schema.json`. This drops two new schema files (`pr-review-findings.schema.json` and its embedded twin) and reduces surface.

### C. impl-finding workspace
Each finding fix runs in a worktree off the **PR head branch** (not `main`). The `impl-finding` sub-pipeline checks out `head_branch` (passed via `input_ref.from: fetch-pr.out.pr-context.head_branch`), applies the fix, commits, then completes — leaving each commit on the head branch in the parent worktree. **Caveat:** parallel iterate runs each finding in its own worktree, so commit ordering is post-merge serial. The verify step pulls the head branch and runs the test suite on the union.

### D. Loop semantics
The issue's diagram shows `verify` → branch → `resolve-each` (loop max_visits: 2). Wave's `loop` primitive runs sub-steps until `until` template evaluates true OR `max_iterations` reached. Encode the loop as: top-level `loop` step containing `resolve-each` + `verify`, exit when `verify.out.verdict.pass == true`, max_iterations: 2. The `branch` primitive then routes the post-loop verdict to `comment-back`.

### E. Triage step
`planner` persona, `model: balanced`. Reads aggregated findings (injected as `findings`) and the diff context (injected as `pr_context`). Produces `triaged-findings.json` matching `triaged-findings.schema.json`. Each actionable finding carries an `id`, `severity`, `file`, `line`, `description`, `remediation`, and `type` (`fix|delete|wire`). The `id` is the iterate key so impl-finding can address each independently.

### F. forge interpolation
The publish step uses `persona: "{{ forge.type }}-commenter"`. Falls through to `github-commenter` etc. The post command uses `{{ forge.cli_tool }} {{ forge.pr_command }} comment <PR_NUMBER> --body-file …`.

## 5. Risks

| # | Risk | Likelihood | Mitigation |
|---|------|-----------|-----------|
| 1 | Parallel `impl-finding` workers race on the same files | Med | Triage emits `groups` field — findings touching same file collapse into one actionable item. Otherwise, fall back to sequential mode in `iterate`. |
| 2 | Sub-pipeline OUT-pill missing (out-of-scope bug) hides progress in webui | Low | Acknowledged out of scope; pipeline still runs end-to-end; webui visibility lands in a separate fix. |
| 3 | `loop` + `branch` are rarely-used primitives; integration may surface bugs | Med | Validation run (acceptance criterion) is the gate. If loop/branch break, capture as separate bug, fall back to a flat retry chain. |
| 4 | Six audit pipelines × 1 verify × N fixes blows token budget | Med | `model: cheapest` on triage / publish; `max_concurrent: 6` on parallel-review caps wall-clock; loop limited to `max_iterations: 2`. |
| 5 | `findings` schema mismatch across audits (some use `shared-findings`, some bespoke) | Med | Audit a sample at impl time; if any audit deviates, normalize via an aggregate `strategy: merge_arrays` + a sanitizing transform step before triage. |
| 6 | Reusing `planner` for triage may produce verbose output overshooting `agent_review` token budget | Low | Tune `token_budget: 8000` on triage agent_review; cap finding count via instructions ("max 25 actionable"). |
| 7 | Branch name `1401-ops-pr-respond` already in use by another worktree | Low | Use `1401-ops-pr-respond-v2` for git operations (already done); spec dir keeps `specs/1401-ops-pr-respond/` per assessment. |

## 6. Testing Strategy

### Unit
None — pipelines are declarative YAML; existing executor coverage applies. The two new schema files are validated by Wave's load-time JSON-schema check.

### Load-time
- `wave validate ops-pr-respond` — exercises WLP rules 1–7. Must pass.
- `wave validate impl-finding` — same.
- Embedded-twin parity: `internal/defaults/pipelines/ops-pr-respond.yaml` byte-equal to `.agents/pipelines/ops-pr-respond.yaml` (sync check enforced by existing `internal/defaults` tests).

### Integration / Acceptance
- Real `wave run ops-pr-respond --input "<owner>/<repo> <pr>"` against a fresh PR with ≥2 reviewable findings. Verify:
  - parallel-review fans out 6 audits in parallel (check `wave logs <run-id>` for concurrent step starts).
  - aggregate produces non-empty merged findings.
  - triage classifies ≥1 actionable.
  - resolve-each commits ≥1 fix to head branch.
  - verify runs `{{ project.test_command }}` and emits a verdict.
  - branch routes to comment-back on pass.
  - comment-back posts a structured comment with finding-→-SHA mapping.
- Confirm the PR comment URL appears in the run output.

### Smoke
- `wave run impl-finding --input '<sample finding JSON>'` standalone — confirms sub-pipeline composes outside `ops-pr-respond`.

## 7. Out-of-scope (filed separately)

- Webui badge relocation.
- Sub-pipeline OUT-pill DB registration.
- Quality-review token budget bump.
