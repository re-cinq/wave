# Feature Specification: Guided Workflow Orchestrator TUI

**Feature Branch**: `248-guided-workflow-tui`
**Created**: 2026-03-16
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/248"

## Clarifications

### C1: Navigation Model — Guided Flow vs Peer Tabs

**Ambiguity**: The spec says Tab toggles between Proposals and FleetView, but the existing TUI uses Tab to cycle through 8 peer views (Pipelines, Personas, Contracts, Skills, Health, Issues, PRs, Suggest). How do these coexist?

**Resolution**: The guided flow introduces a **mode-based navigation** that coexists with the existing peer-tab model. When `wave` starts with no subcommand, the TUI enters **guided mode** where Tab toggles between Proposals and FleetView (the two primary guided views). The existing peer-tab cycling (Personas, Contracts, Skills, etc.) is accessible via a secondary key binding (e.g., `[` and `]` or a dedicated key like `v` for "views menu"). Health is shown as the startup phase, not a persistent tab. This avoids breaking the existing tab-cycling experience while keeping the guided flow focused.

**Rationale**: The issue mockups show a focused 3-state flow (Health → Proposals ↔ Fleet). Mixing 8+ tabs with Tab would dilute the guided experience. A secondary view-switcher key preserves access to all views without cluttering the primary flow.

### C2: Health Checks vs Codebase Analysis — Two Distinct Data Sources

**Ambiguity**: The spec says proposals are "based on the health analysis" but the codebase has two distinct systems: (1) `HealthDataProvider` which checks infrastructure (git, adapter, database, config, tools, skills) and (2) `SuggestDataProvider` which wraps `doctor.Report` for codebase analysis (open issues, PRs, CI status). These produce different data.

**Resolution**: The Health Phase displays **infrastructure checks** from `HealthDataProvider` (existing 6 categories). Proposals are sourced from `SuggestDataProvider` (wrapping `suggest.Suggest()` which uses `doctor.Report`). The health summary line at the top of the proposals view (FR-003) aggregates data from the `doctor.Report` (open issues, PRs, CI status), NOT from infrastructure health checks. Infrastructure health is a gate (must pass before proposals), while codebase health drives proposal ranking.

**Rationale**: The existing `HealthDataProvider` checks infra readiness, while `suggest.Engine` generates proposals from `doctor.Report`. These are complementary — infra checks ensure Wave can run, codebase analysis determines what to run. Conflating them would require restructuring both systems unnecessarily.

### C3: "Proposals" View Identity — Extends Existing Suggest View

**Ambiguity**: The spec introduces "Proposals" as a new view but the codebase already has `ViewSuggest` with `SuggestListModel`, `SuggestDetailModel`, `SuggestDataProvider`, and full multi-select + launch support. Are these the same thing?

**Resolution**: The "Proposals" view in this spec IS the existing `ViewSuggest` view, enhanced with: (1) a health summary header line, (2) DAG preview rendering in the detail pane, (3) skip (`s`) and modify (`m`) keybindings, and (4) integration into the guided flow state machine. The existing `SuggestListModel`/`SuggestDetailModel` are extended in-place — no new view type is created. `ViewSuggest` may be renamed to `ViewProposals` for clarity but this is a cosmetic rename, not a new component.

**Rationale**: The existing Suggest view already implements multi-select (Space), single launch (Enter), cursor navigation (j/k), and compose orchestration (`SuggestComposeMsg`). Building on it avoids duplication and leverages tested code.

### C4: Sequence Execution — Uses PipelineLauncher.LaunchSequence, Not SequenceExecutor

**Ambiguity**: FR-017 references "the existing SequenceExecutor" but no type named `SequenceExecutor` exists in the codebase. Sequence/parallel execution is handled by `PipelineLauncher.LaunchSequence()` which spawns `wave compose` subprocesses.

**Resolution**: All references to "SequenceExecutor" in this spec mean `PipelineLauncher.LaunchSequence()` (in `internal/tui/pipeline_launcher.go`). For sequence proposals (Type="sequence"), `LaunchSequence(names, input, false, nil)` is called. For parallel proposals (Type="parallel"), `LaunchSequence(names, input, true, stages)` is called. The `wave compose` subprocess handles actual orchestration.

**Rationale**: `PipelineLauncher.LaunchSequence` is the existing, tested mechanism for multi-pipeline TUI launches. It generates a group run ID, creates a detached subprocess, and records PID for monitoring — exactly what the fleet view needs.

### C5: Auto-Install Prompt Scope

**Ambiguity**: FR-015 requires "auto-install prompts for missing but auto-installable dependencies" but the existing `HealthDataProvider` only checks presence — it has no concept of installability. The `preflight` package has auto-install for skills (via `requires.skills[].install` config) but not for tools.

**Resolution**: Auto-install prompts during the Health Phase are limited to **skills** that have an `install` command configured in the pipeline's `requires.skills` section (matching the existing `preflight.CheckSkills` behavior). Tools checked by `HealthDataProvider.checkRequiredTools()` only report presence/absence — they show remediation hints (e.g., "Install with: brew install gh") but do not attempt auto-install. This matches the existing preflight contract where only skills with explicit install commands are auto-installable.

**Rationale**: The `preflight` package already distinguishes installable skills (have `install` command) from tools (binary lookup only). Extending auto-install to arbitrary tools would require a new package manager abstraction that is out of scope.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Health-First Startup (Priority: P1)

A developer runs `wave` (no subcommand) and immediately sees health checks running with live progress spinners. Each check category (git, adapter, database, config, tools, skills) displays its status as it completes. When all checks finish, the view automatically transitions to the proposals screen.

**Why this priority**: The health-first startup is the foundational UX change — it replaces the current "peer tabs" model with a guided entry point. Without this, no other guided flow features work.

**Independent Test**: Run `wave` and verify health checks display on startup with live progress, and auto-transition to proposals view on completion.

**Acceptance Scenarios**:

1. **Given** the user runs `wave` with no subcommand, **When** the TUI opens, **Then** health checks begin immediately and are displayed as the primary view with animated spinners for in-progress checks.
2. **Given** health checks are running, **When** a check completes, **Then** its status updates in-place (spinner → checkmark/warning/error) without clearing the screen.
3. **Given** all health checks have completed successfully, **When** the last check finishes, **Then** the view auto-transitions to the proposals screen within 500ms.
4. **Given** a health check fails with a critical dependency (e.g., adapter binary not found), **When** the check completes, **Then** the failure is displayed with a hint message and the user is prompted whether to continue.
5. **Given** a missing dependency is auto-installable, **When** health checks complete, **Then** the user is prompted with an install confirmation before transitioning to proposals.
6. **Given** the user presses `Tab` during health checks, **When** checks are still running, **Then** the user can skip to the fleet view immediately.

---

### User Story 2 - Pipeline Proposal Selection (Priority: P1)

After health checks complete, the developer sees a ranked list of pipeline proposals based on the health analysis. Each proposal shows its type (single, sequence, parallel), priority, and rationale. The developer can navigate proposals, view a DAG preview of execution order and artifact dependencies, and launch selected proposals.

**Why this priority**: Proposals are the core value of the guided orchestrator — turning health analysis into actionable pipeline recommendations. This is the "what should I do?" answer.

**Independent Test**: Navigate to proposals view (via health transition or direct Tab), verify proposals display with type/priority/rationale, and that Enter launches a proposal.

**Acceptance Scenarios**:

1. **Given** health checks have completed, **When** the proposals view loads, **Then** a health summary line appears at the top (e.g., "12 open issues, 3 PRs awaiting review, all deps OK").
2. **Given** proposals are displayed, **When** the user navigates with j/k or arrow keys, **Then** the cursor moves between proposals and the detail pane updates with a DAG preview for the focused proposal.
3. **Given** a proposal of type "sequence" is focused, **When** the DAG preview renders, **Then** it shows the pipeline execution order with artifact dependency arrows between stages.
4. **Given** a proposal of type "parallel" is focused, **When** the DAG preview renders, **Then** it shows pipelines grouped in parallel with a convergence point if applicable.
5. **Given** a single proposal is focused, **When** the user presses Enter, **Then** the proposal launches and the view transitions to the fleet view.
6. **Given** the user presses `s` on a focused proposal, **When** the skip action fires, **Then** the proposal is dimmed/removed from the active list without launching.
7. **Given** the user presses `m` on a focused proposal, **When** the modify overlay appears, **Then** the user can edit the prefilled input text and confirm or cancel the modification.

---

### User Story 3 - Multi-Select and Batch Launch (Priority: P2)

The developer can toggle multiple proposals for batch execution using Space, see a combined execution plan, and launch them all with a single Enter press.

**Why this priority**: Multi-select builds on the single-proposal launch (P1) and enables the common workflow of launching research + review concurrently. It is not required for MVP but significantly improves throughput.

**Independent Test**: Select 2+ proposals with Space, verify the execution plan updates in the detail pane, press Enter and verify all selected pipelines launch.

**Acceptance Scenarios**:

1. **Given** proposals are displayed, **When** the user presses Space on a proposal, **Then** the proposal is toggled as selected (visual indicator changes, e.g., filled circle).
2. **Given** 2+ proposals are selected, **When** the header updates, **Then** it shows the count of selected proposals (e.g., "3 proposals — 2 selected").
3. **Given** 2+ proposals are selected, **When** the detail pane renders, **Then** it shows a combined execution plan listing all selected pipelines with their execution mode (sequence vs parallel).
4. **Given** 2+ proposals are selected, **When** the user presses Enter, **Then** all selected proposals launch (sequences via SequenceExecutor, parallel groups concurrently) and the view transitions to fleet.

---

### User Story 4 - Fleet View with Archive Separation (Priority: P2)

While pipelines are running, the developer monitors all active runs in the fleet view. Active/running pipelines appear above an archive divider. Completed and failed runs appear below the divider. Sequence-linked runs are grouped visually.

**Why this priority**: The fleet view already exists as the Pipelines tab. Enhancing it with archive separation and sequence grouping makes the monitoring experience match the guided workflow — showing what matters now vs. what's done.

**Independent Test**: Launch 2+ pipelines, verify they appear in the active section. Wait for completion, verify they move below the archive divider.

**Acceptance Scenarios**:

1. **Given** pipelines are running, **When** the fleet view renders, **Then** running pipelines appear above a visual "Archive" divider and completed/failed runs appear below it.
2. **Given** a sequence was launched (e.g., research → implement), **When** the fleet view renders, **Then** sequence-linked runs are grouped together with the queued pipeline shown as pending (◌).
3. **Given** a pipeline in a sequence completes, **When** the next pipeline in the sequence auto-starts, **Then** the fleet view updates to show the completed pipeline with ✓ and the next pipeline as running (●).
4. **Given** the user presses Enter on a running pipeline, **When** the live output view loads, **Then** the user sees fullscreen step-by-step progress for that single pipeline.
5. **Given** the user is in the attached (live output) view, **When** the user presses Esc, **Then** the view returns to the fleet view.
6. **Given** the user is in the fleet view, **When** the user presses `p` or Tab, **Then** the view toggles to the proposals view.

---

### User Story 5 - View State Machine Transitions (Priority: P2)

The TUI follows a predictable state machine: HealthPhase → Proposals → FleetView → Attached (and back). Tab toggles between Proposals and FleetView. The user always knows which view they are in and how to navigate.

**Why this priority**: Without clear state transitions, the guided flow feels disjointed. This story ensures the navigation model is consistent and discoverable.

**Independent Test**: Walk through the full flow (health → proposals → launch → fleet → attach → detach → proposals → fleet) and verify each transition works as expected.

**Acceptance Scenarios**:

1. **Given** the TUI is in HealthPhase, **When** all checks complete, **Then** the view transitions to Proposals.
2. **Given** the TUI is in Proposals, **When** the user presses Enter to launch, **Then** the view transitions to FleetView.
3. **Given** the TUI is in FleetView, **When** the user presses Tab or `p`, **Then** the view transitions to Proposals.
4. **Given** the TUI is in Proposals, **When** the user presses Tab, **Then** the view transitions to FleetView.
5. **Given** the TUI is in FleetView, **When** the user presses Enter on a running pipeline, **Then** the view transitions to Attached (fullscreen live output).
6. **Given** the TUI is in Attached view, **When** the user presses Esc, **Then** the view transitions back to FleetView.
7. **Given** the TUI is in any non-Attached view, **When** the user presses `q`, **Then** the TUI exits (with Ctrl+C double-press for cancel-first behavior preserved).

---

### User Story 6 - Backward Compatibility with `wave run` (Priority: P1)

Running `wave run <pipeline>` continues to work exactly as before — single-pipeline execution with no guided flow. The guided orchestrator only activates when `wave` is run with no subcommand.

**Why this priority**: Breaking `wave run` would be a critical regression. This is a non-negotiable compatibility requirement.

**Independent Test**: Run `wave run <pipeline> -- "input"` and verify it executes the pipeline directly without showing health checks or proposals.

**Acceptance Scenarios**:

1. **Given** the user runs `wave run impl-issue -- "fix bug"`, **When** the command executes, **Then** the pipeline runs directly without any guided flow screens.
2. **Given** the user runs `wave` with no subcommand, **When** the TUI opens, **Then** the guided flow starts with health checks (not the old pipeline list).

---

### Edge Cases

- What happens when there are zero proposals after health analysis? → The proposals view displays an empty state message (e.g., "No pipeline recommendations") with an option to launch manually via `n` key or switch to fleet view via Tab.
- What happens when health checks hang or take longer than 30 seconds? → A timeout mechanism displays partial results with the hung check showing a warning, and the user can press Tab to skip to proposals/fleet.
- What happens when the user launches a sequence but a mid-sequence pipeline fails? → The fleet view shows the failed pipeline with ✗, queued downstream pipelines in the sequence are cancelled, and the user can see error details by attaching.
- What happens when the terminal is resized during health check animation? → The Bubble Tea WindowSizeMsg propagates to all child components; health check display re-renders at the new size.
- What happens when there is no GitHub token or repository context? → Issue-based and PR-based proposals are omitted; only tool/dependency/configuration proposals appear.
- What happens when the user presses Ctrl+C during health checks? → First press cancels running checks and shows partial results; second press exits the TUI.
- What happens when multiple proposals reference the same pipeline? → Each proposal is independent; the same pipeline can appear in multiple proposals with different inputs.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The TUI MUST start in a health check phase when `wave` is run with no subcommand, displaying live progress for each check category.
- **FR-002**: The TUI MUST auto-transition from the health phase to the proposals view when all health checks complete.
- **FR-003**: The proposals view MUST display ranked pipeline recommendations with type (single/sequence/parallel), priority, and rationale.
- **FR-004**: The proposals view MUST support single-selection launch via Enter on a focused proposal.
- **FR-005**: The proposals view MUST support multi-selection via Space toggle with a combined execution plan display.
- **FR-006**: The proposals view MUST support input modification via `m` key with an editable text field overlay.
- **FR-007**: The proposals view MUST support skipping proposals via `s` key.
- **FR-008**: The proposals view MUST display a DAG preview showing execution order and artifact dependencies for focused proposals.
- **FR-009**: The fleet view MUST separate active/running pipelines from completed/failed pipelines with a visual archive divider.
- **FR-010**: The fleet view MUST visually group sequence-linked pipeline runs.
- **FR-011**: The fleet view MUST allow attaching to a running pipeline via Enter for fullscreen live output.
- **FR-012**: The TUI MUST support toggling between Proposals and FleetView via Tab key.
- **FR-013**: The TUI MUST support detaching from live output via Esc key to return to fleet view.
- **FR-014**: The `wave run <pipeline>` command MUST continue to work unchanged (no guided flow, no regression).
- **FR-015**: The health phase MUST display auto-install prompts for missing but auto-installable dependencies.
- **FR-016**: The health phase MUST display failure hints for critical missing dependencies with actionable remediation.
- **FR-017**: Sequence launches MUST use `PipelineLauncher.LaunchSequence()` (spawning `wave compose` subprocesses) with cross-pipeline artifact handoff.
- **FR-018**: Parallel proposal launches MUST execute concurrently using the existing ExecutePlan with parallel stages.
- **FR-019**: The fleet view MUST show real-time step progress for running pipelines (step states, tool activity, token counts).
- **FR-020**: The TUI MUST handle zero proposals gracefully with an informative empty state message and manual launch option.

### Key Entities

- **HealthPhase**: The startup view state displaying live health check progress. Wraps existing HealthDataProvider output with phase-aware rendering and auto-transition logic.
- **Proposal**: A ranked pipeline recommendation with type (single/sequence/parallel), priority, rationale, and prefilled input. Sourced from the existing SuggestDataProvider.
- **DAG Preview**: A text-based rendering of pipeline execution order and artifact dependencies for a selected proposal or set of proposals. Builds on existing compose_detail.go artifact flow rendering.
- **Fleet View**: Enhanced version of the existing Pipelines view with archive separation (active above divider, completed below) and sequence grouping.
- **Attached View**: Fullscreen live output for a single running pipeline. Uses the existing LiveOutputModel.
- **View State Machine**: The guided navigation model — HealthPhase → Proposals ↔ FleetView → Attached → FleetView — with consistent keybindings at each state.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Running `wave` with no subcommand displays health checks within 200ms of launch (first frame rendered).
- **SC-002**: Health-to-proposals auto-transition completes within 500ms of the last health check finishing.
- **SC-003**: All existing `go test -race ./...` tests pass without modification (no regression).
- **SC-004**: The view state machine supports the complete cycle (Health → Proposals → Fleet → Attached → Fleet → Proposals) with no dead-end states.
- **SC-005**: Multi-select launch of 3+ proposals results in all pipelines starting within 2 seconds of pressing Enter.
- **SC-006**: The fleet view correctly separates running pipelines from completed/failed pipelines in all observed states.
- **SC-007**: `wave run <pipeline>` behavior is identical to the current implementation (no guided flow injected).
- **SC-008**: DAG preview renders correctly for single, sequence, and parallel proposal types with artifact dependency arrows.
- **SC-009**: The TUI gracefully handles terminal sizes down to 80x24 for all views in the guided flow.
