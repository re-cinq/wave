# Feature Specification: Pipeline Composition UI — Sequence Builder with Artifact Flow Visualization

**Feature Branch**: `261-tui-compose-ui`  
**Created**: 2026-03-07  
**Status**: Clarified  
**Issue**: [#261](https://github.com/re-cinq/wave/issues/261) — Part 10/10 of TUI epic [#251](https://github.com/re-cinq/wave/issues/251)  
**Input**: User description: "Implement the pipeline composition UI accessible via the `s` key when an available pipeline is selected. This enables users to build multi-pipeline sequences where output artifacts of one pipeline become inputs to the next."

## Context

This is the final sub-issue of the TUI epic (#251). All prior issues (#252–#260) are merged, providing:
- Header bar with animated Wave logo (#253)
- Pipeline list left pane with Running/Finished/Available sections (#254)
- Pipeline detail right pane (#255)
- Pipeline launch flow with argument menu and executor integration (#256)
- Live output streaming (#257)
- Finished pipeline actions (chat, branch checkout, diff) (#258)
- Detail views for Personas, Contracts, Skills, Health (#259)
- CLI compliance (NO_COLOR, --no-tui, --json, TTY detection) (#260)

**Dependency note**: This feature is blocked on #249 (cross-pipeline artifact handoff) for actually *executing* sequences. However, the UI itself (sequence builder, artifact flow visualization, compatibility validation, grouped running display) can be built and tested independently. The "Start sequence" action should either invoke the artifact handoff system when available, or show a clear message that sequential execution requires #249.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Enter Compose Mode and Build a Sequence (Priority: P1)

A user wants to chain multiple pipelines together to form a multi-stage workflow (e.g., `speckit-flow → wave-evolve → wave-review`). They select an available pipeline in the left pane and press `s` to enter compose mode, where they can add, remove, and reorder pipelines in the sequence.

**Why this priority**: This is the core interaction — without the ability to build a sequence, nothing else in this feature matters. It unlocks multi-pipeline workflows, which is Wave's primary value proposition for complex development tasks.

**Independent Test**: Can be fully tested by pressing `s` on an available pipeline, verifying the compose mode UI appears with the selected pipeline as the first item, adding/removing/reordering additional pipelines, and pressing `Esc` to cancel without side effects.

**Acceptance Scenarios**:

1. **Given** the user is in the Pipelines view with focus on the left pane and an available pipeline is selected, **When** the user presses `s`, **Then** the compose mode UI opens as a modal state within the Pipelines view: the left pane switches to the sequence list and the right pane switches to the artifact flow visualization, and focus stays on the left pane (sequence list).

2. **Given** compose mode is open with one pipeline in the sequence, **When** the user presses `a`, **Then** a pipeline picker appears listing all available pipelines (including those already in the sequence, since duplicate pipelines are allowed), and the user can select one to append to the sequence.

3. **Given** compose mode is open with two or more pipelines, **When** the user presses `shift+↑` or `shift+↓` on a selected sequence item, **Then** the item moves up or down in the sequence order, and the artifact flow visualization updates to reflect the new ordering.

4. **Given** compose mode is open with two or more pipelines, **When** the user presses `x` on a selected sequence item, **Then** that pipeline is removed from the sequence, and the artifact flow visualization updates accordingly.

5. **Given** compose mode is open, **When** the user presses `Esc`, **Then** compose mode closes without starting any pipeline, and the TUI returns to the normal Pipelines view with left pane focused.

---

### User Story 2 - Artifact Flow Visualization (Priority: P2)

While building a sequence, the user needs to see which artifacts flow between the chained pipelines — specifically which outputs from one pipeline are available as inputs to the next — so they can verify the sequence is coherent before starting it.

**Why this priority**: The artifact flow visualization is what makes the sequence builder useful rather than just a list. Without it, users must manually verify compatibility by reading pipeline definitions.

**Independent Test**: Can be tested by building a sequence of 2–3 known pipelines and verifying that the right pane shows each pipeline's output artifacts and the next pipeline's expected inputs, with visual indicators for matches and mismatches.

**Acceptance Scenarios**:

1. **Given** compose mode is open with two or more pipelines in the sequence, **When** the user views the right pane, **Then** the artifact flow visualization shows each pipeline with its output artifacts listed below it, with connecting indicators (arrows or lines) to the next pipeline's expected inputs.

2. **Given** a sequence where pipeline A's last step produces an output artifact named `spec_info` and pipeline B's first step declares an `inject_artifact` with a matching artifact name, **When** the artifact flow is rendered, **Then** the artifact shows a successful match indicator between the two pipelines.

3. **Given** a sequence where pipeline B's first step declares a required `inject_artifact` that no output artifact from pipeline A's last step can satisfy by name, **When** the artifact flow is rendered, **Then** the missing artifact is highlighted with a warning indicator and the incompatibility is described in text.

---

### User Story 3 - Artifact Compatibility Validation (Priority: P2)

Before starting a sequence, the system validates that artifact outputs from each pipeline match the expected inputs of the next pipeline. Incompatibilities produce visible warnings so the user can fix the sequence before wasting compute.

**Why this priority**: Co-equal with visualization since validation prevents wasted pipeline runs. Together with User Story 2, they form the "informed decision" capability.

**Independent Test**: Can be tested by constructing a sequence with known incompatible pipelines and verifying that warnings appear before the start action is allowed (or that start proceeds with explicit acknowledgment).

**Acceptance Scenarios**:

1. **Given** a sequence with compatible artifact flows across all pipeline boundaries, **When** the user views the compose mode, **Then** a "Ready to start" or equivalent positive indicator is shown.

2. **Given** a sequence where one pipeline's outputs do not satisfy the next pipeline's required inputs, **When** the user views the compose mode, **Then** a warning message identifies the specific missing artifacts and which pipeline boundary has the mismatch.

3. **Given** a sequence with artifact incompatibilities, **When** the user presses `Enter` to start, **Then** a confirmation prompt warns about the incompatibilities and asks the user to confirm or cancel.

---

### User Story 4 - Start a Composed Sequence (Priority: P3)

The user presses `Enter` to launch the built sequence. The sequence starts executing (or shows a blocking message if #249 is not yet available). Once running, the sequence appears as a grouped item in the Running section.

**Why this priority**: Depends on #249 for actual execution. The UI can be built and tested with a placeholder/message, but real execution is gated on the cross-pipeline artifact handoff implementation.

**Independent Test**: Can be tested by pressing `Enter` on a valid sequence. If #249 is available, verify the sequence starts and appears grouped in Running. If #249 is not available, verify a clear informational message is shown.

**Acceptance Scenarios**:

1. **Given** a valid sequence with no incompatibilities and cross-pipeline artifact handoff is available, **When** the user presses `Enter`, **Then** the sequence begins execution and compose mode closes.

2. **Given** a running sequence, **When** the user views the Running section in the left pane, **Then** the sequence appears as a grouped item showing a generated label (e.g., "speckit-flow → wave-evolve → wave-review"), with per-pipeline progress indication.

3. **Given** cross-pipeline artifact handoff (#249) is not implemented, **When** the user presses `Enter` on a sequence, **Then** a message informs the user that sequential execution is not yet available and lists what is needed.

4. **Given** a running sequence with pipeline 2 of 3 active, **When** the user selects the grouped sequence in the Running section, **Then** the right pane shows the currently executing pipeline's live output, plus a summary of completed/pending pipelines in the sequence.

---

### User Story 5 - CLI Equivalent for Sequence Execution (Priority: P3)

Users working in scripts or non-interactive environments need a CLI equivalent to compose and run pipeline sequences without the TUI.

**Why this priority**: CLI parity is important for automation and CI/CD but is secondary to the interactive TUI experience for this feature.

**Independent Test**: Can be tested by running `wave compose p1 p2 p3` from the command line and verifying the same sequence execution behavior as the TUI.

**Acceptance Scenarios**:

1. **Given** a terminal with the wave binary available, **When** the user runs `wave compose p1 p2 p3` (positional pipeline names), **Then** artifact compatibility is validated and the sequence starts if valid, or errors are printed if invalid.

2. **Given** a terminal, **When** the user runs `wave compose --validate-only p1 p2 p3`, **Then** artifact compatibility is checked and results are printed without starting execution.

---

### Edge Cases

- What happens when the user tries to add the same pipeline twice to a sequence? The system should allow it (a pipeline can be run multiple times in a sequence) but display a notice indicating the duplicate.
- What happens when only one pipeline is in the sequence and the user presses `Enter`? It should behave the same as a normal single-pipeline launch (delegate to the existing `PipelineLauncher`).
- What happens when the terminal is too narrow to display the artifact flow visualization? The visualization should degrade gracefully — show a text-only summary of compatibility instead of the graphical flow diagram.
- What happens when a pipeline in the sequence has no declared output artifacts? The flow visualization should show "No artifacts" for that pipeline and warn that the next pipeline may not receive expected inputs.
- What happens when the user presses `s` while a pipeline is already running? Compose mode should still open (composing is independent of running state).
- What happens when compose mode is open and a running pipeline finishes? The finish event should update the left pane behind the compose mode; compose mode is not interrupted.
- What happens when there are no other available pipelines to add to the sequence? The `a` (add) action should be disabled or show "No additional pipelines available".
- What happens when the user presses `s` on a running or finished pipeline? The `s` key should have no effect — it is only active when an available pipeline is selected.
- What happens when the user removes all pipelines from the sequence? Compose mode should remain open with an empty list and the `Enter` (start) action disabled.
- What happens when a pipeline in the middle of a sequence is removed? The artifact flow should re-validate adjacent pipelines that are now directly connected.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST open compose mode when user presses `s` with an available pipeline selected in the Pipelines view left pane.
- **FR-002**: Compose mode MUST display a sequence list in the left pane (replacing the pipeline list) showing pipelines in execution order, with the initially selected pipeline as the first item.
- **FR-003**: System MUST support adding pipelines to the sequence via `a` key, showing a picker of available pipelines.
- **FR-004**: System MUST support removing pipelines from the sequence via `x` key on the selected item.
- **FR-005**: System MUST support reordering pipelines within the sequence via `shift+↑`/`shift+↓` keys (plain `↑`/`↓` navigate the cursor within the sequence list).
- **FR-006**: System MUST display an artifact flow visualization in the right pane (replacing the pipeline detail) showing output artifacts from each pipeline and input expectations of the next pipeline in the sequence.
- **FR-007**: System MUST validate artifact compatibility between adjacent pipelines in the sequence and display warnings for mismatches. Compatibility is determined by matching the last step's `output_artifacts[].name` of pipeline N against the first step's `memory.inject_artifacts[].artifact` of pipeline N+1.
- **FR-008**: System MUST allow starting the sequence via `Enter` key, with a confirmation prompt if artifact incompatibilities exist.
- **FR-009**: System MUST cancel compose mode without side effects via `Esc` key, returning to the normal Pipelines view.
- **FR-010**: Running sequences MUST appear as grouped items in the Running section of the left pane, showing per-pipeline progress within the group.
- **FR-011**: Status bar MUST update to show compose-mode-specific keybindings when compose mode is active (e.g., `a: add  x: remove  Shift+↑↓: reorder  Enter: start  Esc: cancel`).
- **FR-012**: System MUST provide a CLI command `wave compose p1 p2 p3` for sequence validation and execution. The existing `wave run` command is NOT modified (it retains its `[pipeline] [input]` signature).
- **FR-013**: If cross-pipeline artifact handoff (#249) is not available, the `Enter` action in compose mode MUST show an informational message instead of silently failing.
- **FR-014**: Artifact flow visualization MUST degrade gracefully on terminals narrower than 120 columns by switching to a text-only compatibility summary.
- **FR-015**: The `s` key MUST have no effect when pressed in non-Pipelines views, when the right pane is focused, or when a non-available pipeline item is selected.

### Key Entities

- **Sequence**: An ordered list of pipeline references to execute in series, where artifacts from one pipeline are handed off to the next. Contains pipeline identifiers, their resolved artifact compatibility status, and an overall readiness indicator.
- **ArtifactFlow**: A directional mapping between a source pipeline's last step `output_artifacts` and a target pipeline's first step `inject_artifacts`. Each flow entry has a match status: compatible (output name matches inject artifact name), missing (inject artifact expected but no matching output), or unmatched (output produced but not consumed by next pipeline). Optional inject artifacts (where `ArtifactRef.Optional == true`) do not produce warnings when unmatched.
- **CompatibilityResult**: The aggregated result of validating artifact flows across all boundaries in a sequence. Contains per-boundary results, an overall status (valid, warning, error), and human-readable diagnostic messages for each issue found.

## Clarifications

The following ambiguities were identified and resolved during the clarify phase:

### C1: Compose mode architecture — modal state vs. new ViewType

**Ambiguity**: The spec did not clarify whether compose mode is a new `ViewType` (like Personas, Contracts, etc. which cycle via Tab) or a modal state within the Pipelines view (like the launch form or live output).

**Resolution**: Compose mode is a **modal state within the Pipelines view**, not a new ViewType. It replaces the left pane content (pipeline list → sequence list) and right pane content (pipeline detail → artifact flow visualization) while compose mode is active. This follows the established pattern of `stateConfiguring` and `stateRunningLive` in the existing `PipelineDetailModel`. Tab-cycling views should be disabled while compose mode is active (same as when a form is active). Pressing `Esc` exits compose mode and restores the normal Pipelines view.

**Rationale**: Adding a new ViewType would make compose mode permanently accessible via Tab, which is wrong — it's a transient action initiated by `s` on a specific pipeline. The modal state pattern matches how the launch form already works within the Pipelines view.

### C2: Reorder keys conflict with navigation

**Ambiguity**: FR-005 and User Story 1 Scenario 3 specified `↑`/`↓` for reordering, but these keys are universally used for cursor navigation in the existing TUI (pipeline list, persona list, contract list, etc.).

**Resolution**: Use `shift+↑` / `shift+↓` for reordering (moving the selected item up/down in the sequence). Plain `↑`/`↓` retain their standard meaning: cursor navigation within the sequence list. This ensures consistency with the existing TUI keyboard conventions.

**Rationale**: Overloading `↑`/`↓` for both navigation and reordering would require a mode toggle or modifier key anyway. `shift+arrow` is a well-established convention for "move item" vs. "move cursor" in list UIs (VS Code, Sublime Text, etc.).

### C3: Cross-pipeline artifact matching semantics

**Ambiguity**: The ArtifactFlow entity said artifacts match by "name and type", but within a single pipeline, artifacts are linked via explicit `inject_artifacts` referencing a specific step and artifact name. The `type` field on `ArtifactDef` is optional and often omitted. How should cross-pipeline matching work?

**Resolution**: Cross-pipeline matching uses **artifact name only** — specifically, matching the `output_artifacts[].name` of the last step in pipeline N against the `inject_artifacts[].artifact` name of the first step in pipeline N+1. Type matching is not required because the existing codebase treats artifact names as the primary identifier and types are optional metadata. Optional inject artifacts (where `ArtifactRef.Optional == true`) produce no warning when unmatched.

**Rationale**: This mirrors how intra-pipeline artifact injection already works via `ArtifactRef.Artifact` (name-based). Adding type matching would create false negatives since many pipeline definitions omit types entirely. The heuristic of "last step outputs → first step inputs" is a reasonable default for cross-pipeline flow — it covers the common case where pipelines are designed as pipeline-level units with defined boundaries.

### C4: CLI command form — `wave run` modification vs. new `wave compose` command

**Ambiguity**: FR-012 proposed both `wave run p1 p2 p3` (multiple positional args) and `wave compose --sequence p1,p2,p3`. The current `wave run` command signature is `[pipeline] [input]` with `cobra.MaximumNArgs(2)`, so adding multiple pipeline positional args would be a **breaking change** that conflicts with the existing `wave run <pipeline> <input>` pattern.

**Resolution**: Add a **new `wave compose` subcommand** (`wave compose p1 p2 p3`) instead of modifying `wave run`. The `wave run` command retains its current `[pipeline] [input]` signature unchanged. The `wave compose` command accepts variadic pipeline names as positional args and supports a `--validate-only` flag for dry-run compatibility checking.

**Rationale**: Modifying `wave run` would break the existing API contract where the second positional arg is the input string, not a second pipeline. A separate `wave compose` command provides a clean namespace and avoids ambiguity about whether `wave run a b` means "run pipeline a with input b" or "compose pipelines a and b".

### C5: Compose mode pane layout

**Ambiguity**: The spec described a "sequence list" and "artifact flow visualization" appearing in compose mode, but did not explicitly state how they map to the existing two-pane layout (left/right).

**Resolution**: When compose mode is active: (1) the **left pane** shows the sequence list — a navigable list of pipelines in execution order with cursor selection, add/remove/reorder controls; (2) the **right pane** shows the artifact flow visualization — a scrollable view showing per-boundary artifact compatibility. Focus starts on the left pane. The user can press `Enter` on a sequence item to focus the right pane for detailed artifact inspection at that boundary, and `Esc` returns focus to the left pane (a second `Esc` exits compose mode entirely).

**Rationale**: This directly follows the existing left-list/right-detail pattern established by all other views (Pipelines, Personas, Contracts, Skills, Health). Keeping the same layout model reduces cognitive load and implementation complexity.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Users can build a pipeline sequence of 2–5 pipelines in under 30 seconds using the compose mode keyboard controls.
- **SC-002**: Artifact compatibility warnings are shown for 100% of detectable mismatches (missing required inputs) before sequence start is initiated.
- **SC-003**: The `s` key has no effect when pressed outside the valid context (non-available pipeline selected, non-Pipelines view, right pane focused).
- **SC-004**: All compose mode interactions (`a`, `x`, `shift+↑`, `shift+↓`, `Enter`, `Esc`) respond within a single Bubble Tea frame update (no perceptible lag).
- **SC-005**: The artifact flow visualization renders correctly on terminals 80 columns wide and above, with graceful degradation below 120 columns.
- **SC-006**: Running sequences display grouped progress with per-pipeline status visible in the left pane Running section.
- **SC-007**: CLI `wave compose` produces identical sequence validation and execution behavior as the TUI compose mode.
- **SC-008**: All new TUI components follow the existing Bubble Tea patterns (message bus, value semantics, provider interfaces) established in #252–#260.
