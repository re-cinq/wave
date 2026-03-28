# Tasks

## Phase 1: Pipeline Adoption — Core Changes

- [X] Task 1.1: Restructure `impl-issue.yaml` with graph loop (implement→test→fix cycle using edges, conditions, max_visits=3)
- [X] Task 1.2: Add `llm_judge` contract to `ops-pr-review.yaml` quality-review step (mirror security-review pattern with quality criteria)
- [X] Task 1.3: Mirror `impl-issue.yaml` and `ops-pr-review.yaml` changes to `internal/defaults/pipelines/`

## Phase 2: Pipeline Adoption — Bulk Cleanup [P]

- [X] Task 2.1: Remove redundant `max_attempts` lines from all `.wave/pipelines/*.yaml` where named `policy` exists [P]
- [X] Task 2.2: Add `model: claude-haiku-4-5` to all navigator/reviewer/summarizer/analyst steps missing it across `.wave/pipelines/*.yaml` (52 steps in 36 pipelines) [P]
- [X] Task 2.3: Sync all `.wave/pipelines/*.yaml` changes to `internal/defaults/pipelines/*.yaml`

## Phase 3: Doctor Health Checks

- [X] Task 3.1: Add `checkHooksConfig()` health check to `internal/doctor/doctor.go` — scan pipelines for hooks declarations, verify accessibility
- [X] Task 3.2: Add `checkRetroStore()` health check to `internal/doctor/doctor.go` — verify retro store is writable
- [X] Task 3.3: Wire new checks into `RunChecks()` and add table-driven tests in `internal/doctor/doctor_test.go`

## Phase 4: TUI Enhancements

- [X] Task 4.1: Add step type indicators to TUI run detail in `internal/tui/pipeline_detail.go` — show [gate], [cmd], [cond], [pipeline] badges next to step names
- [X] Task 4.2: Create TUI retro viewer component in `internal/tui/retro_view.go` — display quantitative metrics and narrative
- [X] Task 4.3: Integrate retro viewer into pipeline detail view for finished runs in `internal/tui/pipeline_detail.go`

## Phase 5: Documentation [P]

- [X] Task 5.1: Write `docs/guides/graph-loops.md` — graph loops and conditional routing guide [P]
- [X] Task 5.2: Write `docs/guides/approval-gates.md` — human approval gates guide [P]
- [X] Task 5.3: Write `docs/guides/retry-policies.md` — retry policies guide [P]
- [X] Task 5.4: Write `docs/guides/model-routing.md` — multi-adapter model routing guide [P]

## Phase 6: Validation

- [X] Task 6.1: Run `go test ./...` — full regression test
- [X] Task 6.2: Verify `.wave/pipelines/` and `internal/defaults/pipelines/` are in sync
