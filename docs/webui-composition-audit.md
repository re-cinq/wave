# WebUI Composition Audit

**Issue:** [#1412](https://github.com/re-cinq/wave/issues/1412) — *audit(webui): composition runs render flat — overview + detail views need tree-aware redesign*
**Audit baseline SHA:** `8a2b9361` (`main`, 2026-04-27)
**Empirical baseline run:** `ops-pr-respond-20260426-203623-b73e` on PR #1407 (1 parent + 6 audit children + 19 fix children = 26 concurrent runs).
**Status:** Read-only audit. No executor, schema, or handler changes ship under this issue (per non-goals). Implementation is scoped in the follow-up issue draft below.

---

## 1. Inventory

### 1.1 WebUI surfaces (handlers + templates)

| Surface | Handler | Template | Purpose |
|---|---|---|---|
| `/runs` (overview / Running list) | `handlers_runs.go:handleRunsPage` (~L242) | `templates/runs.html`, `partials/run_row.html`, `partials/child_run_row.html` | Lists all runs with running-pipelines section on top. Children nested under parents via `nestChildRuns()` (`handlers_runs.go:201`, `:1233–1265`). |
| `/runs/{id}` (detail) | `handlers_runs.go:handleRunDetailPage` (~L451) | `templates/run_detail.html` (no partial for steps; renders inline) | Per-run Fat-Gantt detail. Children fetched via `store.GetChildRuns()` and exposed as `ChildRuns` keyed by `ParentStepID`. |
| `/pipelines/{name}` | `handlers_pipelines.go:handlePipelineDetailPage` (~L444–466) | `templates/pipeline_detail.html`, `partials/step_card.html`, `partials/dag_svg.html` | Pipeline structure + recent runs list. `ListRuns(PipelineName=name, Limit=1000)` with NO `parent_run_id` filter — child runs of sub-pipelines appear flat. |
| `/api/runs` | `handlers_runs.go:handleAPIRuns` | — | Paginated run list. No `parent_run_id` filter / tree query. |
| `/api/runs/{id}` | `handlers_runs.go:handleAPIRunDetail` | — | Single-run JSON. Does not embed child IDs. |
| `/api/runs/{id}/children` | **DOES NOT EXIST** | — | No dedicated endpoint. Children are loaded server-side per page render. |

All routes registered in `internal/webui/routes.go:16–68`.

### 1.2 Schema (`pipeline_run`)

`internal/state/types.go:8–26` (struct `RunRecord`):

```go
type RunRecord struct {
    RunID, PipelineName, Status, Input, CurrentStep string
    TotalTokens int
    StartedAt, CompletedAt, CancelledAt, LastHeartbeat time.Time
    ErrorMessage string
    Tags []string
    BranchName string
    PID int
    ParentRunID  string  // L23 — added migration #14
    ParentStepID string  // L24 — added migration #14
    ForkedFromRunID string
}
```

Migration #14 (`internal/state/migration_definitions.go:329–333`) added `parent_run_id` + `parent_step_id` to the `pipeline_run` table.

**Missing columns** (none of these exist on `pipeline_run`):

- `iterate_index`, `iterate_total`, `iterate_mode` — iterate position metadata
- `run_kind` (top-level / iterate-child / sub-pipeline-child / loop-iteration / aggregate-target)
- `sub_pipeline_ref` — name of pipeline the parent stepped into
- `subtree_total_tokens` — rolled-up cost across descendants

### 1.3 Store query layer

`internal/state/store.go`:

- `SetParentRun(childRunID, parentRunID, stepID string) error` (L2606) — writes `parent_run_id` + `parent_step_id`.
- `GetChildRuns(parentID)` — exists, used by `handlers_runs.go:242, :451`.
- **Missing:** `GetRunTree(rootID)` — recursive subtree fetch.
- **Missing:** `GetSubtreeTokens(rootID)` — rollup query (no `WITH RECURSIVE` walk anywhere in the codebase).

### 1.4 Composition primitives (parent linkage)

**Two divergent paths create child runs**:

**Path A — `executeSubPipelineStep` in `DefaultPipelineExecutor`** (`internal/pipeline/executor.go:5502–5560`): correctly links parent.

```go
// executor.go:5505–5507
childRunID := e.createRunID(pipelineName, 4, input)
childOpts = append(childOpts, WithRunID(childRunID))
_ = e.store.SetParentRun(childRunID, pipelineID, step.ID)
```

This is the path used when a regular sequential pipeline contains a `sub_pipeline:` step.

**Path B — `CompositionExecutor.runSubPipeline`** (`internal/pipeline/composition.go:513–578`): does **NOT** call `SetParentRun`. Instead it delegates to `SequenceExecutor`:

```go
// composition.go:523
result, err := c.seqExecutor.Execute(ctx, []*Pipeline{p}, c.manifest, input)
// composition.go:576
_ = result   // result discarded — child run ID never returned
```

`SequenceExecutor.executeSinglePipeline` (`internal/pipeline/sequence.go:347`) creates the child via `s.store.CreateRun(pipelineName, input)` and returns only a `PipelineResult` whose `RunID` is never propagated back to the composition executor's call site.

The Composition path is hit by:

- `executeIterate` (`composition.go:137`) → `executeIterateSequential` (L168) / `executeIterateParallel` (L224) → `runSubPipeline` (L204, L272). Both call sites pass `stepID=""`.
- `executeAggregate` (`composition.go:450`) — does not spawn runs but writes `step.Aggregate.Into` to disk WITHOUT registering a `pipeline_artifact` row (no `SaveArtifact()` call between L477 and L482).
- `executeSubPipeline` (`composition.go:497`) → `runSubPipeline` (L507).
- `executeLoop` (`composition.go:368`) — runs sub-steps inline, no child runs.
- `executeBranch` (`composition.go:323`) — chooses one branch sub-step, no child runs.

**Net effect:** every iterate / aggregate / sub-pipeline child spawned by a composition pipeline (i.e. every audit-* and impl-finding child of `ops-pr-respond`) lands in `pipeline_run` with `parent_run_id = ''`. The `nestChildRuns()` filter at `handlers_runs.go:201` walks `ParentRunID` → these rows fall back to top-level and the Running list flattens.

### 1.5 Iterate metadata

Iterate progress is emitted as **events only** (`composition.go:178–185, :263–269`):

```go
c.emit(event.Event{
    State:    event.StateIterationProgress,
    Message:  fmt.Sprintf("item %d/%d", i+1, len(items)),
    Progress: ((i + 1) * 100) / len(items),
})
```

These events are written to `pipeline_event` rows, not to `pipeline_run`. Reconstructing "this child is item 7 of 19" from events requires joining child run start time against parent step events — fragile and not done anywhere.

### 1.6 Output artifact registration (declarative vs runtime)

- **Declarative**: `step.OutputArtifacts []ArtifactDef` (`internal/pipeline/types.go`). Persisted into the `artifact` table by the executor's writeOutputArtifacts pass.
- **Runtime injection**: `step.Memory.InjectArtifacts` lists `(step, artifact)` references resolved at run start. Resolution happens in the executor; success is observable only via the file actually existing in the workspace.
- **Aggregate's `into:` path** is written by `os.WriteFile` (`composition.go:477`) but is **never registered** as an artifact row. The downstream `inject_artifacts` reference resolves at runtime against the filesystem and works, but the WebUI cannot see the upstream artifact in the artifact table.

### 1.7 IN/OUT badge + ReviewVerdict (the "fail" badge symptom)

`templates/run_detail.html:192–198` (the IO row):

```html
<span class="ws-io-lbl ws-io-click" onclick="toggleIn(this)">IN</span>
{{if $step.InputArtifacts}}
  {{range $ia := $step.InputArtifacts}}
    <a class="w-tab" data-step-id="{{$ia.Step}}" data-artifact-name="{{$ia.Name}}"
       onclick="toggleArt(this);return false;">{{$ia.Step}}/{{$ia.Name}}</a>
  {{end}}
{{else}}…{{end}}
{{if $step.ReviewVerdict}}<span class="w-ctr w-ctr-{{if eq $step.ReviewVerdict "pass"}}pass{{else}}fail{{end}}">{{$step.ReviewVerdict}}</span>{{end}}
<span class="ws-io-sep"></span>
<span class="ws-io-lbl">OUT</span>
{{if $step.Artifacts}}{{range $a := $step.Artifacts}}…{{end}}{{end}}
```

`InputArtifactRef` (`internal/webui/types.go:99–102`) carries only `{Step, Name}` — no per-ref resolution status, no `Resolved bool`. There is **no fail badge tied to artifact resolution anywhere in the templates**.

The red "fail" pill the operator saw between IN refs is the **`ReviewVerdict` span** (L195) emitted whenever the step's review LLM returned `fail`. The visual placement makes it look like it adjudicates the IN refs — that is the UX bug. The data-truth bug the issue posits ("UI flags input as fail because artifact name is not formally registered as upstream output") does not exist as described; what is real is that the verdict's positioning is ambiguous and `aggregate.into` outputs are invisible to the artifact table.

### 1.8 Step-type discrimination at the handler

`handlers_runs.go:744–751` (the only place `StepType` is set):

```go
stepType := step.Type
if stepType == "" && step.Gate != nil { stepType = "gate" }
if stepType == "" && step.SubPipeline != "" { stepType = "pipeline" }
```

Composition kinds are not detected:

| Step has… | Resulting `StepType` |
|---|---|
| `step.Iterate != nil` and `step.SubPipeline != ""` | `"pipeline"` (iterate folded in) |
| `step.Iterate != nil` and `step.SubPipeline == ""` | `""` (fall through) |
| `step.Aggregate != nil` | `""` |
| `step.Loop != nil` | `""` |
| `step.Branch != nil` | `""` |
| `step.SubPipeline != ""` (plain sub-pipeline) | `"pipeline"` |

Templates only branch on `pipeline | gate | conditional | command` (`run_detail.html:160, 172–176`; `step_card.html:12–16, 28–69`). Iterate / aggregate / loop / branch all render with no badge, no icon, and no body — and the prompt fetcher prints `(no prompt)` (`run_detail.html:379`) since `step.prompt` is empty.

### 1.9 Token aggregation

`run_stats.go:31–36` rolls token counts per *issue / PR*, not per *run-tree*. Inside `handlers_runs.go:348–352` the run-detail handler sums **per-step** tokens into `runSummary.TotalTokens` (which overwrites the parent's stored `TotalTokens` for the page render only). No code path sums child-run tokens into a parent.

### 1.10 In-flight #709 / #710 work

- **PR #710** (Fat-Gantt-Shapes v2, merged Apr 2026): redesigned `/runs/{id}` with IN/OUT hero cards, gantt shapes, child-run inline cards under sub-pipeline steps. Child cards appear **only when the step has `step.SubPipeline != ""`** — i.e., not for iterate/aggregate/loop/branch.
- **PR #711** (#709 detail-page redesign, merged Apr 2026): unified card design across pipelines / issues / PRs / personas / contracts. Pipeline detail's recent-runs list still uses `ListRuns` with no parent filter.
- **`specs/772-webui-running-pipelines/`**: expandable Running section on `/runs`. Child nesting rendered via `partials/child_run_row.html` (single-level indent glyph `└`).
- **None of the merged work** touches: schema columns, `SetParentRun` for composition, iterate-index metadata, run-kind discrimination, subtree-token rollup, or composition step kinds beyond `pipeline`.

---

## 2. UX Gaps (numbered)

| # | Gap | Surface | Citation |
|---|---|---|---|
| **U1** | Composition children of `ops-pr-respond` (audit-*, impl-finding) appear as top-level rows in the Running list because `parent_run_id` is empty in the DB. | `/runs` overview | `composition.go:513–578` (no `SetParentRun`); `handlers_runs.go:201`, `:1233` (nesting walks `ParentRunID`) |
| **U2** | Parent-run detail does not embed/link the spawned child run pages for steps of kind `iterate`, `aggregate`, `loop`, `branch`. Sub-pipeline ones do (PR #710), so coverage is uneven. | `/runs/{id}` | `run_detail.html:160, 172–176`, child cards rendered only when `StepType == "pipeline"` |
| **U3** | Child-run detail page lacks "I am child 7/19 of `resolve-each` in `ops-pr-respond-…`" provenance. Only `ParentRunID` breadcrumb (run_detail.html:9) — no iterate index, no parent step name. | `/runs/{id}` (child) | `types.go:99–102` (no IterateIndex); `run_detail.html:9` |
| **U4** | No grouping by run kind. `audit-*` and `impl-finding` rows render identically to ad-hoc top-level runs. | `/runs`, `/pipelines/{name}` | `RunRecord` has no `RunKind` field (`state/types.go:8–26`); `run_row.html` shows pipeline name only. |
| **U5** | Parent token total shows only the parent's own tokens, not subtree cost — usually zero until late steps. | `/runs/{id}`, `/runs` | `handlers_runs.go:348–352` (per-step rollup, parent only); no `GetSubtreeTokens` query. |
| **U6** | Composition steps render `(no prompt)` because they have no `exec.prompt`. The orchestration metadata (`iterate.over`, `iterate.mode`, `aggregate.from`, `aggregate.into`, `pipeline: "{{ item }}"`, child progress) is not surfaced. | `/runs/{id}` step body | `run_detail.html:379` (literal); `handlers_runs.go:744–751` (no kind detection for iterate/aggregate/loop/branch) |
| **U7** | `ReviewVerdict` red badge sits inside the IN row between artifact chips — operator reads it as "fail on this input" when it actually means "review LLM said the step failed". Same span renders identically for all step kinds. | `/runs/{id}` | `run_detail.html:195` (verdict between IN tabs and OUT label); also no per-`InputArtifactRef` resolution flag in `types.go:99–102`. |
| **U8** | Pipeline detail page recent-runs list (`/pipelines/{name}`) flattens children of sub-pipeline runs into the same table — operator cannot tell standalone runs from spawned children for the same pipeline name. | `/pipelines/{name}` | `handlers_pipelines.go:444–466` (`ListRunsOptions` has no parent filter) |
| **U9** | `child_run_row.html` indent glyph is single-level (`└`) — does not generalise to ≥2 levels of nesting (composition spawning composition). | `/runs` | `partials/child_run_row.html:15` |
| **U10** | Iterate / aggregate steps do not show child status counts (`5/6 done`, `2 failed`) on the parent step row. | `/runs/{id}` | No aggregation in `buildStepDetails`; `composition.go:213, 284` only emits per-iteration events. |

---

## 3. Data / API Gaps (numbered)

| # | Gap | Layer | Citation |
|---|---|---|---|
| **D1** | `CompositionExecutor` never sets `parent_run_id` on iterate / aggregate / sub-pipeline children. Root cause of U1, U2, U3, U5, U8. | `internal/pipeline/composition.go` | L513–578 (no `SetParentRun`); `sequence.go:347` (creates run, returns ID via `PipelineResult` that is discarded). |
| **D2** | `pipeline_run` lacks `iterate_index`, `iterate_total`, `iterate_mode` columns. | schema | `state/types.go:8–26`; `migration_definitions.go` (no migration adds these). |
| **D3** | `pipeline_run` lacks `run_kind` discriminator (`top-level` / `sub-pipeline` / `iterate-child` / `loop-iteration` / `aggregate-target`). | schema | as above. |
| **D4** | `pipeline_run` lacks `sub_pipeline_step_ref` — the parent step's `step.SubPipeline` value at the time the child was spawned (e.g. `audit-security` resolved from `{{ item }}`). | schema | as above. |
| **D5** | No store method `GetRunTree(rootID) []RunRecord` for recursive subtree fetch. | store | `state/store.go` (no recursive query). |
| **D6** | No store method `GetSubtreeTokens(rootID) int64`. | store | `run_stats.go:31–36` only aggregates per-issue/PR. |
| **D7** | No `/api/runs/{id}/children` REST endpoint. Tree must be assembled in handlers. | API | `routes.go:41–68`. |
| **D8** | `Aggregate.Into` writes to disk without registering an `pipeline_artifact` row (`composition.go:473–478`). The downstream `inject_artifacts` ref resolves via filesystem, but the artifact is invisible to artifact-table consumers. | runtime | `composition.go:477`. |
| **D9** | `InputArtifactRef` (`types.go:99–102`) carries no `Resolved`/`SourcePath` field. UI cannot show per-ref injection truth even after the resolver runs. | view model | no field for runtime resolution. |
| **D10** | `SequenceExecutor.PipelineResult` is discarded by `runSubPipeline` (`composition.go:576`). Even if D1 were fixed, the composition executor has no handle to call `SetParentRun(childID, …)` because the child ID is not surfaced. | API contract between sequence and composition | `composition.go:523, :576`. |

---

## 4. Step-kind renderer matrix

Source-of-truth: `internal/webui/handlers_runs.go:744–751`, `internal/webui/templates/run_detail.html:160–199`, `internal/webui/templates/partials/step_card.html:12–69`.

| Step kind | YAML signal | `StepType` produced by handler | run_detail.html branch | step_card.html branch | Body content rendered | Missing affordance |
|---|---|---|---|---|---|---|
| `prompt` (default) | `exec.prompt: "…"` | `""` | none (default IO row) | none | prompt + IN refs + OUT refs + logs | OK |
| `command` | `exec.script: "…"` or `step.Type == "command"` | `"command"` | `tp-command` icon `>_` | `step-type-command-detail` (script preformatted) | script | OK |
| `gate` | `step.Gate != nil` | `"gate"` | `tp-gate` icon `◆` | `step-type-gate-detail` (prompt + choices + interaction panel) | OK | OK |
| `conditional` | `step.Type == "conditional"` + `step.Edges` | `"conditional"` | `tp-conditional` icon `◆` | `step-type-conditional-detail` (edges joined) | edges | OK |
| `sub_pipeline` (plain) | `step.SubPipeline != ""`, no `Iterate` | `"pipeline"` | `tp-pipeline` icon `⟐` + child cards | `step-type-pipeline-detail` (link to `/pipelines/{name}`) | linked child run cards (PR #710) | child cards work for plain sub-pipeline only |
| `iterate` | `step.Iterate != nil`, `step.SubPipeline = "{{ item }}"` | `"pipeline"` (folds in) | `tp-pipeline` shape, but `SubPipeline` is the unresolved template | none | child cards may render but template path is wrong | iterate config (over, mode, max_concurrent), per-item progress, list of resolved children with status |
| `iterate` (no SubPipeline) | `step.Iterate != nil`, SubPipeline empty | `""` | default | none | nothing — `(no prompt)` on click | full composition renderer |
| `aggregate` | `step.Aggregate != nil` | `""` | default | none | nothing | aggregate config (from, into, strategy, key), output artifact size, link to merged file |
| `loop` | `step.Loop != nil` | `""` | default | none | nothing | loop config (max_iterations), per-iteration sub-step state, exit condition |
| `branch` | `step.Branch != nil` | `""` | default | none | nothing | branch config (when), chosen sub-step, dead branches greyed out |

`(no prompt)` literal: `internal/webui/templates/run_detail.html:379`.

---

## 5. IN/OUT indicator truth table

Rows = step kinds that produce implicit / runtime-only artifacts. Columns = whether that artifact is declared in `output_artifacts:`, whether it resolves at runtime, and what badge the UI currently emits.

| Step kind | Declared in `output_artifacts:` | Registered as `pipeline_artifact` row | Resolves at runtime | Current UI badge / chip | Truth gap |
|---|---|---|---|---|---|
| `prompt` with explicit `output_artifacts:` | yes | yes (writeOutputArtifacts pass) | yes | OUT chip with size | none |
| `prompt` with `inject_artifacts` referencing upstream chip | (consumes) | n/a | yes (filesystem) | IN chip clickable; **no resolution flag** | per-ref `Resolved bool` missing — D9 |
| `aggregate.into: ".agents/output/x.json"` | **no** | **no** (never indexed — D8) | yes (file written by `composition.go:477`) | none on producer; downstream IN chip works because resolver hits filesystem | producer step shows no OUT chip → UX gap U6 |
| `iterate.collected-output` (collected child outputs registered to step.ID by `collectIterateOutputs` at `composition.go:298`) | **no** | **no** | yes (template context only) | none | no OUT chip; downstream `inject_artifacts` cannot reference it by `(step, name)` |
| `loop.last-iteration-output` | **no** | **no** | yes (template context) | none | same |
| `agent_review` step verdict | n/a | recorded via `ReviewVerdict` field (`types.go:89`) | yes | red/green pill at `run_detail.html:195` between IN refs and OUT label | placement implies it adjudicates IN refs → UX gap U7 |

Key takeaway: the false-positive `fail` badge described in the issue comment is **not** generated by an artifact resolver. It is the `ReviewVerdict` pill displayed in a misleading position. The schema-side gap (aggregate / iterate / loop emit no artifact rows) is real and orthogonal — it does not currently produce a `fail` badge, but it blocks any future "artifact resolved? yes/no" indicator.

---

## 6. Recommendations (gap → fix lane)

Lanes:
- **R** = redesign work in this audit's follow-up issue.
- **D** = data-layer / executor change required first; UI change rides on it.
- **709** = ride on (or extend) the merged Fat-Gantt-Shapes / detail-page redesign.

| Gap | Lane | Notes |
|---|---|---|
| U1 | **D**, then **R** | Fix D1 + D10 first (composition.go must call `SetParentRun`). UI nesting at `handlers_runs.go:201` already works — feed it the right data. |
| U2 | **709** | Extend run_detail.html child-card branch beyond `StepType == "pipeline"` to also render iterate/aggregate child cards once D1 is in. |
| U3 | **D**, then **R** | Needs D2 (iterate index columns) + D4 (sub_pipeline_step_ref). Render breadcrumb with "child 7/19 of resolve-each in `ops-pr-respond`". |
| U4 | **D**, then **R** | D3 `run_kind` enum; UI groups Running list by kind (top-level, audit, fix, sub-pipeline). |
| U5 | **D**, then **R** | D6 subtree token rollup. Surface as `tokens (subtree)` next to `tokens (own)` in `run_row.html` + `run_detail.html` summary. |
| U6 | **R** | Pure UI: composition-step renderer in `run_detail.html` per step kind. Use existing `step.Iterate`, `step.Aggregate`, `step.Loop`, `step.Branch` data already loaded into `Step`. Add child-progress aggregation (depends on D1 → child runs reachable; depends on D2 → child position). |
| U7 | **R** | Move `ReviewVerdict` pill out of the IN cluster. Place it on the Row-1 step header next to status, OR introduce a verdict row separate from IO. |
| U8 | **D**, then **R** | Add `ParentRunID *string` filter to `ListRunsOptions` (`state/types.go:39`); update `/pipelines/{name}` to default `ParentRunID = NULL` (only top-level). Optional toggle to show children. |
| U9 | **R** | Replace single-glyph indent with depth-based indent or a tree connector svg in `child_run_row.html`. |
| U10 | **R** | Fan-in in step-row renderer: count children where `parent_step_id == this step` and bucket by status. Depends on D1. |
| D1 | **D** | Refactor `runSubPipeline` to return `childRunID` (or accept a callback). Then call `SetParentRun(childRunID, parentRunID, step.ID)` immediately after `seqExecutor.Execute` (matching the executor.go:5505–5507 pattern). |
| D2 | **D** | Migration #N+1 adds `iterate_index INT NULL`, `iterate_total INT NULL`, `iterate_mode TEXT NULL`. Pass through `RunOption` from composition into `CreateRun`. |
| D3 | **D** | Same migration: `run_kind TEXT` defaulted via insert-time logic. Avoid re-deriving at read time. |
| D4 | **D** | Same migration: `sub_pipeline_step_ref TEXT NULL` — the resolved pipeline name (post-template) of the step that spawned the child. |
| D5 | **D** | Add `GetRunTree(rootID) ([]RunRecord, error)` using SQLite `WITH RECURSIVE`. |
| D6 | **D** | Add `GetSubtreeTokens(rootID) (int64, error)` — same recursive CTE on `total_tokens`. |
| D7 | **709** or **D** | Endpoint is small; can ship with D5 to power AJAX expansion of nested rows (avoids full page reload). |
| D8 | **D** | After `os.WriteFile` at `composition.go:477`, call `store.SaveArtifact(...)` with the `Aggregate.Into` path. |
| D9 | **D** + **R** | Extend `InputArtifactRef` with `ResolvedFromPath string` and `Resolved bool` — populated by the inject resolver at run time. Render a per-chip status dot on IN tabs. |
| D10 | **D** | Surface `RunID` from `SequenceExecutor.executeSinglePipeline` to `runSubPipeline`'s caller. |

**Summary of effort split:** ~5 schema/executor changes (D1–D4, D8) unlock 6 of 10 UX gaps. The remaining 4 UX gaps (U2, U6, U7, U9) are pure UI.

---

## 7. Follow-up issue draft

> **Title:** feat(webui): tree-aware run rendering — composition pipelines (parent + iterate/aggregate children)
>
> **Body:**
>
> Implementation issue derived from audit #1412 (`docs/webui-composition-audit.md`). Implements the data-layer (D1–D10) and UI redesign (R) gaps the audit catalogues.
>
> **Phase 1 — Data layer (blocking)**
>
> 1. `CompositionExecutor.runSubPipeline` returns child run ID (D10) and calls `store.SetParentRun(childID, parentID, stepID)` after `seqExecutor.Execute` — same pattern as `executor.go:5505–5507` (D1).
> 2. Migration adds columns `iterate_index INT`, `iterate_total INT`, `iterate_mode TEXT`, `run_kind TEXT`, `sub_pipeline_step_ref TEXT` to `pipeline_run` (D2 + D3 + D4).
> 3. Composition pipelines pass these via a new `RunOption` into `s.store.CreateRun(...)`.
> 4. New store methods `GetRunTree(rootID)` and `GetSubtreeTokens(rootID)` using `WITH RECURSIVE` (D5 + D6).
> 5. `aggregate.Into` writes also call `store.SaveArtifact(...)` registering an artifact row (D8).
> 6. `InputArtifactRef` extended with `Resolved bool` + `ResolvedFromPath string`, populated by the inject resolver (D9).
> 7. New API endpoint `GET /api/runs/{id}/children` (D7).
>
> **Phase 2 — UI**
>
> 8. `runs.html` running section: group by `run_kind`; depth-aware indent in `child_run_row.html` (U4, U9).
> 9. `pipeline_detail.html` recent-runs list defaults to top-level only via new `ParentRunID` filter on `ListRunsOptions`; toggle to include children (U8).
> 10. `run_detail.html` step-kind renderer: composition-step body for `iterate`, `aggregate`, `loop`, `branch` showing config + per-child progress + clickable child run links (U2, U6, U10).
> 11. Move `ReviewVerdict` pill out of IO row to step Row-1 (U7).
> 12. Child run page header: "child *N* of *M* of `step` in `parent`" breadcrumb (U3).
> 13. Subtree token total surfaced on parent rows and parent detail header (U5).
>
> **Acceptance**
>
> - Re-run `ops-pr-respond` on any PR. The Running list shows the parent with `5/6 audits done` aggregate; the 6 audit-* and 19 impl-finding children are nested and grouped by run-kind.
> - Each child run page shows its position in the iteration and a parent breadcrumb.
> - Aggregate step shows `into: x.json` with a clickable size chip; downstream `inject_artifacts` IN chip resolves green.
> - `ReviewVerdict` pill never appears in the IN cluster.
>
> **Out of scope:** changes to the executor / composition model itself; new step kinds.
>
> **Audit reference:** `docs/webui-composition-audit.md` @ `8a2b9361`.
> **Related:** #709 (merged), #710 (merged), #772 (running-pipelines section).
