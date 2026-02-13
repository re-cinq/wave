# Implementation Plan: Pipeline Step Visibility in Default Run Mode

**Branch**: `100-pipeline-step-visibility` | **Date**: 2026-02-13 | **Spec**: `specs/100-pipeline-step-visibility/spec.md`
**Input**: Feature specification from `specs/100-pipeline-step-visibility/spec.md`

## Summary

Display all pipeline steps (pending, running, completed, failed, skipped, cancelled) in the default run mode TUI, replacing the current behavior that only shows completed and running steps. Each step shows its name, persona, state indicator, and timing information. The implementation modifies the rendering layer (`internal/display/`) without changing pipeline execution logic.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss` (existing)
**Storage**: N/A (display-only changes)
**Testing**: `go test ./internal/display/...` (unit), `go test -race ./...` (full suite)
**Target Platform**: Linux/macOS terminals with TTY + ANSI support
**Project Type**: Single Go binary
**Performance Goals**: <5% render overhead (existing target), 30 FPS refresh (existing 33ms tick)
**Constraints**: No new dependencies, no pipeline execution changes, verbose mode unaffected
**Scale/Scope**: Typical pipelines have 2-10 steps; edge case allows overflow for >terminal height

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary, Minimal Dependencies | ✅ Pass | No new dependencies added |
| P2: Manifest as Single Source of Truth | ✅ Pass | No manifest changes |
| P3: Persona-Scoped Execution Boundaries | ✅ Pass | No execution boundary changes |
| P4: Fresh Memory at Every Step Boundary | ✅ Pass | Display-only, no memory model changes |
| P5: Navigator-First Architecture | ✅ Pass | No pipeline ordering changes |
| P6: Contracts at Every Handover | ✅ Pass | No contract changes |
| P7: Relay via Dedicated Summarizer | ✅ Pass | No relay changes |
| P8: Ephemeral Workspaces for Safety | ✅ Pass | No workspace changes |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling |
| P10: Observable Progress, Auditable Operations | ✅ Enhanced | More comprehensive progress visibility |
| P11: Bounded Recursion and Resource Limits | ✅ Pass | No recursion changes |
| P12: Minimal Step State Machine | ✅ Pass | Uses existing 6 states, no new states |
| P13: Test Ownership for Core Primitives | ✅ Required | Full test suite must pass |

**Post-Phase 1 Re-check**: All principles remain compliant. The design adds a single field (`StepPersonas`) to `PipelineContext` and rewrites rendering methods. No constitutional violations detected.

## Project Structure

### Documentation (this feature)

```
specs/100-pipeline-step-visibility/
├── plan.md              # This file
├── research.md          # Phase 0 research findings
├── data-model.md        # Phase 1 entity definitions
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
internal/display/
├── types.go                  # PipelineContext + StepPersonas field (MODIFY)
├── bubbletea_model.go        # renderCurrentStep() → renderStepList() (MODIFY)
├── bubbletea_progress.go     # toPipelineContext() StepPersonas population (MODIFY)
├── progress.go               # toPipelineContext() + CreatePipelineContext() (MODIFY)
├── dashboard.go              # renderStepStatusPanel() all-step rendering (MODIFY)
├── bubbletea_model_test.go   # New test cases for all-step rendering (ADD TESTS)
├── types_test.go             # StepPersonas field test (ADD TESTS)
├── progress_test.go          # toPipelineContext persona propagation test (ADD TESTS)
└── dashboard_test.go         # All-step dashboard rendering test (ADD TESTS)
```

**Structure Decision**: All changes are contained within the existing `internal/display/` package. No new packages or files required. This feature is a pure rendering enhancement.

## Implementation Design

### Change 1: Add `StepPersonas` to `PipelineContext` (types.go)

Add a new field to `PipelineContext`:

```go
// Step persona mapping
StepPersonas map[string]string // stepID → persona name
```

This field sits alongside the existing `StepStatuses`, `StepOrder`, and `StepDurations` maps and follows the same pattern.

### Change 2: Populate `StepPersonas` in BubbleTea path (bubbletea_progress.go)

In `toPipelineContext()` (line ~268), build the `StepPersonas` map from `btpd.steps`:

```go
stepPersonas := make(map[string]string)
for stepID, step := range btpd.steps {
    stepPersonas[stepID] = step.Persona
}
// ... add to returned PipelineContext
```

### Change 3: Populate `StepPersonas` in ProgressDisplay path (progress.go)

In `toPipelineContext()` (line ~436), build the `StepPersonas` map from `pd.steps`:

```go
stepPersonas := make(map[string]string)
for stepID, step := range pd.steps {
    stepPersonas[stepID] = step.Persona
}
```

Also update `CreatePipelineContext()` to accept and store persona data.

### Change 4: Rewrite `renderCurrentStep()` in BubbleTea model (bubbletea_model.go)

Replace the current method that only shows completed + running steps with a method that iterates ALL steps in `StepOrder` and renders each based on its state:

**Algorithm**:
```
for each stepID in ctx.StepOrder:
    state = ctx.StepStatuses[stepID]
    persona = ctx.StepPersonas[stepID]

    switch state:
    case not_started:
        render "○ stepID (persona)" in muted color
    case running:
        render "spinner stepID (persona) (live_elapsed)" in yellow
        if tool activity: render tool line
    case completed:
        render "✓ stepID (persona) (final_duration)" in cyan
        if deliverables: render deliverable tree
    case failed:
        render "✗ stepID (persona) (final_duration)" in red
    case skipped:
        render "— stepID (persona)" in muted color
    case cancelled:
        render "⊛ stepID (persona)" in warning color
```

**Key decisions**:
- Deliverable tree is preserved for completed steps (existing behavior)
- Tool activity line is preserved for the running step (existing behavior)
- The blank line between completed steps and running step is removed — the step list is now a continuous ordered list
- Colors match existing codebase conventions (lipgloss color codes)

### Change 5: Rewrite `renderStepStatusPanel()` in Dashboard (dashboard.go)

Replace the current method that only shows the running step with a method that iterates ALL steps using `StepOrder` (deterministic ordering, not random map iteration):

```go
for _, stepID := range ctx.StepOrder {
    state := ctx.StepStatuses[stepID]
    icon := d.getStatusIcon(state)
    persona := ctx.StepPersonas[stepID]
    // render: icon stepID (persona) [duration]
}
```

### Change 6: Update `getStatusIcon()` in Dashboard (dashboard.go)

The existing `getStatusIcon()` method uses `"-"` for skipped and `"X"` for cancelled. Update to use the spec-mandated indicators:

| State | Current | New |
|-------|---------|-----|
| Skipped | `"-"` | `"—"` (em dash) |
| Cancelled | `"X"` | `"⊛"` |

### Test Plan

| Test | File | Validates |
|------|------|-----------|
| All 6 states render with correct indicators | `bubbletea_model_test.go` | FR-003 through FR-007 |
| Step order matches StepOrder slice | `bubbletea_model_test.go` | FR-008 |
| Persona shown for all steps | `bubbletea_model_test.go` | FR-002 |
| Running step shows elapsed time | `bubbletea_model_test.go` | FR-010 |
| Completed step shows final duration | `bubbletea_model_test.go` | FR-010 |
| At most one spinner at a time | `bubbletea_model_test.go` | FR-012 |
| Single-step pipeline renders correctly | `bubbletea_model_test.go` | Edge case |
| StepPersonas populated in toPipelineContext | `progress_test.go` | C-4 |
| Dashboard renders all steps in order | `dashboard_test.go` | FR-001, FR-008 |
| Existing tests pass unchanged | All test files | SC-006 |

## Complexity Tracking

_No constitution violations. No complexity justifications needed._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|-------------------------------------|
| (none) | — | — |
