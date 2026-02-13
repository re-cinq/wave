# Research: Pipeline Step Visibility in Default Run Mode

**Feature Branch**: `100-pipeline-step-visibility`
**Date**: 2026-02-13
**Spec**: `specs/100-pipeline-step-visibility/spec.md`

## Phase 0 — Research Findings

### 1. Current Rendering Architecture

**Decision**: Modify `BubbleTeaProgressDisplay` and `ProgressModel.View()` as the primary target.
**Rationale**: This is the active rendering path for TTY default mode. The `Dashboard`/`ProgressDisplay` path is a secondary fallback that also needs updating for consistency.
**Alternatives Rejected**:
- Creating a new rendering component — unnecessary complexity, existing architecture is well-suited
- Modifying only `Dashboard` — it's the secondary path; `BubbleTeaProgressDisplay` is the active default

#### Three Rendering Paths

| Path | Component | Used When | In Scope |
|------|-----------|-----------|----------|
| **BubbleTea TUI** | `BubbleTeaProgressDisplay` → `ProgressModel.View()` | TTY + ANSI support (default `auto` mode) | **Primary target** |
| **ANSI text** | `ProgressDisplay` → `Dashboard.Render()` | `--output text` or non-TTY fallback | Secondary update |
| **Basic text** | `BasicProgressDisplay` | Non-TTY pipe mode | Out of scope (FR-011) |

#### Current BubbleTea `renderCurrentStep()` Behavior (bubbletea_model.go:247-344)

The current implementation in `renderCurrentStep()`:
1. Iterates `StepOrder` to collect completed steps, then shows the single running step
2. For **completed** steps: renders `✓ stepID (duration)` with deliverable tree
3. For **running** step: renders spinner + stepID + persona + elapsed time + current action + tool activity
4. **Does NOT render pending, failed, skipped, or cancelled steps**

This is the core gap the feature addresses.

#### Current Dashboard `renderStepStatusPanel()` Behavior (dashboard.go:155-185)

The `Dashboard` path iterates `StepStatuses` (a map, so order is non-deterministic) and only shows the current running step with a pulsating effect. It also lacks pending/failed/skipped step rendering.

### 2. Data Flow Analysis

**Decision**: Extend `PipelineContext` with `StepPersonas map[string]string` field.
**Rationale**: The persona name is available at step registration time (`AddStep(stepID, stepName, persona)`) but is not propagated through `PipelineContext` to the renderer. Only `CurrentPersona` is set for the running step. All other steps lose their persona association.
**Alternatives Rejected**:
- Embedding persona in step name string — loses structured data, breaks formatting
- Querying pipeline definition at render time — violates separation of concerns, renderer shouldn't need pipeline YAML

#### Complete Data Flow

```
Pipeline YAML (persona per step)
  → pipeline.Execute() registers steps
    → BubbleTeaProgressDisplay.AddStep(stepID, stepName, persona)
      → steps map[string]*StepStatus (stores persona per step)
        → toPipelineContext() converts to PipelineContext
          → PipelineContext.StepStatuses (has states)
          → PipelineContext.StepOrder (has ordering)
          → PipelineContext.StepDurations (has timing)
          → PipelineContext.CurrentPersona (ONLY for running step) ← GAP
            → ProgressModel.View() → renderCurrentStep()
```

**Gap**: `toPipelineContext()` in `bubbletea_progress.go:268` does NOT populate a `StepPersonas` map. Only `CurrentPersona` is set from the running step. The renderer cannot show persona names for completed/pending steps.

### 3. Step State Indicators

**Decision**: Use existing codebase-consistent indicator characters for all 6 states.
**Rationale**: The codebase already defines these characters in `progress.go` and `dashboard.go`. Reusing them ensures consistency with existing ASCII fallback mechanisms.
**Alternatives Rejected**:
- Custom emoji indicators — inconsistent with existing Unicode charset approach
- Colored text-only indicators (no icons) — less scannable, accessibility concern

| State | Character | Source | Color |
|-------|-----------|--------|-------|
| Pending | `○` | `progress.go:189`, `dashboard.go:305` | Muted (gray) |
| Running | Braille spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) | `bubbletea_model.go:299` | Yellow (bright) |
| Completed | `✓` | `charSet.CheckMark` | Cyan (bright) |
| Failed | `✗` | `charSet.CrossMark` | Red |
| Skipped | `—` | `dashboard.go:299` uses `"-"` | Muted (gray) |
| Cancelled | `⊛` | `progress.go:187` | Warning (yellow) |

### 4. Step Display Format

**Decision**: Use format `<indicator> <step-name> (<persona-name>) [timing-info]` for all steps.
**Rationale**: Matches FR-002 requirement and extends the existing pattern from `renderCurrentStep()` which already shows `spinner stepID (persona) (elapsed)` for the running step.
**Alternatives Rejected**:
- Table layout — overkill for a step list, harder to scan
- Two-line format per step — wastes vertical space

#### Display Format Per State

| State | Format Example | Notes |
|-------|---------------|-------|
| Pending | `○ scan-issues (github-analyst)` | No timing |
| Running | `⠋ implement (implementer) (23s)` | Live elapsed timer |
| Completed | `✓ navigate (navigator) (12.3s)` | Final static duration |
| Failed | `✗ validate (reviewer) (45.2s)` | Final static duration |
| Skipped | `— optional-step (checker)` | No timing |
| Cancelled | `⊛ interrupted-step (worker)` | No timing |

### 5. Existing Test Coverage Analysis

**Decision**: Add new tests for the new rendering logic; existing tests should pass unchanged.
**Rationale**: SC-006 requires existing tests to pass. The new functionality adds to the renderer output without breaking the existing completed/running step rendering.
**Alternatives Rejected**:
- Modifying existing test assertions — violates SC-006 requirement

#### Existing Test Files

| File | Tests | Impact |
|------|-------|--------|
| `bubbletea_model_test.go` | Tests ProgressModel View, Update, Init | Need new test cases for all-step rendering |
| `progress_test.go` | Tests ProgressDisplay, StepStatus, AddStep | Need new test for `StepPersonas` propagation |
| `dashboard_test.go` | Tests Dashboard rendering | Need update for all-step rendering |
| `types_test.go` | Tests PipelineContext, state helpers | Need test for new `StepPersonas` field |

### 6. Thread Safety and Atomicity

**Decision**: No new mutexes needed; use existing locking in `BubbleTeaProgressDisplay`.
**Rationale**: The existing `sync.Mutex` in `BubbleTeaProgressDisplay` protects all state mutations. The bubbletea framework handles View() calls on a single goroutine. The `toPipelineContext()` method already runs under the lock and creates a snapshot.
**Alternatives Rejected**:
- Adding per-step locks — unnecessary given existing mutex already protects the map
- Using channels for state updates — bubbletea already uses its own message-passing model

**Edge Case (FR-012 / atomicity)**: The spec requires at most one spinner at a time. This is guaranteed by the existing `currentStepID` tracking in `BubbleTeaProgressDisplay.updateFromEvent()` — when a step completes, `currentStepID` is cleared, and when a new step starts, it's set to the new step. The renderer only shows a spinner for `StepStatuses[stepID] == StateRunning`.

### 7. Performance Considerations

**Decision**: Rendering all steps adds O(N) string concatenation where N is the number of steps. This is negligible for typical pipelines (2-10 steps).
**Rationale**: The bubbletea model re-renders the full View() on every tick (33ms). Adding 5-10 extra lines of text to the output is insignificant compared to the existing rendering cost (progress bar animation, lipgloss styling).
**Alternatives Rejected**:
- Lazy rendering with dirty flags — premature optimization for N < 20
- Virtual list for large pipelines — explicitly out of scope per spec edge case

### 8. Constitution Compliance Pre-Check

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Compliant | No new dependencies |
| P2: Manifest as Source | ✅ Compliant | No manifest changes |
| P3: Persona-Scoped Execution | ✅ Compliant | No execution changes |
| P4: Fresh Memory | ✅ Compliant | Display-only changes |
| P5: Navigator-First | ✅ Compliant | No pipeline changes |
| P6: Contracts at Handovers | ✅ Compliant | No contract changes |
| P10: Observable Progress | ✅ Enhanced | More visible progress |
| P12: Minimal State Machine | ✅ Compliant | Uses existing states |
| P13: Test Ownership | ✅ Required | Must run full test suite |
