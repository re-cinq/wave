# feat(pipelines): audit-issue — composition showcase for read-only audit work (multi-axis evidence + per-gap fan-out + auto follow-up filing)

**Issue**: [re-cinq/wave#1442](https://github.com/re-cinq/wave/issues/1442)
**Author**: nextlevelshit
**Labels**: enhancement
**State**: OPEN

## Context

Today, audit-only issues (e.g. #1412 "audit webui composition rendering") get routed to `impl-issue` — the default issue→PR pipeline. `impl-issue` is shaped for code → PR delivery, not analysis → doc + screenshots + follow-up issue. Result on PR #1440 (the impl-issue execution of #1412):

- Audit doc landed (319 lines, file:line tables, gap inventory) ✓
- No screenshots (issue acceptance demanded "annotated screenshots of each broken view") ✗
- Follow-up impl issue not filed (drafted in body only — never reached `gh issue create`) ✗
- Speckit `specs/<branch>/{spec,plan,tasks}.md` boilerplate (50+ lines) shipped on a doc-only deliverable ✗
- "10 each" gap padding (model hit the count target with hairsplit findings U9/U10) ✗

The bottleneck is the pipeline, not the prompt or the model. Wave is a building-blocks toolkit; the right answer is a new composition that mirrors `ops-pr-respond` but for audit-only work.

## Proposal: `audit-issue` composition pipeline

```
input: issue_ref
       │
       ▼
fetch-issue (navigator, mount-ro)                       [typed I/O: issue_ref → issue_context]
       │
       ▼
parallel-evidence  (iterate.parallel max=4, 4 sub-pipelines)
  ├── audit-webui-shots    (browser adapter, chromedp screenshots)
  ├── audit-code-walk      (navigator, file:line evidence tables)
  ├── audit-db-trace       (command, sqlite query for runtime examples)
  └── audit-event-trace    (analyst on event_log for behavioural examples)
       │
       ▼
aggregate-evidence  (aggregate.merge_jsons → evidence.json)
       │
       ▼
enumerate-gaps  (planner, json_schema contract)         [severity rubric, no count target]
       │
       ▼ iterate.parallel max=3 over gaps[]
per-gap-deepdive  (sub-pipeline: gap-analyze)           [refines severity + draft remediation]
       │
       ▼
synthesize  (analyst + agent_review handover)           [novel-signal-first criteria]
       │
       ▼ branch on verdict
     pass → file-each-followup
     warn → revise (loop, max_visits=2)
       │
       ▼
file-each-followup  (iterate.serial, command: gh issue create)  [one issue per high-severity gap]
       │
       ▼
create-pr  (implementer, commit doc + screenshots, gh pr create)
```

## Wave primitives lit

Mirrors `ops-pr-respond` for the audit shape; every primitive in one workflow:

| Primitive | Where |
|---|---|
| sub-pipeline | `parallel-evidence` × 4, `per-gap-deepdive` × N |
| iterate.parallel | evidence + per-gap deepdive |
| iterate.serial | file-each-followup |
| aggregate (merge_jsons) | evidence consolidation |
| branch | synthesize verdict routing |
| loop (max_visits) | revise on warn |
| agent_review | synthesize + per-gap-deepdive judges |
| json_schema contract | gap-set, evidence, followup-spec |
| test_suite contract | revise step (audit-doc passes acceptance) |
| typed I/O | issue_ref → evidence_set → gap_set → audit_doc → followup_refs |
| forge interpolation | `{{ forge.cli_tool }}` for `gh` |
| concurrency caps | max_concurrent on both iterates |
| workspace modes | mount-ro for evidence, worktree for create-pr |
| browser adapter | audit-webui-shots |
| event_log inspection | audit-event-trace (ontology hook) |
| multi-persona | navigator, planner, analyst, reviewer, implementer |

## New building blocks

### Pipelines (~6 new YAMLs)

1. **`audit-issue.yaml`** — composition (above). `internal/defaults/pipelines/` + `.agents/pipelines/`.
2. **`audit-webui-shots.yaml`** — single-step sub-pipeline. Browser adapter, chromedp. Reads list of route paths from `issue_context.cited_routes`, captures full-page PNGs to `.agents/output/screenshots/<slug>.png`.
3. **`audit-code-walk.yaml`** — single-step. Navigator persona, mount-ro. Walks code paths cited in issue, emits `code-evidence.json` (file:line + excerpt + role).
4. **`audit-db-trace.yaml`** — single-step. Command type. Runs sqlite queries against `.agents/state.db` to surface concrete runtime examples (e.g. `SELECT … WHERE parent_run_id IS NULL`). Output: `db-evidence.json`.
5. **`audit-event-trace.yaml`** — single-step. Analyst persona. Reads `event_log` rows for the empirical baseline run cited in the issue, extracts behavioural patterns. Output: `event-evidence.json`.
6. **`gap-analyze.yaml`** — single-step. Analyst persona. Takes one gap (JSON blob) from `enumerate-gaps`, deepens severity assessment, drafts the remediation spec (the body of the follow-up issue). Analog of `impl-finding`.

### Personas

- **`webui-capturer`** (NEW) — chromedp tool allowlist only (no Read/Write/Bash beyond adapter). Used by `audit-webui-shots`.

### Schemas

- `evidence.schema.json` — unified evidence shape across the four axes.
- `gap-set.schema.json` — `{gaps: [{id, title, severity, citation, recommendation}]}`.
- `followup-spec.schema.json` — `{title, body, labels[], acceptance[]}`.

### Criteria

- `audit-doc-criteria.md` — agent_review criteria for `synthesize`. Enforces:
  - Lead with novel signal (≤25% recap of issue body)
  - Severity rubric (critical / high / medium / low / info) — no count target
  - Per-gap recommendation must include file:line edit anchor
  - Screenshots inlined for any UX-tier gap
  - Follow-up spec emitted (not just drafted in body)

## Acceptance Criteria

- [ ] `audit-issue` ships in `internal/defaults/pipelines/` + `.agents/pipelines/`.
- [ ] All 5 sub-pipelines ship in both locations.
- [ ] 3 schemas land in `internal/defaults/contracts/` + `.agents/contracts/`.
- [ ] `webui-capturer` persona ships.
- [ ] `audit-doc-criteria.md` ships.
- [ ] WLP-clean (Rules 1-7).
- [ ] Real `wave run audit-issue --input "<owner>/<repo>#<num>"` against #1412 (or another audit-only issue) completes end-to-end and produces:
  - `docs/<slug>-audit.md` with inlined screenshots
  - One follow-up `gh issue create` per high-severity gap (issue URLs captured)
  - PR opened linking the audit doc + follow-up issues
- [ ] AGENTS.md "Pipeline Selection" table updated with `audit-issue` row.

## Out of scope

- `webui-capturer` persona's full tool allowlist contract (chromedp adapter integration may need a new sandbox profile).
- Generalising the pattern to `audit-pr` (audit on a PR diff rather than an issue) — natural follow-up once `audit-issue` proves out.

## Empirical baseline

- PR #1440 (impl-issue on #1412) — 60% acceptance, no screenshots, no follow-up issue filed. The doc itself is solid (file:line citations correctly identify Path A vs Path B in `composition.go:513-578`), proving the analysis quality is there — what's missing is the pipeline shape.
- ops-pr-respond demonstrates the same composition pattern works at scale (26 sub-pipelines on one parent).

## Source

Session 2026-04-27: ran impl-issue on #1411, #1412, #1041, #1434 in parallel. PR #1440 (#1412 audit) shipped 60% of acceptance because the pipeline could not capture screenshots or open a follow-up issue. Discussion confirmed the gap is structural, not authorial.
