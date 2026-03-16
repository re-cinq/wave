# Feature Specification: Guided Workflow Orchestrator TUI

**Feature Branch**: `248-guided-tui-orchestrator`
**Created**: 2026-03-16
**Status**: Draft
**Input**: [GitHub Issue #248](https://github.com/re-cinq/wave/issues/248) — Evolve `wave` TUI into a guided workflow orchestrator with health → proposals → execution flow

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Health-First Startup (Priority: P1)

A developer types `wave` (no subcommand) and immediately sees system health checks running with live progress indicators. Each check completes asynchronously with a status glyph (success, warning, error). When all checks complete, the view automatically transitions to the proposals screen.

**Why this priority**: The health-first startup is the foundation of the guided flow. Without it, users still land on the generic pipeline list with no guidance. This single change transforms the default entry point from "here are tabs, explore" to "here's what's happening with your project."

**Independent Test**: Can be fully tested by running `wave` and verifying (a) health checks appear as the initial view, (b) each check shows progress then resolves to a status, (c) auto-transition to proposals occurs when all checks complete. Delivers immediate value by surfacing project health without requiring navigation.

**Acceptance Scenarios**:

1. **Given** a project with Wave initialized, **When** the user runs `wave` with no subcommand, **Then** the TUI opens to the health check view (not the pipeline list) showing progress spinners for each check category
2. **Given** health checks are running, **When** all checks complete successfully, **Then** the view automatically transitions to the proposals view within 1 second
3. **Given** a health check fails (e.g., missing adapter binary), **When** the check completes, **Then** the failure is shown with an error glyph, a hint message, and a prompt asking whether to continue
4. **Given** health checks are running, **When** the user presses `Tab`, **Then** the user can skip ahead to the proposals or fleet view without waiting
5. **Given** a missing auto-installable dependency is detected, **When** the health phase completes, **Then** the user is prompted to install it before transitioning to proposals

---

### User Story 2 - Pipeline Proposal Selection (Priority: P1)

After health checks complete, the user sees ranked pipeline proposals generated from the health analysis. Each proposal shows the pipeline type (single, sequence, or parallel), a priority rating, and a rationale. The user can accept, skip, or modify individual proposals using keyboard shortcuts, and can multi-select proposals for batch launching.

**Why this priority**: Proposals are the core value proposition — they answer "what should I do next?" without the user needing to know which pipeline to run. This is the guided part of "guided workflow orchestrator."

**Independent Test**: Can be tested by mocking the suggest provider to return proposals and verifying (a) proposals render with type badges and priority, (b) Enter launches a single proposal, (c) Space toggles multi-select, (d) `m` opens input modification, (e) `s` skips/dismisses a proposal.

**Acceptance Scenarios**:

1. **Given** proposals are available after health analysis, **When** the proposals view renders, **Then** each proposal displays its type badge (`[sequence]`, `[single]`, `[parallel]`), pipeline name(s), priority level, and a rationale description
2. **Given** the user is on the proposals view, **When** they press `Enter` on a single-pipeline proposal, **Then** the pipeline launches with the pre-filled input and the view transitions to the fleet view
3. **Given** the user is on the proposals view, **When** they press `Space` on multiple proposals then press `Enter`, **Then** all selected proposals launch (sequences and parallels respected) and the view transitions to the fleet view
4. **Given** the user is on the proposals view, **When** they press `m` on a proposal, **Then** an input editor overlay appears allowing modification of the prefilled input before launching
5. **Given** the user is on the proposals view, **When** they press `s` on a proposal, **Then** the proposal is dismissed from the list

---

### User Story 3 - DAG Preview (Priority: P2)

Before launching a proposal, the user can see a text-based DAG preview in the detail pane showing execution order and artifact dependencies between pipeline steps. For sequences, arrows show the flow direction and artifact handoff points. For parallel groups, pipelines are shown in a grouped layout.

**Why this priority**: DAG preview provides confidence before committing to execution. Users can understand what will happen without reading pipeline definitions. However, the guided flow works without it — users can still launch proposals blindly.

**Independent Test**: Can be tested by selecting a sequence proposal and verifying the detail pane renders a DAG with pipeline names, directional arrows, and artifact labels between connected steps.

**Acceptance Scenarios**:

1. **Given** a sequence proposal (e.g., research → implement), **When** the user navigates to it in the proposals view, **Then** the detail pane shows a DAG with pipeline names connected by arrows and labeled with artifact names
2. **Given** a parallel proposal (e.g., implement | pr-review), **When** the user navigates to it, **Then** the detail pane shows pipelines in a grouped layout indicating concurrent execution
3. **Given** a multi-select with mixed types, **When** the user has selected both a sequence and a single pipeline, **Then** the detail pane shows a combined execution plan with stage ordering

---

### User Story 4 - Fleet Monitoring with Archive Separation (Priority: P2)

After launching pipelines, the fleet view shows active runs at the top with live step progress, and completed/failed runs below an archive divider. Sequence-linked runs are visually grouped. The user can attach to any running pipeline for a fullscreen live output view and detach back to the fleet.

**Why this priority**: Fleet monitoring makes multi-pipeline execution observable. Without it, users launch pipelines and have no way to track progress across runs. The archive separation prevents completed runs from cluttering the active view.

**Independent Test**: Can be tested by launching 2+ pipelines and verifying (a) active runs appear above the archive divider, (b) completed runs move below the divider, (c) Enter attaches to fullscreen, (d) Esc detaches back to fleet.

**Acceptance Scenarios**:

1. **Given** 2 running and 1 completed pipeline, **When** the fleet view renders, **Then** running pipelines appear above an "Archive" divider and completed pipelines appear below it
2. **Given** a running pipeline is selected, **When** the user presses `Enter`, **Then** the live output view shows fullscreen step-by-step progress with token counts and tool activity
3. **Given** the user is in the attached fullscreen view, **When** they press `Esc`, **Then** they return to the fleet view
4. **Given** a sequence (research → implement), **When** both runs appear in the fleet, **Then** they are visually grouped to indicate they are part of the same sequence

---

### User Story 5 - View Toggle Navigation (Priority: P2)

The user can toggle between the proposals view and fleet view using `Tab` from any non-attached state. The existing views (Personas, Contracts, Skills, Health, Issues, Pull Requests) remain accessible via number keys `1`–`8` as a secondary navigation mode for power users.

**Why this priority**: Seamless navigation between proposals and fleet is essential for the guided flow state machine, but the feature works with just the auto-transitions from P1 stories.

**Independent Test**: Can be tested by pressing `Tab` from proposals (should go to fleet), pressing `Tab` from fleet (should go back to proposals), and verifying all existing views remain accessible via number keys.

**Acceptance Scenarios**:

1. **Given** the user is on the proposals view, **When** they press `Tab`, **Then** the view switches to the fleet (pipelines) view
2. **Given** the user is on the fleet view, **When** they press `Tab`, **Then** the view switches back to the proposals view
3. **Given** the user is in the attached live output view, **When** they press `Tab`, **Then** nothing happens (Tab is blocked during attachment)
4. **Given** the user wants to access the Personas or Contracts views, **When** they press the corresponding number key (e.g., `2` for Personas), **Then** that view activates and `Tab` resumes toggling between the two primary views

---

### User Story 6 - Sequence and Parallel Execution via TUI (Priority: P3)

When the user launches a sequence proposal, the TUI invokes the SequenceExecutor backend. For sequences, pipelines execute in order with artifact handoff between them. For parallel proposals, pipelines execute concurrently. The fleet view reflects the execution state of each component pipeline in real time.

**Why this priority**: The SequenceExecutor backend already exists and works. Wiring it to the TUI launch mechanism is the final integration step. The guided flow works with single-pipeline launches (P1) even without this.

**Independent Test**: Can be tested by launching a sequence proposal and verifying (a) the first pipeline starts, (b) upon completion its artifacts are passed to the next pipeline, (c) the second pipeline starts automatically, (d) both runs are visible in the fleet view.

**Acceptance Scenarios**:

1. **Given** a sequence proposal with 2 pipelines, **When** the user launches it, **Then** the first pipeline starts and the second is queued
2. **Given** the first pipeline in a sequence completes, **When** it produces output artifacts, **Then** those artifacts are injected into the second pipeline which starts automatically
3. **Given** a parallel proposal, **When** the user launches it, **Then** all pipelines in the group start concurrently
4. **Given** a running sequence, **When** the first pipeline fails, **Then** the second pipeline does not start and the failure is shown in the fleet view

---

### User Story 7 - Non-Regression of `wave run` (Priority: P1)

The `wave run <pipeline>` CLI subcommand continues to work exactly as before. The guided flow only activates when `wave` is invoked with no subcommand. All existing pipeline execution, single-run output, and CLI flags remain unchanged.

**Why this priority**: Breaking existing workflows is unacceptable. This is a non-negotiable regression guard.

**Independent Test**: Can be tested by running `wave run <any-pipeline> -- "<input>"` and verifying output, exit codes, and behavior match the pre-change baseline.

**Acceptance Scenarios**:

1. **Given** a user runs `wave run impl-issue -- "some input"`, **When** the pipeline executes, **Then** behavior is identical to before this feature was implemented
2. **Given** a user runs `wave run -v <pipeline>`, **When** output is produced, **Then** verbose output matches the pre-change format
3. **Given** a user runs `wave list runs`, **When** results are displayed, **Then** both TUI-launched and CLI-launched runs appear

---

### Edge Cases

- What happens when health checks take longer than 30 seconds? The view should remain on health with a "still checking..." indicator and allow the user to skip via `Tab`.
- What happens when the suggest provider returns zero proposals? The proposals view should display a "No proposals available" message with instructions to run pipelines manually via `wave run`.
- What happens when all proposals are skipped? The proposals view should show an empty state and the user can navigate to the fleet view.
- What happens when a pipeline launch fails (e.g., adapter binary not found)? The fleet view should show the run with a failed status and an error message, not crash the TUI.
- What happens when the user resizes the terminal during health checks? The layout should reflow correctly for the health check and proposals views, respecting the minimum 80x24 terminal size.
- What happens when multiple sequence proposals share a common pipeline? Each sequence launches independently — no deduplication of pipeline runs.
- What happens when the user presses `q` during health checks? The TUI exits cleanly.
- What happens when `wave` is run in a project without `wave.yaml`? The health check for configuration reports an error and the user can still continue to proposals (which may be empty).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The TUI MUST open to the health check view as the default initial view when `wave` is invoked with no subcommand
- **FR-002**: Health checks MUST run asynchronously on startup, each displaying a progress spinner that resolves to a status glyph (success ✓, warning ▲, error ✗)
- **FR-003**: The TUI MUST auto-transition from health to proposals within 1 second of all health checks completing
- **FR-004**: The user MUST be able to skip the health phase by pressing `Tab` before checks complete
- **FR-005**: Missing auto-installable dependencies MUST prompt the user for installation before transitioning to proposals
- **FR-006**: The proposals view MUST display ranked pipeline recommendations with type badge, priority, and rationale
- **FR-007**: The user MUST be able to launch a single proposal with `Enter`, toggle multi-select with `Space`, modify input with `m`, and skip with `s`
- **FR-008**: Multi-selected proposals MUST launch respecting their types (sequences in order, parallels concurrently)
- **FR-009**: The fleet view MUST separate active runs from completed runs with a visual divider
- **FR-010**: The user MUST be able to attach to a running pipeline (fullscreen live output) with `Enter` and detach with `Esc`
- **FR-011**: In guided mode, `Tab` MUST toggle between proposals (Suggest) and fleet (Pipelines) views. Other views MUST be accessible via number keys `1`–`8`
- **FR-012**: Sequence execution MUST use the existing SequenceExecutor with artifact handoff between pipelines
- **FR-013**: The `wave run <pipeline>` command MUST remain fully backward compatible
- **FR-014**: The DAG preview MUST render in the detail pane when a sequence or parallel proposal is selected
- **FR-015**: Sequence-linked runs MUST be visually grouped in the fleet view
- **FR-016**: Failed health checks MUST display a hint message and prompt the user to continue or quit
- **FR-017**: The TUI MUST handle terminal resize events correctly in all views

### Key Entities

- **HealthPhase**: The startup state representing the period during which system health checks execute. Contains a collection of health check results, tracks completion state, and controls auto-transition timing.
- **Proposal**: A pipeline recommendation generated from health analysis. Has a type (single, sequence, or parallel), a priority level, a rationale, prefilled input, and references to one or more pipeline definitions. Can be selected, modified, skipped, or launched.
- **FleetRun**: A tracked pipeline execution visible in the fleet view. Has a status (running, completed, failed, queued), optional sequence group membership, elapsed time, step progress, and token usage metrics.
- **DAGPreview**: A text-based visualization of pipeline execution order and artifact dependencies. Rendered from a proposal's pipeline sequence or parallel group structure.
- **GuidedFlowState**: The state machine governing view transitions. States: HealthPhase → Proposals ↔ FleetView → Attached (and back). Controls which view is active and what transitions are valid.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Running `wave` with no subcommand shows health checks within 500ms of launch, completing all checks within 10 seconds under normal conditions
- **SC-002**: A user can go from `wave` launch to pipeline execution in 3 interactions or fewer (health auto-completes → select proposal → Enter to launch)
- **SC-003**: All existing `go test -race ./...` tests pass without modification (no regression)
- **SC-004**: `wave run <pipeline>` produces identical behavior and output to the pre-change baseline
- **SC-005**: The fleet view updates running pipeline step progress at least every 2 seconds
- **SC-006**: The TUI handles at least 10 concurrent tracked runs in the fleet view without UI lag or rendering artifacts
- **SC-007**: Terminal resize during any view does not cause crashes or rendering corruption
- **SC-008**: All new TUI components have corresponding unit tests with table-driven edge case coverage

## Clarifications

The following ambiguities were identified and resolved based on codebase analysis.

### C1: "Proposals" view maps to the existing "Suggest" view

**Ambiguity**: The spec introduces a "proposals view" but the codebase already has `ViewSuggest` with `SuggestListModel`, `SuggestDetailModel`, `SuggestProposedPipeline`, and `SuggestDataProvider`. Are these the same thing or separate views?

**Resolution**: The "proposals view" in this spec IS the existing `ViewSuggest` view, enhanced with the guided-flow behaviors described here. No new view type is created. The existing `SuggestListModel` already supports multi-select (`Space`), launch (`Enter`), type badges (`[seq]`/`[par]`), and priority display. The `SuggestDetailModel` already renders execution plans for multi-selected proposals. Implementation should extend these existing models rather than creating parallel components.

**Rationale**: The existing suggest infrastructure (`suggest_list.go`, `suggest_detail.go`, `suggest_messages.go`, `suggest_provider.go`) already implements ~70% of the proposal functionality described in User Story 2. Creating a duplicate view would violate DRY and fragment the codebase.

### C2: "Fleet view" maps to the existing "Pipelines" view

**Ambiguity**: The spec introduces a "fleet view" for monitoring active/completed runs, but `ViewPipelines` already tracks runs with live output attachment (`Enter`/`Esc`), step progress, and a stale detector.

**Resolution**: The "fleet view" in this spec IS the existing `ViewPipelines` view, enhanced with an archive divider separating active runs from completed/failed runs, and visual grouping of sequence-linked runs. The existing `PipelineListModel`, `PipelineDetailModel`, `LiveOutputModel`, and `PipelineLauncher` serve as the foundation. The archive divider and sequence grouping are additive enhancements to the existing pipeline list rendering.

**Rationale**: `ViewPipelines` already supports run listing, live output attachment (`Enter`), detachment (`Esc`), cancellation (`c`), and step progress polling via SQLite events. Building a separate fleet view would duplicate all of this infrastructure.

### C3: Tab navigation uses a two-tier model

**Ambiguity**: Currently `Tab` cycles through all 8 views sequentially (`cycleView` with modulo 8). The spec says Tab toggles between proposals and fleet, with existing tabs as a "secondary mode." How does the secondary mode work?

**Resolution**: In the guided flow (activated when `wave` launches with no subcommand), `Tab` toggles between the two primary views only: Suggest (proposals) and Pipelines (fleet). The remaining 6 views (Personas, Contracts, Skills, Health, Issues, Pull Requests) are accessible via number keys `1`–`8` as a secondary navigation mode, consistent with how many TUI dashboards handle tab navigation. `Shift+Tab` continues to work as reverse toggle within the two primary views. When `wave run` launches the TUI directly into a specific view (e.g., via a future `--view` flag), the full 8-view cycle remains available.

**Rationale**: The guided flow is about reducing cognitive load — the user should only need `Tab` to switch between "what should I do?" (proposals) and "what's running?" (fleet). Number keys provide power-user escape hatch without cluttering the primary flow. This avoids breaking the existing `cycleView` mechanism for non-guided contexts.

### C4: GuidedFlowState is a new layer above the existing view system

**Ambiguity**: Should the `GuidedFlowState` machine replace the existing `ContentModel.currentView` switching, or sit above it?

**Resolution**: `GuidedFlowState` is a new field on `ContentModel` (or `AppModel`) that controls the startup sequence and primary navigation. It does NOT replace `ViewType` or `cycleView`. Instead, it acts as a mode flag:
- When `GuidedFlowState` is active (no-subcommand launch), it overrides `Init()` to start at `ViewHealth`, controls auto-transition to `ViewSuggest` on health completion, and constrains `Tab` to toggle between `ViewSuggest` and `ViewPipelines`.
- When `GuidedFlowState` is nil/inactive (`wave run` or direct TUI invocation with a specific view), the existing 8-view cycle works as before.
- The state machine transitions are: `HealthPhase` → (auto) → `Proposals` ↔ (Tab) ↔ `Fleet` → (Enter) → `Attached` → (Esc) → `Fleet`.

**Rationale**: Layering the guided flow above the existing system preserves backward compatibility (User Story 7 / FR-013) and minimizes structural changes to the well-tested `ContentModel`.

### C5: Health checks feed proposals via the existing doctor → suggest pipeline

**Ambiguity**: The spec says health checks run on startup and feed into proposal generation. But the existing `HealthDataProvider.RunCheck()` returns `HealthCheckResultMsg` (adapter binary found, SQLite OK, etc.), while `suggest.Suggest()` takes a `doctor.Report` (CI status, issue counts, PR metrics). These are different data sources.

**Resolution**: The health checks visible in the TUI (adapter binary, git repo, SQLite, etc.) are **infrastructure health checks** — they verify the system is functional. Proposal generation uses the **codebase health report** from `doctor.Report` (CI status, issue quality, PR staleness). These are two separate health dimensions:
1. **Infrastructure health** (displayed in the health phase): Uses the existing `HealthDataProvider` with its 6 checks. This runs first to ensure the system is operational.
2. **Codebase health** (feeds proposals): Uses the existing `doctor.Scan()` → `suggest.Suggest()` pipeline, already wired through `SuggestDataProvider`. This runs concurrently with or after infrastructure checks.

The auto-transition from health to proposals triggers when all infrastructure checks complete. The proposals view loads its data independently from `SuggestDataProvider`, which internally runs the doctor scan. If the doctor scan takes longer than the infrastructure checks, the proposals view shows a loading state until data arrives.

**Rationale**: The existing `doctor` and `suggest` packages are purpose-built for codebase analysis and proposal generation. The TUI health checks serve a different purpose (system readiness). Conflating them would require either gutting the doctor package or duplicating its logic in the health provider.
