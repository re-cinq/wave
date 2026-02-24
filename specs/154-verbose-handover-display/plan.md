# Implementation Plan: Verbose Handover Display (#154)

## Objective

Show artifact paths, contract validation results, and handover target step names beneath completed pipeline steps when running in verbose mode (`wave run -v`). Default output remains unchanged.

## Approach

The implementation follows a data-flow strategy: enrich the `PipelineContext` with handover metadata collected from events, then render that metadata in both the BubbleTea TUI and the non-TTY `BasicProgressDisplay`.

### Data Flow

1. **Event emission** (already exists): The pipeline executor already emits `contract_passed`, `contract_failed`, `contract_soft_failure`, and `completed` events with artifact paths. No changes needed in `internal/pipeline/`.
2. **Data capture**: Both `BubbleTeaProgressDisplay` and `BasicProgressDisplay` must capture handover-related event data (artifact paths, contract status, handover targets) per step.
3. **Context propagation**: Add new fields to `PipelineContext` to carry per-step handover metadata to the rendering layer.
4. **Rendering**: Extend `renderCurrentStep()` in `bubbletea_model.go` (TTY path) and `EmitProgress()` in `progress.go` (non-TTY path) to display the metadata in tree format below completed steps, gated by verbose mode.

## File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/display/types.go` | modify | Add `HandoverInfo` struct and `HandoversByStep` field to `PipelineContext` |
| `internal/display/bubbletea_model.go` | modify | Render handover metadata in tree format below completed steps (verbose only) |
| `internal/display/bubbletea_progress.go` | modify | Capture handover events (`contract_passed`, `contract_failed`, artifacts) in `updateFromEvent()` and propagate to `PipelineContext` in `toPipelineContext()` |
| `internal/display/progress.go` | modify | Capture handover data in `BasicProgressDisplay.EmitProgress()` and render below completed step lines (non-TTY verbose path) |
| `internal/display/bubbletea_model_test.go` | modify | Add tests for verbose handover rendering in TUI |
| `internal/display/progress_test.go` | modify | Add tests for verbose handover rendering in non-TTY path |

## Architecture Decisions

### AD-1: Handover metadata stored in PipelineContext (not in events)

The rendering layer already receives a `PipelineContext` snapshot. Adding handover metadata to `PipelineContext` keeps rendering pure — the view function reads from a single data source rather than accumulating state from events.

### AD-2: Verbose gating at the rendering layer

The handover metadata is always captured into `PipelineContext`, but only rendered when verbose mode is enabled. This means the data is available for future features (e.g., web dashboard) without conditional logic in the data pipeline.

### AD-3: Reuse existing tree format

The `├─` / `└─` tree format is already used for deliverables (lines 294-303 of `bubbletea_model.go`). The handover metadata will be rendered using the same pattern, placed after deliverables and before the next step.

### AD-4: Handover target determined by step order

The handover target step name is derived from `StepOrder` — the next step in the pipeline definition order after the current step. For matrix/parallel steps, only the `TargetStep` field from `HandoverConfig` is used if explicitly set. If no explicit target and the step is the last one, the handover line is omitted.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Verbose output becomes too noisy with many artifacts | Low | Medium | Show at most 3 artifact lines, then "… and N more" |
| Contract events may not fire for steps without contracts | N/A | None | Simply omit the contract line when no contract data exists |
| PipelineContext grows with new fields | Low | Low | Fields are maps keyed by stepID — same pattern as existing fields |

## Testing Strategy

### Unit Tests

- **BubbleTea model**: Test `renderCurrentStep()` output with handover data in PipelineContext (verbose=true and verbose=false)
- **BasicProgressDisplay**: Test `EmitProgress()` output for completed steps with handover events (verbose=true and verbose=false)
- **PipelineContext**: Test that `HandoversByStep` is correctly populated from event sequences

### Integration Tests

- Verify that a full pipeline execution with verbose mode shows handover metadata in the output
- Verify that non-verbose mode output is unchanged

### Regression Tests

- Existing `bubbletea_model_test.go` and `progress_test.go` tests must continue to pass
- Deliverable tree-format rendering must not be disrupted
