# Research: Guided Workflow Orchestrator TUI

## Decision 1: View State Machine Architecture

**Decision**: Introduce a `GuidedMode` flag on `ContentModel` with a `GuidedState` enum (HealthPhase, Proposals, FleetView, Attached) that controls view routing when guided mode is active.

**Rationale**: The existing `cycleView()` iterates through 8 `ViewType` values via Tab. The guided flow needs Tab to toggle between only Proposals and FleetView. Rather than replacing the existing system, a `guidedMode` boolean gates Tab behavior: when true, Tab toggles between `ViewSuggest` (Proposals) and `ViewPipelines` (FleetView); when false, existing 8-view cycling is preserved. This is minimal-diff and preserves backward compatibility.

**Alternatives Rejected**:
- **New ViewType values**: Adding `ViewHealthPhase`, `ViewProposals`, `ViewFleet` would require updating every `switch m.currentView` block (15+ locations). The existing `ViewHealth`, `ViewSuggest`, `ViewPipelines` already map to the guided concepts.
- **Separate AppModel**: A second `GuidedAppModel` would duplicate header/statusbar/content wiring and diverge from the single-model architecture.

## Decision 2: Health Phase Auto-Transition

**Decision**: Add a `HealthPhaseCompleteMsg` emitted when all health checks finish. `ContentModel.Update` handles this message by switching to `ViewSuggest` (Proposals) when in guided mode.

**Rationale**: The existing `HealthListModel` processes individual `HealthCheckResultMsg` messages but has no concept of "all checks done." Adding completion detection to the health list model (counting completed checks vs total) and emitting a batch-complete message is straightforward. The content model then handles the transition.

**Alternatives Rejected**:
- **Timer-based polling**: Polling health check status on a tick would add latency and complexity.
- **Centralized health orchestrator**: Over-engineering for 6 health checks.

## Decision 3: Guided Mode Activation

**Decision**: Guided mode activates in `RunTUI()` when called from the root command (no subcommand). A `GuidedMode bool` field in `LaunchDependencies` signals this. `wave run <pipeline>` never sets this flag.

**Rationale**: The root command already has the `RunE` handler that calls `tui.RunTUI(deps)`. Adding a boolean to `LaunchDependencies` is the simplest way to signal guided mode without changing the CLI interface. FR-014 requires `wave run` to remain unchanged.

## Decision 4: DAG Preview Rendering

**Decision**: Extend `SuggestDetailModel.View()` to render a text-based DAG for sequence/parallel proposals. For sequence proposals, show `A → B → C` with artifact arrows. For parallel proposals, show columns with a convergence point. Reuse rendering patterns from `compose_detail.go`'s `renderArtifactFlow()`.

**Rationale**: `compose_detail.go` already has artifact flow rendering with boundary compatibility visualization. The suggest detail pane needs a simpler version — just showing execution order and type without full compatibility analysis. Building on the same visual language (arrows, columns) keeps the UX consistent.

**Alternatives Rejected**:
- **Full compose validation in suggest detail**: Too heavy — compose validation loads pipeline YAMLs and compares artifacts. The suggest detail only needs visual representation.

## Decision 5: Fleet View Archive Separation

**Decision**: Modify `PipelineListModel.buildNavigableItems()` to insert a visual "Archive" divider between running and finished sections. Add sequence grouping by parsing `compose:` prefixed run names.

**Rationale**: The existing pipeline list already separates running from finished items. The archive divider is a visual enhancement (section header item). Sequence grouping uses the existing `RunningSequence` struct (currently marked TODO #249 in the codebase).

## Decision 6: Input Modification Overlay

**Decision**: Add an `inputOverlay` field to `SuggestListModel` — a `textinput.Model` that activates on `m` key press. When active, it captures the proposal's prefilled input for editing. Enter confirms, Esc cancels.

**Rationale**: The codebase already uses `textinput.Model` for filter inputs (pipeline list, issue list). The same pattern works for input modification. The overlay renders at the bottom of the suggest list view.

## Decision 7: Skip Proposal Behavior

**Decision**: On `s` key press, mark the proposal as skipped (dimmed in the list, excluded from batch launch). Store skip state in a `skipped map[int]bool` on `SuggestListModel`, parallel to the existing `selected map[int]bool`.

**Rationale**: Mirror the existing multi-select toggle pattern. Skipped proposals are visually dimmed but not removed (user can un-skip).

## Decision 8: Zero Proposals Empty State

**Decision**: Extend the existing empty state in `SuggestListModel.View()` (currently "No suggestions available") with a manual launch hint (`n` key to open pipeline chooser) and Tab to switch to fleet view.

**Rationale**: FR-020 requires an informative empty state. Adding keybinding hints to the existing empty state message is minimal change.
