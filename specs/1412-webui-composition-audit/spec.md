# audit(webui): composition runs render flat — overview + detail views need tree-aware redesign

**Issue:** [#1412](https://github.com/re-cinq/wave/issues/1412)
**Repository:** re-cinq/wave
**Labels:** enhancement
**State:** OPEN
**Author:** nextlevelshit

## Context

Running `ops-pr-respond` on PR #1407 spawns:

- 1 parent (`ops-pr-respond`)
- 6 audit sub-pipelines (`parallel-review` step → `iterate.parallel`)
- 19 fix sub-pipelines (`resolve-each` step → `iterate.parallel` over `impl-finding`)
- = **26 concurrent runs**, all visible in the WebUI Running list

Screenshot evidence (this session): the Running tab displays a flat list of 14+ rows where every `impl-finding` and `audit-*` row looks identical to a top-level run. There is no visual hierarchy showing "these belong to ops-pr-respond, parallel-review step, item 7/19". You cannot tell which audits or fixes belong to which parent without opening each run individually.

## Problem

The current overview and detail views were designed when most runs were single-pipeline. They break down for compositions:

1. **Overview** — every sub-pipeline appears as an independent run row. The parent's progress (`5/6` in the badge) does not surface which step is running, which children are done, or how many children remain.
2. **Detail (parent)** — the parent run page shows step events but does not embed or link to its child runs in tree form. Operators chase IDs across pages.
3. **Detail (child)** — each `impl-finding` run page does not show "I am child 7/19 of resolve-each in ops-pr-respond-…". Provenance is lost.
4. **No grouping by run kind** — `audit-*` (review fan-out), `impl-finding` (fix fan-out), and ad-hoc child pipelines all render the same shape; the user cannot tell scan-from-fix at a glance.
5. **Token / status totals are not aggregated** — the parent's token counter shows only the parent's tokens (often 0 until late steps), not the rolled-up subtree cost.

## Proposed scope

Audit-first, before redesign. Inventory must answer:

### Inventory

- Every webui surface that lists or details runs (`/runs`, `/runs/<id>`, `/pipelines`, `/pipelines/<name>`, dashboard cards).
- Every place in `internal/webui/` that queries `pipeline_run` joins on `parent_run_id` (or fails to).
- Schema check: does `pipeline_run` carry parent / iterate-index / iterate-total fields needed for tree rendering? If not, what's the minimum addition?
- API surface (`/api/runs`, `/api/runs/<id>/children`, etc.) — does the data layer already expose the tree, or is the missing UX a backend gap?

### Findings to produce

- Concrete list of UI affordances that fail for compositions (with screenshots).
- Concrete list of data gaps (DB columns, API endpoints, store methods) blocking a tree view.
- Mapping: which parts of the redesign can ride on the in-flight #709 / #710 Fat-Gantt-Shapes work, and which need their own change.

### Non-goals (this issue)

- Implementation. This issue is the audit; implementation gets a follow-up issue once the gap list is concrete.
- Changing the underlying executor / composition model.

## Acceptance Criteria

- Audit doc lands at `docs/webui-composition-audit.md` (or comment thread on this issue) with:
  - Annotated screenshots of each broken view.
  - Numbered list of UX gaps (one per finding).
  - Numbered list of data / API gaps.
  - Recommendation: "redesign here", "data layer here", "ride on existing #709 work here".
- Follow-up issue opened for the redesign with clear parent-child / tree-view requirements.

## Source

Empirical baseline: `ops-pr-respond-20260426-203623-b73e` running on PR #1407 (this session's `ops-pr-respond` run produced 26 sub-pipelines, all flat in the Running list).

Related: #709 (pipeline / PR / issue pages WebUI redesign — in progress), #710 (runs done — merged).

## Comment Addendum (nextlevelshit)

### 1. `(no prompt)` rendering on composition steps

The detail view shows `(no prompt)` for every step that delegates to a sub-pipeline:

- `parallel-review` (`iterate.parallel` over six audit-* sub-pipelines)
- `merge-findings` (`aggregate` of the six audit outputs)
- `resolve-and-verify` (`iterate.parallel` over `impl-finding`)

These steps have no `exec.prompt` by design — they are orchestration metadata (iterate config, aggregate strategy, sub-pipeline reference). Rendering them as `(no prompt)` is technically correct but tells the operator nothing useful. Composition-step renderer must surface:

- Sub-pipeline reference (`pipeline: "{{ item }}"`).
- Iterate config (`over`, `mode`, `max_concurrent`).
- Aggregate config (`from`, `into`, `strategy`).
- List of children with progress, with links into each child run page.

### 2. `fail` badge on artifact references that work at runtime

`triage` step shows: `IN fetch-pr/pr-context merge-findings/merged-findings` with a red `fail` badge between the two input refs.

- `merge-findings` is an `aggregate` step. It writes `.agents/output/merged-findings.json` via `aggregate.into`, but does not declare an `output_artifacts:` block.
- `triage.memory.inject_artifacts` references `step: merge-findings, artifact: merged-findings`.
- At runtime injection works fine; planner produced valid `triaged-findings.json`.
- WebUI flags input as `fail` because artifact name is not formally registered as upstream output.

UI-truth bug: lights up `fail` for an injection that succeeded. Two fix paths:
- (a) UI fix — when `inject_artifacts` resolves at runtime, do not show `fail`. Flag should reflect actual injection result, not declarative completeness.
- (b) Schema fix — `aggregate` and `iterate` steps auto-register implicit output artifacts.

### Audit checklist additions

- Inventory which step kinds (`prompt`, `command`, `agent_review`, `iterate`, `aggregate`, `loop`, `branch`, `sub_pipeline`) currently render coherent detail views and which fall back to `(no prompt)`.
- Inventory which `IN/OUT` indicators reflect runtime truth vs declarative completeness.
- Cross-reference `fail` badge logic with `inject_artifacts` resolver — produce list of false-positive cases.

Source: run `ops-pr-respond-20260426-203623-b73e` on PR #1407.
