# Implementation Plan — #1412 WebUI Composition Audit

## Objective

Produce a written audit of how WebUI surfaces (overview, run detail, pipeline detail) render composition pipelines (parent + iterate/aggregate/sub_pipeline children), enumerating UX gaps and data-layer gaps blocking a tree-aware redesign. Read-only deliverable; no executor or schema changes in this issue.

## Approach

Five-stage audit, code-only inspection (no live reproduction required since the issue body and comment already supply empirical baseline + screenshots-by-description from `ops-pr-respond-20260426-203623-b73e`):

1. **Inventory webui surfaces** — every handler, template, and partial that lists or details runs/pipelines. Capture current rendering shape per step kind.
2. **Inventory data layer** — schema columns on `pipeline_run` (parent, iterate index/total), store query methods, API endpoints exposing parent/child relationships.
3. **Map UX gaps to data gaps** — per numbered problem in the issue, identify whether the gap is purely view-layer (data exists, not surfaced) or backend (data missing).
4. **Reconcile with #709/#710** — read in-flight Fat-Gantt-Shapes specs/PRs to label which gaps already have proposed fixes versus which need their own change.
5. **Write `docs/webui-composition-audit.md`** — annotated findings with numbered UX-gap list, numbered data/API-gap list, recommendation table mapping each gap to {redesign-here, data-layer-here, ride-on-#709}, and follow-up-issue draft for the redesign.

## File Mapping

### Created

- `docs/webui-composition-audit.md` — main deliverable. Sections: Inventory, UX Gaps (numbered), Data/API Gaps (numbered), Step-Kind Renderer Matrix, IN/OUT Indicator Truth Table, Recommendations, Follow-up Issue Draft.
- `specs/1412-webui-composition-audit/spec.md` — issue mirror (this artifact).
- `specs/1412-webui-composition-audit/plan.md` — this file.
- `specs/1412-webui-composition-audit/tasks.md` — work breakdown.

### Read-only (audit targets, no edits)

- `internal/webui/handlers_runs.go` — list + detail handlers, parent_run_id queries.
- `internal/webui/handlers_pipelines.go` — pipeline page run listings.
- `internal/webui/types.go` — view models for runs (ParentRunID surface).
- `internal/webui/templates/runs.html`, `run_detail.html`, `pipelines.html`, `pipeline_detail.html` — render shape.
- `internal/webui/templates/partials/` — step renderers, run rows.
- `internal/webui/run_stats.go` — token/cost aggregation logic.
- `internal/state/types.go`, `internal/state/store.go`, `internal/state/migration_definitions.go` — schema for `pipeline_run` (parent_run_id, iterate fields).
- `internal/pipeline/iterate.go`, `internal/pipeline/aggregate.go`, `internal/pipeline/sub_pipeline.go` — child run creation, parent linkage, iterate index/total propagation.
- `internal/pipeline/composition.go` (if exists) — composition primitives wiring.
- `specs/772-webui-running-pipelines/`, `specs/585-sub-pipeline-composition/` — prior plans referencing iterate/parent fields.
- Open PRs / specs for #709, #710 — verify what's already in flight.

### No deletions, no schema migrations, no handler changes

## Architecture Decisions

- **Audit-only output**: deliverable is markdown under `docs/`. No code changes ship under this issue (per non-goals). Implementation lands in a follow-up issue scoped from this audit's gap list.
- **Doc location chosen over issue comment**: per acceptance criteria the doc is the canonical artifact; comment thread would scatter findings.
- **Screenshot substitute**: live screenshot reproduction is not in worktree scope; instead the doc cross-references the `ops-pr-respond-20260426-203623-b73e` baseline and describes each broken view by handler + template path so a reviewer can reproduce. Real screenshots can be added to the doc post-merge if useful.
- **Step-kind renderer matrix**: instead of prose per kind, produce a single table with rows = {`prompt`, `command`, `agent_review`, `iterate`, `aggregate`, `loop`, `branch`, `sub_pipeline`} and columns = {detail-template-path, current-rendering, what-it-misses}.
- **IN/OUT truth table**: rows = step kinds that produce implicit artifacts (iterate.collected-output, aggregate.into), columns = {declared-in-output_artifacts, resolved-at-runtime, ui-badge-shown}. Documents the false-positive `fail` badge case.

## Risks

- **Risk: audit findings drift from current code by the time follow-up implementation lands.** Mitigation: include git SHAs in the doc for handlers and templates audited; reviewer can rebase.
- **Risk: #709 in-flight changes may already obsolete some gaps.** Mitigation: explicitly read the latest #709 PR/spec before writing recommendations and label each gap with "may overlap #709".
- **Risk: empirical baseline run is no longer reproducible (logs may have rotated).** Mitigation: rely on description in issue + comment; do not re-run `ops-pr-respond` to reproduce (out of scope, expensive).

## Testing Strategy

No code change → no automated test coverage required. Validation = doc review:

- Spell-check / link-check the produced markdown.
- Verify every gap in the doc cites a concrete file path + line range or template name.
- Verify recommendations table covers every numbered gap (UX + data/API).
- Verify follow-up-issue draft is concrete enough to scope (acceptance criteria, file pointers).
