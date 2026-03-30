# Implementation Plan: WebUI Start Pipeline Dialog Flags

## Objective

Expose the five high-priority CLI flags (`--model`, `--from-step`, `--steps`, `--exclude`, `--dry-run`) in the web UI's "Start Pipeline" dialog so users can configure pipeline runs without the CLI. Step lists must be dynamically populated from the pipeline definition.

## Approach

The implementation touches three layers: **types** (request struct), **handlers** (wiring to executor options), and **UI** (templates + JS). There are three distinct start dialogs that need updating:

1. **Runs page** (`runs.html`) — full dialog with pipeline selector + new controls
2. **Pipelines list page** (`pipelines.html`) — quickstart dialog (pipeline is pre-selected)
3. **Pipeline detail page** (`pipeline_detail.html`) — quickstart dialog (pipeline is pre-selected)

The existing `GET /api/pipelines/{name}` endpoint already returns step IDs, personas, and dependencies — this will be used to populate step dropdowns dynamically when a pipeline is selected.

### Key Design Decisions

1. **Collapsible "Advanced Options" section**: The five new controls go into a collapsible `<details>` element below the Input field. This keeps the dialog simple for basic use while exposing power-user controls on demand.

2. **Step selection via checkboxes, not multi-select**: Multi-select `<select>` elements are notoriously hard to use. Use checkbox lists that appear after pipeline selection, populated from the API.

3. **Dry-run returns a validation report**: When `dry_run: true`, the handler runs `DryRunValidator.Validate()` and returns the report as JSON instead of launching execution. The UI shows the report inline.

4. **from-step is separate from steps/exclude**: `from-step` means "resume from this step onward" (sequential). `steps` means "run only these specific steps". They serve different purposes but are mutually exclusive in the UI.

5. **Both `StartPipelineRequest` and `SubmitRunRequest` need updating**: The runs page uses `handleSubmitRun` (POST /api/runs), while pipelines pages use `handleStartPipeline` (POST /api/pipelines/{name}/start). Both need the new fields.

## File Mapping

### Modify

| File | Changes |
|------|---------|
| `internal/webui/types.go` | Add `Model`, `FromStep`, `Steps`, `Exclude`, `DryRun` fields to `StartPipelineRequest` and `SubmitRunRequest`. Add `DryRunResponse` type. |
| `internal/webui/handlers_control.go` | Wire new request fields into `ExecutorOption` list in `launchPipelineExecution`. Add dry-run branch in `handleStartPipeline` and `handleSubmitRun`. |
| `internal/webui/templates/runs.html` | Add collapsible "Advanced Options" section with model dropdown, from-step dropdown, steps/exclude checkboxes, dry-run toggle. Add JS to populate step lists on pipeline select. |
| `internal/webui/templates/pipelines.html` | Expand quickstart dialog with same advanced options. Pipeline is pre-selected so step lists load immediately. |
| `internal/webui/templates/pipeline_detail.html` | Expand quickstart dialog with same advanced options. Pipeline is pre-selected so step lists load immediately. |
| `internal/webui/static/app.js` | Update `startPipeline()` to collect and send new fields. Add `loadPipelineSteps()` helper. Update `submitQuickStart()` pattern in templates. |

### No New Files Needed

All changes fit within existing files. No new packages, no new routes (the existing `GET /api/pipelines/{name}` provides step data).

## Architecture Decisions

1. **Reuse existing API**: `GET /api/pipelines/{name}` already returns `PipelineDetail` with step IDs. No new endpoint needed for step population.

2. **Executor options via `launchPipelineExecution`**: The shared launch function already builds `[]pipeline.ExecutorOption`. We add conditional appends for model, timeout, step filter, and auto-approve — mirroring what `cmd/wave/commands/run.go` does.

3. **Dry-run is validation-only**: The CLI's `--dry-run` runs `DryRunValidator.Validate()` and prints a report. The web UI does the same but returns JSON. No pipeline execution occurs.

4. **from-step creates a new run**: Consistent with how `handleResumeRun` works — creates a new run record and calls `launchPipelineExecution` with the fromStep parameter. The difference is this is for a fresh pipeline start (not resuming a failed run), so we skip the "must be failed/cancelled" check.

5. **Model dropdown values**: `""` (default/auto), `"haiku"`, `"sonnet"`, `"opus"`. These match what the CLI accepts and what `resolveModel()` processes.

## Risks

| Risk | Mitigation |
|------|-----------|
| Step IDs contain special characters | Step IDs are already validated by pipeline YAML loading; safe to use in HTML attributes |
| from-step + steps are mutually exclusive | UI disables steps/exclude when from-step is set, and vice versa. Handler validates mutual exclusivity. |
| Dry-run for pipelines with template vars | `DryRunValidator` already handles template vars gracefully (warns but doesn't error). Return warnings in response. |
| Quickstart dialogs diverge from main dialog | Extract shared JS function `collectAdvancedOptions()` used by all three dialogs |

## Testing Strategy

1. **Unit tests for handler changes**: Test `handleStartPipeline` with each new field (model, from-step, steps, exclude, dry-run). Verify executor options are correctly built.
2. **Unit tests for dry-run response**: Test that dry-run returns validation report and does NOT create a run record.
3. **Unit tests for mutual exclusivity**: Test that from-step + steps returns a 400 error.
4. **Integration verification**: Manual testing of all three dialogs (runs page, pipelines list, pipeline detail) to confirm step lists populate and options are sent correctly.
