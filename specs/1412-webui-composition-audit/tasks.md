# Work Items — #1412 WebUI Composition Audit

## Phase 1: Setup

- [X] Item 1.1: Create feature branch `1412-webui-composition-audit` (done by planner; resume worktree uses `-r2` suffix)
- [X] Item 1.2: Create `specs/1412-webui-composition-audit/` directory with `spec.md`, `plan.md`, `tasks.md` (done in plan step)
- [X] Item 1.3: Create `docs/webui-composition-audit.md` (skeleton step skipped — final doc written directly)

## Phase 2: Inventory (parallelizable)

- [X] Item 2.1: Audit `internal/webui/handlers_runs.go` — list/detail handlers, parent_run_id queries, child fetch logic [P]
- [X] Item 2.2: Audit `internal/webui/handlers_pipelines.go` — pipeline-page run listings, grouping behavior [P]
- [X] Item 2.3: Audit `internal/webui/types.go` + `run_stats.go` — view models, token aggregation, ParentRunID exposure [P]
- [X] Item 2.4: Audit `internal/webui/templates/runs.html`, `run_detail.html`, `pipeline_detail.html`, partials — current render shape per step kind [P]
- [X] Item 2.5: Audit `internal/state/types.go`, `store.go`, `migration_definitions.go` — `pipeline_run` schema (parent_run_id, iterate index/total, run kind) [P]
- [X] Item 2.6: Audit `internal/pipeline/composition.go`, `sequence.go`, `executor.go` — child run creation, parent linkage, implicit artifact registration [P]
- [X] Item 2.7: Locate API endpoints for runs (e.g. `/api/runs`, `/api/runs/<id>/children`) — verify whether tree data is exposed [P]
- [X] Item 2.8: Read #709 / #710 specs and open PRs to identify in-flight overlap [P]

## Phase 3: Synthesis

- [X] Item 3.1: Compile step-kind renderer matrix (rows = step kinds, columns = template path / current render / missing)
- [X] Item 3.2: Compile IN/OUT indicator truth table — declared vs runtime-resolved, document false-positive `fail` cases
- [X] Item 3.3: Map each issue UX gap (#1–#5 + comment items) to {view-only, backend-required, both}
- [X] Item 3.4: Map data-layer gaps (schema columns, store methods, API endpoints) needed for tree rendering
- [X] Item 3.5: Build recommendations table — per gap: {redesign-here, data-layer-here, ride-on-#709}

## Phase 4: Document Production

- [X] Item 4.1: Write `docs/webui-composition-audit.md` Inventory section
- [X] Item 4.2: Write Numbered UX Gaps section (one per finding, with file:line citations)
- [X] Item 4.3: Write Numbered Data/API Gaps section
- [X] Item 4.4: Write Step-Kind Renderer Matrix and IN/OUT Indicator Truth Table
- [X] Item 4.5: Write Recommendations table
- [X] Item 4.6: Write Follow-up Issue Draft (title, body, acceptance criteria, file pointers)

## Phase 5: Validation

- [X] Item 5.1: Verify every gap cites a concrete file path or template name
- [X] Item 5.2: Verify recommendations cover every numbered gap
- [X] Item 5.3: Verify follow-up issue draft is implementation-ready
- [X] Item 5.4: Markdown sanity pass (tables / fences balanced)
- [X] Item 5.5: Commit doc + spec artifacts on feature branch
- [ ] Item 5.6: Open follow-up redesign issue and link from this issue (deferred — requires admin authorization per memory feedback_no_external_posts)
