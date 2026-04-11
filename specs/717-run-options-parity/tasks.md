# Tasks: Run Options Parity Across All Surfaces

**Feature**: #717 | **Branch**: `717-run-options-parity` | **Generated**: 2026-04-11

## Phase 1: Setup

- [X] T001 [P] Verify branch and baseline — run `go build ./...` and `go test ./...` to confirm green baseline before changes

## Phase 2: Foundational — API Types & Server-Side RunOptions (Blocking)

These tasks extend the backend types that all surfaces depend on. Must complete before any UI work.

- [X] T002 Extend `webui.RunOptions` struct with Tier 2–4 fields — `internal/webui/handlers_control.go:30-37`
- [X] T003 [P] Extend `StartPipelineRequest` with Tier 1–4 fields — `internal/webui/types.go:205-213`
- [X] T004 [P] Extend `SubmitRunRequest` with matching fields — `internal/webui/types.go:540-549`
- [X] T005 [P] Create named `StartIssueRequest` type with Tier 1–3 fields (replace anonymous struct) — `internal/webui/types.go` (new type)
- [X] T006 [P] Create `StartPRRequest` type with Tier 1–3 fields — `internal/webui/types.go` (new type)
- [X] T007 Wire new `webui.RunOptions` fields to subprocess flags in `spawnDetachedRun()` — `internal/webui/handlers_control.go` (extend args slice)
- [X] T008 Update `handleStartPipeline` to populate all new `RunOptions` fields from `StartPipelineRequest` — `internal/webui/handlers_control.go`
- [X] T009 Update `handleSubmitRun` to populate all new `RunOptions` fields from `SubmitRunRequest` — `internal/webui/handlers_control.go`
- [X] T010 Add mutual exclusion validation (continuous + from-step) and on-failure enum validation in handler layer — `internal/webui/handlers_control.go`
- [X] T011 Run `go build ./...` and `go test ./...` to verify Phase 2 compiles and passes

## Phase 3: User Story 1 — WebUI Inline Run Form on Pipeline Detail (P1)

- [X] T012 Remove modal dialog trigger and replace with inline tiered form skeleton — `internal/webui/templates/pipeline_detail.html`
- [X] T013 Implement Tier 1 section (Input, Adapter, Model) always visible in inline form — `internal/webui/templates/pipeline_detail.html`
- [X] T014 Implement collapsible "Advanced" section with Tier 2 options (from-step, force, dry-run, detach, steps, exclude, timeout, on-failure) — `internal/webui/templates/pipeline_detail.html`
- [X] T015 Implement conditional Force field visibility (only shown when from-step has value) via JS — `internal/webui/templates/pipeline_detail.html`
- [X] T016 Implement collapsible "Continuous" section with Tier 3 options (continuous toggle, source, max-iterations, delay) — `internal/webui/templates/pipeline_detail.html`
- [X] T017 Add client-side mutual exclusion validation: continuous + from-step, steps ∩ exclude overlap — `internal/webui/templates/pipeline_detail.html`
- [X] T018 Wire form submission to `POST /api/pipelines/{name}/start` with full field set — `internal/webui/templates/pipeline_detail.html`
- [X] T019 Implement detach navigation behavior (navigate to run detail page immediately) — `internal/webui/templates/pipeline_detail.html`
- [X] T020 Implement dry-run inline report rendering (display report on same page, no navigation) — `internal/webui/templates/pipeline_detail.html`
- [X] T021 Add inline form styles (collapsible sections, tier visual grouping) — `internal/webui/static/style.css`
- [X] T022 Add from-step picker populated from pipeline manifest steps — `internal/webui/templates/pipeline_detail.html` + handler data
- [X] T023 Add adapter selector populated from available adapters — `internal/webui/templates/pipeline_detail.html` + handler data
- [X] T024 Run `go build ./...` and `go test ./...` to verify Phase 3

## Phase 4: User Story 2 — API Request Types Full Wiring (P1)

- [X] T025 Verify `handleStartPipeline` passes all Tier 1–4 fields through to subprocess (integration-level test) — `internal/webui/handlers_control.go`
- [X] T026 [P] Add table-driven unit tests for `spawnDetachedRun` with Tier 2–4 fields — `internal/webui/handlers_control_test.go` (new or extend)
- [X] T027 [P] Add table-driven unit tests for mutual exclusion and on-failure validation — `internal/webui/handlers_control_test.go`
- [X] T028 Run `go test ./internal/webui/...` to verify Phase 4

## Phase 5: User Story 3 — TUI Pipeline Launcher Expansion (P2)

- [X] T029 Add typed fields to `LaunchConfig` struct (Adapter, Timeout, FromStep, Steps, Exclude, Detach, OnFailure) — `internal/tui/pipeline_messages.go:46-54`
- [X] T030 Add --detach to `DefaultFlags()` — `internal/tui/run_selector.go:24-34`
- [X] T031 Update `run_selector_test.go` expectations for new DefaultFlags — `internal/tui/run_selector_test.go`
- [X] T032 Add adapter selector field to TUI argument form — `internal/tui/pipeline_detail.go`
- [X] T033 Add timeout input field to TUI argument form — `internal/tui/pipeline_detail.go`
- [X] T034 Add from-step picker (populated from manifest steps) to TUI argument form — `internal/tui/pipeline_detail.go`
- [X] T035 Add steps and exclude text input fields to TUI argument form — `internal/tui/pipeline_detail.go`
- [X] T036 Add detach toggle to TUI argument form — `internal/tui/pipeline_detail.go`
- [X] T037 Add on-failure selector (halt/skip) to TUI argument form — `internal/tui/pipeline_detail.go`
- [X] T038 Map new typed `LaunchConfig` fields to subprocess flags in launcher — `internal/tui/pipeline_launcher.go`
- [X] T039 Add TUI-side mutual exclusion validation (continuous + from-step) — `internal/tui/pipeline_detail.go`
- [X] T040 Disable from-step picker when pipeline has no steps — `internal/tui/pipeline_detail.go`
- [X] T041 Run `go test ./internal/tui/...` to verify Phase 5

## Phase 6: User Story 4 — Issues/PRs Pages Expose Overrides (P2)

- [X] T042 Update `handleAPIStartFromIssue` to use named `StartIssueRequest` type and populate `RunOptions` — `internal/webui/handlers_issues.go`
- [X] T043 Add Adapter and Model selectors to issue detail template — `internal/webui/templates/issue_detail.html`
- [X] T044 Add collapsible "Advanced" section with Tier 2 options to issue detail template — `internal/webui/templates/issue_detail.html`
- [X] T045 Wire issue form submission to pass overrides in request body — `internal/webui/templates/issue_detail.html`
- [X] T046 Create `handleAPIStartFromPR` handler for `POST /api/prs/start` — `internal/webui/handlers_prs.go`
- [X] T047 Register `POST /api/prs/start` route — `internal/webui/routes.go`
- [X] T048 Add "Run Pipeline" button with Adapter/Model selectors and Advanced section to PR detail template — `internal/webui/templates/pr_detail.html`
- [X] T049 Wire PR form submission to `POST /api/prs/start` — `internal/webui/templates/pr_detail.html`
- [X] T050 Run `go build ./...` and `go test ./...` to verify Phase 6

## Phase 7: User Story 5 — CLI Help Groups Flags by Tier (P3)

- [X] T051 Group CLI flags into four sections (Essential, Execution, Continuous, Dev/Debug) using Cobra flag groups or custom UsageFunc — `cmd/wave/commands/run.go:163-186`
- [X] T052 Align flag descriptions with canonical tier model language — `cmd/wave/commands/run.go`
- [X] T053 Run `wave run --help` and verify four-section output — manual verification

## Phase 8: User Story 6 — Documentation (P3)

- [X] T054 [P] Update `docs/reference/cli.md` with tier-grouped flag documentation — `docs/reference/cli.md`
- [X] T055 [P] Create or update running-pipelines guide with TUI and WebUI run options — `docs/running-pipelines.md`
- [X] T056 [P] Add CHANGELOG entry for run options parity — `CHANGELOG.md`

## Phase 9: Polish & Cross-Cutting

- [X] T057 Verify `steps` ∩ `exclude` overlap validation works on all surfaces — cross-cutting
- [X] T058 Verify from-step with non-existent step shows inline validation error (WebUI + TUI) — cross-cutting
- [X] T059 Verify detach + dry-run precedence (dry-run wins, detach ignored) on all surfaces — cross-cutting
- [X] T060 Verify continuous + max_iterations=0 shows "runs indefinitely" warning in WebUI/TUI — cross-cutting
- [X] T061 Verify timeout=0 treated as infinite on all surfaces — cross-cutting
- [X] T062 Run full `go build ./...` and `go test ./...` — final gate
