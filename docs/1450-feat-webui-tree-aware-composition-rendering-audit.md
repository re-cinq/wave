## Audit caveats

- **Webui axis produced no screenshots.** Chromedp evidence collection
  did not deposit any files under `.agents/output/screenshots/`, so
  UX-tier gaps below carry their template citations only. Criterion
  #4 of `.agents/contracts/audit-doc-criteria.md` is therefore
  violated by upstream evidence — not by gap selection. The rework
  loop should re-collect screenshots, not re-classify gaps.
- **Evidence merge degenerate.** All four axes returned identical
  copies of the issue body (`evidence_set` length 4, all entries
  equal). Gap selection therefore clusters directly off the 12
  acceptance-criteria checkboxes plus the cited file/route anchors,
  not off independent axis findings. The cross-axis dependency map
  in §Gaps is the audit's actual novel contribution.

# feat(webui): tree-aware composition rendering — implementation per #1412 audit — Audit

> Audit of issue [#1450](https://github.com/re-cinq/wave/issues/1450) by Wave `audit-issue`.

## Summary

The novel finding is the **dependency chain**, not the gap list itself:
the 12 acceptance-criteria items in #1450 are not parallel work — they
form a strict DAG anchored on `g2` (schema migration `add_run_kind_and_iterate_metadata`)
and `g1` (`composition.go:513-578` `runSubPipeline` calling
`store.SetParentRun`). Five WebUI gaps (`g7`, `g8`, `g9`, `g10`, plus
the `5/6 done` rollup inside `g7`) read columns that `g2` introduces
and rows that `g1` populates; none of them can ship green without the
backend pair landing first. `g3` (artifact registration in
`executeAggregate`/`collectIterateOutputs`) is **independent of the
linkage chain** despite touching the same composition path — it
unblocks the OUT pill regardless of tree state, and its partial
fix in PR #1441 should be re-verified before duplicating effort. The
remaining gaps split cleanly: `g5` (`/api/runs/{id}/children` JSON)
and `g4` (`store.GetSubtreeTokens` recursive walk) are independent
backend additions, `g6` (renderer kind switch) is independent
template work, `g11` and `g12` are local polish. The decomposition
recommendation in the issue body matches this DAG; bucket #1
(schema + linkage) is correctly identified as the keystone.

## Severity Map

| Severity | Count |
|----------|-------|
| critical | 0     |
| high     | 7     |
| medium   | 5     |
| low      | 0     |
| info     | 0     |

## Gaps

### G1 — CompositionExecutor.runSubPipeline does not persist parent linkage (high)

- **Axis:** code
- **Citation:** `internal/pipeline/composition.go:513-578`
- **Recommendation:** In `runSubPipeline` at `composition.go:513-578`, call `store.SetParentRun(childRunID, parentRunID, stepID)` and persist `iterate_index`/`iterate_total`/`sub_pipeline_ref` before kicking off the child run. Mirror the existing `executor.go:5505-5507` path that the `executeSubPipelineStep` flow already uses.
- **Cross-axis dependency:** Requires schema columns from G2; downstream consumers G7, G8, G9, G10 cannot render meaningfully until both G1 and G2 land.

### G2 — pipeline_run schema lacks tree/iterate metadata columns (high)

- **Axis:** db
- **Citation:** schema migration `add_run_kind_and_iterate_metadata` on table `pipeline_run`
- **Recommendation:** Add migration `add_run_kind_and_iterate_metadata` introducing columns `iterate_index INTEGER NULL`, `iterate_total INTEGER NULL`, `iterate_mode TEXT`, `run_kind TEXT`, `sub_pipeline_ref TEXT NULL` on `pipeline_run`.
- **Cross-axis dependency:** Required by G1 (writes), G5 (`/children` payload fields), G9 (breadcrumb), G10 (run-kind chip), G12 (filter predicate).

### G3 — executeAggregate / collectIterateOutputs do not register artifacts in the store (high)

- **Axis:** code
- **Citation:** `executeAggregate` and `collectIterateOutputs` in the `internal/pipeline` composition path
- **Recommendation:** Call `store.RegisterArtifact` for each output produced by `executeAggregate` and `collectIterateOutputs`, so the WebUI OUT pill resolves and downstream `inject_artifacts` finds them in the registry rather than only on disk. Verify the state of PR #1441 before duplicating work — it partially addressed this.
- **Independent of linkage chain:** Fix unblocks OUT-pill resolution regardless of G1/G2 tree state.

### G4 — No subtree-token rollup query for run cost aggregation (medium)

- **Axis:** db
- **Citation:** `store.GetSubtreeTokens(rootID)` absent; consumers `/runs` and `/runs/{id}`
- **Recommendation:** Add `store.GetSubtreeTokens(rootID) (int64, error)` implemented as a `WITH RECURSIVE` walk over `parent_run_id`. Wire into the `/runs` list view and `/runs/{id}` detail view to display rolled-up cost.
- **Depends on:** G1+G2 populating `parent_run_id` correctly for the recursion to find children.

### G5 — /api/runs/{id}/children JSON endpoint missing (high)

- **Axis:** code
- **Citation:** `/api/runs/{id}/children` route not registered
- **Recommendation:** Add a dedicated `/api/runs/{id}/children` handler returning `[{run_id, pipeline_name, status, parent_step_id, iterate_index, iterate_total}, ...]` sorted by `iterate_index` then `started_at`. Required by G7 (tree-view child cards and `5/6 done` counts).
- **Depends on:** G2 columns being present in the result rows.

### G6 — Composition step renderer has no cases for iterate/aggregate/loop/branch (high)

- **Axis:** webui
- **Citation:** `templates/run_detail.html:160`, `templates/run_detail.html:172-176`, `partials/step_card.html`
- **Recommendation:** In `templates/run_detail.html:160,172-176` and `partials/step_card.html`, extend the kind switch from `{pipeline, gate, conditional, command}` to also handle `iterate` (`over` / `mode` / `max_concurrent`), `aggregate` (`from` / `into` / `strategy`), `loop` (`max_visits` / `condition`), and `branch` (verdict casing).
- **Independent of linkage chain.** Pure template work, no schema dependency.
- *Screenshot unavailable — see Audit caveats above.*

### G7 — Inline child-card pattern only fires for StepType=="pipeline" (high)

- **Axis:** webui
- **Citation:** `partials/step_card.html` (PR #710 inline-child-card branch)
- **Recommendation:** Generalise the PR #710 inline-child-card branch in `partials/step_card.html` to all composition kinds. Iterate parents must also display child status counts (e.g. `5/6 done`, `2 failed`) on the parent step row, sourced from G5's `/children` endpoint.
- **Cross-axis dependency:** G5 (data source) + G2 (columns) + G1 (rows linked).
- *Screenshot unavailable — see Audit caveats above.*

### G8 — Indent glyph hardcoded to a single nesting level (high)

- **Axis:** webui
- **Citation:** `partials/child_run_row.html:15`
- **Recommendation:** Generalise the indent glyph at `partials/child_run_row.html:15` to ≥2 levels so composition-spawning-composition runs (e.g. `ops-pr-respond` audit children spawning fix children) render correctly. Render breadcrumb `parent → step → item-N` on the child detail page.
- **Cross-axis dependency:** G1 must populate parent linkage at every nesting level for the indent glyph to know its depth.
- *Screenshot unavailable — see Audit caveats above.*

### G9 — Child-run detail page lacks iterate index breadcrumb (medium)

- **Axis:** webui
- **Citation:** `templates/run_detail.html:9`
- **Recommendation:** At `run_detail.html:9` surface `item N/M of <step.id> in <parent_pipeline_name>` when the run has a `parent_run_id` and `iterate_index`/`iterate_total` set.
- **Cross-axis dependency:** G1 (parent_run_id populated) + G2 (iterate_index/iterate_total columns).
- *Screenshot unavailable — see Audit caveats above.*

### G10 — Overview rows have no run-kind chip to filter children from top-level runs (medium)

- **Axis:** webui
- **Citation:** `partials/run_row.html`
- **Recommendation:** Add a run-kind chip (`top-level` / `iterate-child` / `sub-pipeline-child` / `loop-iteration`) to `partials/run_row.html` so operators can filter `audit-*` / `impl-finding` from ad-hoc top-level runs at a glance. Reads `run_kind` column introduced by G2.
- **Cross-axis dependency:** G2 (`run_kind` column).
- *Screenshot unavailable — see Audit caveats above.*

### G11 — ReviewVerdict pill rendered inside the IN row (medium)

- **Axis:** webui
- **Citation:** `templates/run_detail.html:195`
- **Recommendation:** Move the `ReviewVerdict` `<span>` at `run_detail.html:195` out of the IN row into a dedicated pill above the OUT row so it visually adjudicates the run output rather than the input refs.
- **Independent of linkage chain.** Pure template fix, can ship in isolation.
- *Screenshot unavailable — see Audit caveats above.*

### G12 — Pipeline detail recent-runs table flattens sub-pipeline children (medium)

- **Axis:** code
- **Citation:** `handlers_pipelines.go:444-466`
- **Recommendation:** Extend `ListRunsOptions` at `handlers_pipelines.go:444-466` with a `parent_run_id IS NULL` filter (defaulted on for the recent-runs table) so children of sub-pipeline runs do not show alongside top-level runs.
- **Cross-axis dependency:** G1 must populate `parent_run_id` for the filter to actually exclude anything.

## Follow-ups

- **G1** — Set parent linkage from `CompositionExecutor.runSubPipeline` (`composition.go:513-578`).
- **G2** — Schema migration `add_run_kind_and_iterate_metadata` on `pipeline_run`.
- **G3** — Register artifacts in `executeAggregate` / `collectIterateOutputs`; verify against PR #1441.
- **G5** — Add `/api/runs/{id}/children` handler returning child summaries sorted by iterate_index.
- **G6** — Extend renderer kind switch to `iterate` / `aggregate` / `loop` / `branch`.
- **G7** — Generalise inline child-card pattern to all composition kinds with status rollup.
- **G8** — Generalise indent glyph to ≥2 nesting levels with breadcrumb.

(Medium-severity gaps G4, G9, G10, G11, G12 have follow-up specs in
`.agents/output/followup-specs.json`; the file-each-followup step
will file them downstream regardless of severity.)

---
*Generated by Wave `audit-issue` — run id `audit-issue-20260428-121221-a580`.*
