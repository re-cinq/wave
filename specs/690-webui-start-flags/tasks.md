# Tasks

## Phase 1: Backend Types and Handler Wiring

- [ ] Task 1.1: Expand `StartPipelineRequest` and `SubmitRunRequest` with new fields (`Model`, `FromStep`, `Steps`, `Exclude`, `DryRun`) in `internal/webui/types.go`. Add `DryRunResponse` type for dry-run results.
- [ ] Task 1.2: Wire new fields into executor options in `handleStartPipeline` — add conditional `WithModelOverride`, `WithStepFilter`, and `WithStepTimeout` appends. Add from-step support via `launchPipelineExecution`. Validate mutual exclusivity of `from-step` vs `steps`.
- [ ] Task 1.3: Wire same fields into `handleSubmitRun` (POST /api/runs) — same logic as 1.2 but for the runs page submit path.
- [ ] Task 1.4: Implement dry-run branch in both handlers — when `DryRun` is true, run `DryRunValidator.Validate()` and return the report as JSON without creating a run record.

## Phase 2: Frontend — Runs Page Dialog

- [ ] Task 2.1: Add collapsible "Advanced Options" `<details>` section to the start dialog in `runs.html` with: model dropdown (auto/haiku/sonnet/opus), from-step dropdown, steps checkboxes, exclude checkboxes, dry-run toggle.
- [ ] Task 2.2: Add JS in `runs.html` to call `GET /api/pipelines/{name}` on pipeline selection and populate step dropdowns/checkboxes from the response. Disable steps/exclude when from-step is set and vice versa.
- [ ] Task 2.3: Update `startPipeline()` in `app.js` to collect and include new fields in the API request body. Handle dry-run response (show validation report instead of redirecting).

## Phase 3: Frontend — Quickstart Dialogs [P]

- [ ] Task 3.1: Expand quickstart dialog in `pipelines.html` with advanced options. Load step list on dialog open (pipeline is already known). [P]
- [ ] Task 3.2: Expand quickstart dialog in `pipeline_detail.html` with advanced options. Load step list on dialog open (pipeline is already known). [P]
- [ ] Task 3.3: Update `submitQuickStart()` in both templates to collect and send new fields. Handle dry-run response. [P]

## Phase 4: Testing

- [ ] Task 4.1: Write unit tests for `handleStartPipeline` with new fields — model override, step filter, from-step, dry-run. [P]
- [ ] Task 4.2: Write unit tests for `handleSubmitRun` with new fields. [P]
- [ ] Task 4.3: Write unit tests for mutual exclusivity validation (from-step + steps = 400 error). [P]
- [ ] Task 4.4: Write unit test for dry-run response (returns report, no run created). [P]

## Phase 5: Polish

- [ ] Task 5.1: Run `go test ./...` and `go vet ./...` to ensure no regressions.
- [ ] Task 5.2: Manual verification of all three dialogs (runs, pipelines list, pipeline detail).
