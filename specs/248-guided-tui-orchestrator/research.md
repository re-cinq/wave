# Research: Guided TUI Orchestrator

**Feature Branch**: `248-guided-tui-orchestrator`
**Date**: 2026-03-16

## Phase 0 — Research Findings

### R1: GuidedFlowState Machine Integration

**Decision**: Layer GuidedFlowState as a mode field on ContentModel, not replace the existing ViewType system.

**Rationale**: The existing `ContentModel` uses `currentView ViewType` (8 values: ViewPipelines through ViewSuggest) with `cycleView()` doing modulo-8 rotation. Adding a `guidedFlow *GuidedFlowState` field that overrides `Init()` start view and constrains `Tab` behavior is the minimal-invasion approach. When nil, the existing 8-view cycle works unchanged (FR-013).

**Alternatives Rejected**:
- Replacing `ViewType` entirely — would break all existing view switching logic across 1200+ lines of content.go
- Separate `GuidedAppModel` wrapping `AppModel` — over-engineering, would need to duplicate all message routing

### R2: Health Phase → Proposals Auto-Transition

**Decision**: Use a `HealthAllCompleteMsg` message emitted when the last `HealthCheckResultMsg` resolves, triggering a delayed view switch via `tea.Tick`.

**Rationale**: Health checks already run asynchronously via `HealthListModel.Init()` → batch of `RunCheck` cmds. Each check returns `HealthCheckResultMsg`. The `HealthListModel.Update` already processes these. Adding completion tracking (count resolved vs total) and emitting `HealthAllCompleteMsg` when count == total is a 10-line change. The `ContentModel` handles this message by starting a 1-second `tea.Tick` → `HealthTransitionMsg` → switch to `ViewSuggest`.

**Alternatives Rejected**:
- Polling-based: check if all complete on every tick — wasteful, race-prone
- Synchronous barrier: block until all complete — defeats async UX

### R3: Tab Navigation in Guided Mode

**Decision**: When `guidedFlow` is active, `Tab` toggles between `ViewSuggest` and `ViewPipelines` only. Number keys `1`-`8` provide direct-jump to any view.

**Rationale**: The current `cycleView()` does `(currentView + 1) % 8`. In guided mode, replace this with a toggle: if current is `ViewSuggest`, go to `ViewPipelines`; if `ViewPipelines`, go to `ViewSuggest`. `Shift+Tab` reverses. Number key handling is a new `tea.KeyMsg` branch in `ContentModel.Update()`: `"1"` → `ViewPipelines`, `"2"` → `ViewPersonas`, etc.

**Alternatives Rejected**:
- Removing other views in guided mode — breaks power-user access (clarification C3)
- Tab cycles only through Health/Suggest/Pipelines — adds complexity without value after health phase completes

### R4: Suggest View Launch → Fleet View Transition

**Decision**: Reuse existing `SuggestLaunchMsg` / `SuggestComposeMsg` handling which already switches to `ViewPipelines` and launches. No new mechanism needed.

**Rationale**: The current `SuggestLaunchMsg` handler (content.go:869-887) already does exactly what's needed: switches `currentView` to `ViewPipelines`, launches the pipeline via `LaunchRequestMsg`, and emits `ViewChangedMsg`. The `SuggestComposeMsg` handler (content.go:889-939) similarly bridges to compose mode on `ViewPipelines`. This is already wired.

**Alternatives Rejected**: None — the existing mechanism is correct.

### R5: Archive Divider in Fleet View

**Decision**: Modify `PipelineListModel.buildNavigableItems()` to insert a visual divider `navigableItem` between running and finished sections.

**Rationale**: Currently, `buildNavigableItems()` groups by pipeline name (tree view). The archive divider needs a different layout: running runs first (all pipelines), then a divider, then completed/failed runs. This means adding a new `itemKind` (e.g., `itemKindDivider`) and conditionally switching the grouping strategy when in guided mode.

**Alternatives Rejected**:
- Separate list model for archive — would duplicate all rendering/navigation logic
- Rendering-only divider (no navigable item) — scroll offset math breaks without it

### R6: DAG Preview Rendering

**Decision**: Extend `SuggestDetailModel.View()` to render a text-based DAG for sequence/parallel proposals using box-drawing characters.

**Rationale**: The `SuggestDetailModel` already renders proposal details including sequence display (`Sequence: research → implement`). Enhancing this to show a proper DAG with artifact labels requires:
- For sequences: `[pipeline-a] ──artifact──→ [pipeline-b]`
- For parallels: stacked boxes with `│` grouping
- For mixed: staged layout with `Stage 1: ...`, `Stage 2: ...`

This is a view-only change to `suggest_detail.go`.

**Alternatives Rejected**:
- External DAG library — over-engineering for text-based preview
- Canvas/image rendering — not compatible with terminal

### R7: Sequence Grouping in Fleet View

**Decision**: Add a `SequenceGroupID` field to `RunningPipeline` and `FinishedPipeline`, populated from the compose group run ID. Group visually with tree connectors.

**Rationale**: `PipelineLauncher.LaunchSequence()` already generates a `groupRunID` and stores it via `CreateRun("compose:"+names...)`. The pipeline provider can expose this group ID. Rendering grouped runs uses the existing tree connector pattern (`├`, `└`) already used for pipeline-name grouping.

**Alternatives Rejected**:
- Flat list with color coding — harder to visually associate related runs
- Nested sub-lists — would require significant list model changes

### R8: Input Modification Overlay

**Decision**: Add `m` key handler in `SuggestListModel` that emits a `SuggestModifyMsg` with the current proposal. `ContentModel` shows a `textinput.Model` overlay for editing the input field before launching.

**Rationale**: The pipeline launch form in `PipelineDetailModel` (stateConfiguring) already has a `textinput.Model`-based form. The suggest modify overlay can reuse similar patterns. A simpler approach: on `m`, switch to the existing compose mode but pre-populated with just the single proposal's input field editable.

**Alternatives Rejected**:
- Full modal dialog — not available in Bubble Tea without custom implementation
- Inline editing in the list — too cramped, poor UX

### R9: Health Check Failure Prompts

**Decision**: When a health check fails, show a confirmation prompt in the health detail pane: "Continue anyway? (y/n)". If `y`, proceed to proposals. If `n`, stay on health.

**Rationale**: FR-016 requires failed checks to "display a hint message and prompt the user to continue or quit." This is a health-detail enhancement: when `HealthAllCompleteMsg` fires and any check has `HealthCheckErr` status, show the prompt instead of auto-transitioning.

**Alternatives Rejected**:
- Blocking modal — not idiomatic in Bubble Tea
- Auto-continue with warning banner — doesn't give user agency
