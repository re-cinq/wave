# Feature Specification: Pipeline Step Visibility in Default Run Mode

**Feature Branch**: `100-pipeline-step-visibility`
**Created**: 2026-02-13
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/99"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View All Pipeline Steps During Execution (Priority: P1)

As a pipeline operator running `wave run` in default (non-verbose) mode, I want to see all pipeline steps listed beneath the progress bar — not just the currently active step — so that I can understand the full scope of work, what has completed, what is running now, and what remains.

**Why this priority**: This is the core request. Currently the TUI only renders the active step, leaving operators blind to pipeline breadth. Showing all steps is the fundamental behavior change that makes the feature useful.

**Independent Test**: Can be fully tested by running any multi-step pipeline in default mode and verifying that all steps (pending, active, completed) appear in the step list area with correct status indicators.

**Acceptance Scenarios**:

1. **Given** a pipeline with 4 steps is started in default run mode, **When** the first step begins executing, **Then** all 4 steps are displayed: the first step shows a spinner indicator, and the remaining 3 steps show a pending indicator (`○`).
2. **Given** a pipeline with 4 steps where step 1 has completed and step 2 is running, **When** the display refreshes, **Then** step 1 shows a completed indicator (`✓`), step 2 shows a spinner indicator, and steps 3-4 show pending indicators (`○`).
3. **Given** a pipeline completes all steps successfully, **When** the final display renders, **Then** all steps show the completed indicator (`✓`).

---

### User Story 2 - Identify Step Persona Assignments at a Glance (Priority: P1)

As a pipeline operator, I want each step in the list to display both its step name and assigned persona so that I can understand which AI persona is responsible for each phase of work without consulting the pipeline definition file.

**Why this priority**: Step-persona association is essential context that was already shown for the active step. Extending it to all steps makes the full pipeline readable. It shares the same priority as US-1 because step names alone are insufficient for pipeline comprehension.

**Independent Test**: Can be tested by running a pipeline and verifying each listed step renders as `<indicator> <step-name> (<persona-name>)`.

**Acceptance Scenarios**:

1. **Given** a pipeline step named `scan-issues` assigned to persona `github-analyst`, **When** the step list renders, **Then** that step displays as `○ scan-issues (github-analyst)` (or the equivalent indicator for its current state).
2. **Given** a step has transitioned from pending to active, **When** the display updates, **Then** the persona name remains visible alongside the step name and the indicator changes to the spinner.

---

### User Story 3 - See Elapsed Time for Active Step (Priority: P2)

As a pipeline operator, I want the currently active step to show its elapsed execution time so that I can gauge whether execution is progressing or potentially stalled.

**Why this priority**: Elapsed time is already displayed for the active step in the current TUI. This story ensures the behavior is preserved in the new multi-step display layout. It is lower priority because the existing behavior already covers this — it just needs to be maintained, not built from scratch.

**Independent Test**: Can be tested by running a pipeline, observing the active step, and verifying elapsed time increments in real time.

**Acceptance Scenarios**:

1. **Given** a step has been running for 15 seconds, **When** the display renders, **Then** the active step line shows the elapsed time as `(15s)` or equivalent human-readable duration.
2. **Given** a step transitions from active to completed, **When** the display updates, **Then** the live elapsed timer stops and is replaced by the final duration (e.g., `(23.5s)`). The completed indicator (`✓`) replaces the spinner. This preserves the existing codebase behavior where completed steps show their total duration.

---

### User Story 4 - Distinguish Skipped Steps (Priority: P3)

As a pipeline operator, I want skipped steps (due to conditional logic or pipeline configuration) to display a distinct skip indicator so that I can differentiate them from steps that haven't run yet versus steps that were intentionally bypassed.

**Why this priority**: Skipped steps are an edge case that occurs with conditional pipelines. While less common, misinterpreting a skipped step as pending could cause confusion. This is lower priority because skip scenarios are infrequent in typical usage.

**Independent Test**: Can be tested by configuring a pipeline with a conditional step that evaluates to skip, running the pipeline, and verifying the skipped step shows a distinct indicator.

**Acceptance Scenarios**:

1. **Given** a pipeline where step 3 is skipped due to a condition, **When** the step list renders after that step is skipped, **Then** step 3 displays a skip indicator (`—`) distinct from both the pending (`○`) and completed (`✓`) indicators.
2. **Given** a step is skipped, **When** the display renders, **Then** the skipped step still shows its step name and persona for traceability.

---

### User Story 5 - Distinguish Failed Steps (Priority: P3)

As a pipeline operator, I want failed steps to display a distinct failure indicator so that I can immediately identify which step caused a pipeline failure when viewing the final step list.

**Why this priority**: Failed steps are critical for debugging but the failure state is already surfaced through other means (error output, exit codes). A visual indicator in the step list is additive rather than essential.

**Independent Test**: Can be tested by running a pipeline where a step fails and verifying the failed step shows a distinct failure indicator.

**Acceptance Scenarios**:

1. **Given** a pipeline where step 2 fails during execution, **When** the step list renders, **Then** step 2 displays a failure indicator (e.g., `✗`) distinct from other state indicators.
2. **Given** a step has failed, **When** the display renders, **Then** subsequent pending steps retain their pending indicator (they were never executed).

---

### Edge Cases

- **Single-step pipeline**: When a pipeline has only one step, the display still renders the step list with that single step and its correct indicator (no regression from current behavior).
- **Step transitions during render cycle**: When a step completes and the next step starts between render frames, the display must update atomically — no frame should show two steps with spinner indicators simultaneously.
- **Rapid step completion**: When a step completes in under one second (e.g., a trivial validation step), the display must still show the completed state for that step and not skip rendering it.
- **Pipeline with many steps**: For pipelines exceeding terminal height, the display should not crash or corrupt rendering. The initial implementation may allow the step list to overflow the terminal; scrolling/truncation is explicitly out of scope per the issue and can be addressed in a follow-up.
- **Terminal resize during execution**: The display must handle terminal width/height changes gracefully without corrupting the step list layout.
- **Non-TTY output**: When stdout is not a terminal (e.g., piped to a file), the step display behavior should degrade gracefully or be suppressed (consistent with existing non-TTY behavior).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The default run mode step list area MUST render all pipeline steps, not just the currently active step.
- **FR-002**: Each step in the list MUST display its step name and assigned persona name in the format `<indicator> <step-name> (<persona-name>)`.
- **FR-003**: The currently active step MUST display an animated spinner indicator (matching the existing spinner behavior).
- **FR-004**: Completed steps MUST display a checkmark indicator (`✓`).
- **FR-005**: Pending (not yet started) steps MUST display a circle indicator (`○`).
- **FR-006**: Skipped steps MUST display a dash indicator (`—`), visually distinguishable from pending and completed states. This aligns with the existing `dashboard.go` convention that uses `"-"` for skipped steps.
- **FR-007**: Failed steps MUST display a cross mark indicator (`✗`), visually distinguishable from other states. This aligns with the existing `charSet.CrossMark` used in `progress.go` and `dashboard.go`.
- **FR-008**: The step list ordering MUST match the pipeline definition order (as declared in the pipeline YAML).
- **FR-009**: The display MUST update in real time as steps transition between states (pending → running → completed/failed/skipped).
- **FR-010**: The active step MUST display a live elapsed execution time that updates each render frame. Completed steps MUST display their final static duration.
- **FR-011**: This feature MUST apply only to the default run mode. Verbose mode (`--verbose`) rendering MUST NOT be affected.
- **FR-012**: At most one step MUST display the active/spinner indicator at any given time.

### Key Entities

- **Step Display Entry**: Represents a single step in the rendered step list. Attributes: step name, persona name, current state (pending/running/completed/failed/skipped), elapsed time (for running state only).
- **Step State**: The lifecycle state of a pipeline step. Possible values: `not_started`, `running`, `completed`, `failed`, `skipped`, `cancelled`. Maps to visual indicators in the display. The `cancelled` state (which exists in the codebase as `StateCancelled`) MUST render with a distinct indicator (`⊛`) and use the warning color, consistent with the existing `progress.go` rendering. Cancelled steps are treated as a terminal state alongside completed, failed, and skipped.
- **Pipeline Context (display)**: The aggregate view of all pipeline steps with their ordering and states, consumed by the rendering layer to produce the step list. Must include: `StepOrder []string` for deterministic ordering, `StepStatuses map[string]ProgressState` for state lookup, `StepPersonas map[string]string` for persona name lookup (new field required by this feature), and `StepDurations map[string]int64` for completed step durations.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: When running any multi-step pipeline in default mode, 100% of pipeline steps are visible in the step list area at all times during execution.
- **SC-002**: Each step's displayed state indicator matches its actual execution state within one render cycle (no stale indicators persisting across frames).
- **SC-003**: The step list order matches the pipeline YAML definition order for every pipeline tested.
- **SC-004**: All six visual states (pending, active, completed, skipped, failed, cancelled) are visually distinct from each other when displayed in a standard terminal.
- **SC-005**: Verbose mode (`--verbose`) output is unchanged by this feature (no regressions in existing verbose display).
- **SC-006**: Existing unit and integration tests for the display, event, and pipeline packages continue to pass without modification (except tests directly related to the changed rendering behavior).

## Clarifications

The following ambiguities were identified during review and resolved based on codebase patterns and existing architecture:

### C-1: Target Rendering Component

**Ambiguity**: The spec references "default run mode" but the codebase has three rendering paths: `BubbleTeaProgressDisplay` (bubbletea TUI with `charmbracelet/bubbletea`), `ProgressDisplay` (ANSI escape code renderer using `Dashboard`), and `BasicProgressDisplay` (plain text fallback for non-TTY).

**Resolution**: The primary target is `BubbleTeaProgressDisplay` and its `ProgressModel.View()` method (`bubbletea_model.go`), which is the active default TUI renderer. The `renderCurrentStep()` method in `bubbletea_model.go` is where the step list is rendered and where pending steps must be added. The `ProgressDisplay`/`Dashboard` path should also be updated for consistency, but the bubbletea path is the authoritative implementation. The `BasicProgressDisplay` (non-TTY) is out of scope per FR-011 and the Non-TTY edge case.

**Rationale**: The bubbletea model is the active rendering path used by `BubbleTeaProgressDisplay`, which is instantiated for TTY environments. The older `Dashboard`/`ProgressDisplay` is a secondary fallback. Focusing on the bubbletea path ensures the most commonly used code path is correct.

### C-2: Specific Indicator Characters

**Ambiguity**: FR-006 proposed multiple options (`⊘` or `—`) for skipped steps, and FR-007 suggested `✗` without being definitive.

**Resolution**: Fixed indicators are:
- **Pending**: `○` (open circle) — consistent with `progress.go:189` and `dashboard.go:305`
- **Running**: Animated spinner from braille character set (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) — consistent with `bubbletea_model.go:299`
- **Completed**: `✓` (check mark) — consistent with `charSet.CheckMark` in `progress.go:175`
- **Failed**: `✗` (cross mark) — consistent with `charSet.CrossMark` in `progress.go:177`
- **Skipped**: `—` (em dash) — consistent with `dashboard.go:299` which uses `"-"`
- **Cancelled**: `⊛` (circled asterisk) — consistent with `progress.go:187`

**Rationale**: Aligning with existing codebase conventions minimizes inconsistency and ensures the `UnicodeCharSet`/`AsciiOnly` fallback system continues to work.

### C-3: Completed Step Duration vs. Elapsed Time Removal

**Ambiguity**: US-3 Scenario 2 originally stated "the elapsed time indicator is removed" on completion, but the existing codebase shows completed steps with their final duration (e.g., `✓ step-name (2.3s)` in `bubbletea_model.go:275`).

**Resolution**: Completed steps MUST show their final duration (not a live timer). The live elapsed timer stops on completion and is replaced by the static final duration. This preserves existing behavior that users rely on for post-execution analysis.

**Rationale**: Showing the final duration for completed steps is standard practice in CI/CD tooling and is already implemented. Removing it would be a regression.

### C-4: Persona Data Availability for Non-Running Steps

**Ambiguity**: FR-002 requires all steps to display persona names, but `PipelineContext.StepStatuses` is a `map[string]ProgressState` which does not carry persona information. Currently only the running step's persona is tracked via `CurrentPersona`. Pending/completed steps need persona data too.

**Resolution**: The `PipelineContext` type must be extended with a `StepPersonas map[string]string` field (mapping stepID to persona name). This map is populated when steps are registered via `AddStep()` and propagated through `toPipelineContext()`. The `renderCurrentStep()` method in `bubbletea_model.go` will use this map to show persona names for all steps, not just the running one.

**Rationale**: The pipeline YAML already declares `persona` per step (`pipeline/types.go:52`), and `AddStep()` already receives persona as a parameter. The data is available at registration time; it just needs to be persisted in the context passed to the renderer.

### C-5: Cancelled State Visual Indicator

**Ambiguity**: The spec's Key Entities section listed `cancelled` as a possible step state, but no user story, functional requirement, or indicator was defined for it.

**Resolution**: Added cancelled state indicator (`⊛` with warning color) to the Step State entity definition. A dedicated user story is NOT added because cancellation is an infrastructure concern (e.g., Ctrl+C or context timeout), not a pipeline-design concern like skip/fail. The rendering implementation MUST handle `StateCancelled` to avoid a panic or blank indicator, but no new acceptance scenario is required. FR-009's state transitions implicitly cover cancelled as a terminal state.

**Rationale**: The cancelled state already exists in `display/types.go:16` and is rendered in `progress.go:187`. Ignoring it would leave a gap in the step list rendering.
