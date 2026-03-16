# Implementation Plan: Guided TUI Orchestrator

**Branch**: `248-guided-tui-orchestrator` | **Date**: 2026-03-16 | **Spec**: `specs/248-guided-tui-orchestrator/spec.md`
**Input**: Feature specification from `/specs/248-guided-tui-orchestrator/spec.md`

## Summary

Evolve the Wave TUI from a static tab-based dashboard into a guided workflow orchestrator with a Health → Proposals → Fleet execution flow. When `wave` is invoked with no subcommand, the TUI starts at the health check view, auto-transitions to pipeline proposals, and provides streamlined Tab navigation between proposals and fleet monitoring. All existing `wave run` behavior remains unchanged.

The implementation layers a `GuidedFlowState` above the existing `ContentModel` view system, reuses the existing `ViewHealth`, `ViewSuggest`, and `ViewPipelines` views with targeted enhancements, and adds DAG preview rendering and archive divider to the fleet view.

## Technical Context

**Language/Version**: Go 1.25+ with Bubble Tea (bubbletea) TUI framework
**Primary Dependencies**: `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`
**Storage**: SQLite for pipeline state (existing), filesystem for pipelines/workspaces (existing)
**Testing**: `go test -race ./...`, table-driven tests with mock providers
**Target Platform**: Linux/macOS terminals, minimum 80x24
**Project Type**: Single Go binary (existing constraint)
**Performance Goals**: Health checks within 500ms of launch, fleet updates every 2s (existing), 10+ concurrent tracked runs
**Constraints**: Single static binary, no new runtime dependencies, backward compatible with `wave run`
**Scale/Scope**: ~14 files modified/created, 3 new files, ~800-1000 LOC added

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new external dependencies. DAG rendering is pure Go string manipulation |
| P2: Manifest as SSOT | PASS | No manifest schema changes. Guided mode is a runtime behavior flag |
| P3: Persona-Scoped Execution | N/A | TUI feature, no persona changes |
| P4: Fresh Memory at Boundaries | N/A | TUI feature, no pipeline execution changes |
| P5: Navigator-First | N/A | TUI feature, no pipeline architecture changes |
| P6: Contracts at Handovers | N/A | TUI feature, no contract changes |
| P7: Relay via Summarizer | N/A | TUI feature, no relay changes |
| P8: Ephemeral Workspaces | N/A | TUI feature, no workspace changes |
| P9: Credentials Never Touch Disk | PASS | No credential handling in new code |
| P10: Observable Progress | PASS | Guided mode enhances observability via structured health → proposals → fleet flow |
| P11: Bounded Recursion | N/A | No meta-pipeline changes |
| P12: Minimal State Machine | PASS | GuidedFlowState sits above pipeline state machine, does not modify the 5-state step lifecycle |
| P13: Test Ownership | PASS | All new components will have table-driven tests. `go test -race ./...` must pass |

**Post-Phase 1 Re-check**: All principles remain satisfied. The guided flow is a UI layer that does not alter pipeline execution, contract validation, workspace isolation, or adapter invocation.

## Project Structure

### Documentation (this feature)

```
specs/248-guided-tui-orchestrator/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 entity design
├── spec.md              # Feature specification
└── checklists/          # Checklist artifacts
```

### Source Code (repository root)

```
internal/tui/
├── guided_flow.go          # NEW: GuidedFlowState type, phase transitions, helpers
├── guided_flow_test.go     # NEW: State machine transition tests
├── guided_messages.go      # NEW: HealthAllCompleteMsg, HealthTransitionMsg, etc.
├── suggest_dag.go          # NEW: DAG preview rendering
├── suggest_dag_test.go     # NEW: DAG rendering tests
├── content.go              # MODIFY: guided flow integration, number-key nav, new message handlers
├── content_test.go         # MODIFY: guided flow test cases
├── app.go                  # MODIFY: accept guided flag, pass to ContentModel
├── app_test.go             # MODIFY: guided mode test cases
├── health_list.go          # MODIFY: completion tracking, emit HealthAllCompleteMsg
├── suggest_list.go         # MODIFY: add m/s key handlers
├── suggest_list_test.go    # MODIFY: new key handler tests
├── suggest_detail.go       # MODIFY: integrate DAG preview
├── pipeline_list.go        # MODIFY: archive divider, sequence grouping
├── pipeline_list_test.go   # MODIFY: divider and grouping tests
├── pipeline_messages.go    # MODIFY: SequenceGroup field on run types
├── statusbar.go            # MODIFY: guided-mode context hints

cmd/wave/
├── main.go                 # MODIFY: pass guided=true when no subcommand
```

**Structure Decision**: All changes are within the existing `internal/tui/` package. Three new files for guided flow state, messages, and DAG rendering. No new packages or directories.

## Implementation Phases

### Phase A: GuidedFlowState Foundation (P1 scope)

**Goal**: Introduce the state machine and modify startup to begin at ViewHealth when guided mode is active.

**Files**:
- `internal/tui/guided_flow.go` — NEW: `GuidedFlowState`, `GuidedFlowPhase`, phase transition methods
- `internal/tui/guided_messages.go` — NEW: `HealthAllCompleteMsg`, `HealthTransitionMsg`, `HealthContinueMsg`
- `internal/tui/content.go` — Add `guidedFlow *GuidedFlowState` field to `ContentModel`. Override `Init()` to start at `ViewHealth` when guided. Handle `HealthAllCompleteMsg` and `HealthTransitionMsg`
- `internal/tui/app.go` — Add `Guided bool` field to `LaunchDependencies`. Pass through to `ContentModel`
- `cmd/wave/main.go` — Set `deps.Guided = true` when no subcommand detected (existing `shouldLaunchTUI` path)

**Key behaviors**:
1. `wave` (no subcommand) → `Init()` sets `currentView = ViewHealth`, lazy-creates health models, starts health checks
2. Health checks complete (no errors) → `HealthAllCompleteMsg` → 1s timer → `HealthTransitionMsg` → switch to `ViewSuggest`
3. Health check errors → prompt in health detail pane → user presses `y` to continue or `q` to quit

### Phase B: Tab Navigation Override (P1 scope)

**Goal**: In guided mode, Tab toggles between ViewSuggest and ViewPipelines. Number keys provide direct-jump.

**Files**:
- `internal/tui/content.go` — Modify Tab handler: when `guidedFlow != nil`, toggle between ViewSuggest and ViewPipelines. Extract `setView(v ViewType) tea.Cmd` from `cycleView()`. Add number key `1`-`8` direct-jump in `Update()`

**Key behaviors**:
1. Tab from Suggest → Pipelines, Tab from Pipelines → Suggest
2. Tab while attached to live output → blocked
3. Shift+Tab reverses the toggle
4. Number keys `1`-`8` jump to any view in both modes

### Phase C: Health Completion Tracking (P1 scope)

**Goal**: Track async health check completion and emit transition signals.

**Files**:
- `internal/tui/health_list.go` — Count resolved checks in `Update()`. When all resolve, return cmd emitting `HealthAllCompleteMsg{HasErrors: ...}`
- `internal/tui/content.go` — Handle `HealthAllCompleteMsg`: no errors → start transition timer; errors → wait for user input

### Phase D: Suggest View Enhancements (P1/P2 scope)

**Goal**: Add input modification (`m`), skip/dismiss (`s`), and DAG preview.

**Files**:
- `internal/tui/suggest_list.go` — Add `s` key (dismiss proposal), `m` key (emit `SuggestModifyMsg`)
- `internal/tui/content.go` — Handle `SuggestModifyMsg`: show input editor overlay
- `internal/tui/suggest_dag.go` — NEW: `RenderDAG()` for text-based DAG visualization
- `internal/tui/suggest_detail.go` — Call `RenderDAG()` for sequence/parallel proposals

### Phase E: Fleet View Enhancements (P2 scope)

**Goal**: Archive divider and sequence grouping in fleet view.

**Files**:
- `internal/tui/pipeline_list.go` — Add `itemKindDivider`, modify `buildNavigableItems()` for archive layout in guided mode
- `internal/tui/pipeline_messages.go` — Add `SequenceGroup string` to run types
- `internal/tui/pipeline_provider.go` — Populate `SequenceGroup` from state store

### Phase F: Status Bar & Polish (P2 scope)

**Goal**: Context-appropriate status bar hints for guided mode.

**Files**:
- `internal/tui/statusbar.go` — Add guided-mode hint branches

### Phase G: Sequence/Parallel Execution Wiring (P3 scope)

**Goal**: Route proposal launch to `LaunchSequence()` for multi-pipeline proposals.

**Files**:
- `internal/tui/content.go` — Enhance `SuggestLaunchMsg` handler: check `Pipeline.Type`, route to `LaunchSequence()` for sequences/parallels

### Phase H: Non-Regression (continuous)

Run `go test -race ./...` after each phase. Verify `wave run` behavior is unchanged. No modifications to `pipeline/executor.go`, `adapter/`, `contract/`, or `workspace/` packages.

## Dependency Order

```
Phase A (foundation) → Phase B (tab nav) → Phase C (health tracking)
                                         ↘
                                          Phase D (suggest enhancements)
                                          Phase E (fleet enhancements)
                                          Phase F (polish)
                                          Phase G (sequence wiring)
Phase H runs continuously
```

Phases D, E, F, G are independent after A+B+C.

## Complexity Tracking

No constitution violations found. No complexity justifications needed.

| Area | Estimated LOC | Risk | Mitigation |
|------|---------------|------|------------|
| GuidedFlowState | ~80 | Low | Simple state machine with 4 phases |
| Tab override | ~40 | Low | Conditional branch in existing handler |
| Health completion | ~30 | Low | Counter in existing Update handler |
| DAG preview | ~100 | Medium | Text layout edge cases — comprehensive test suite |
| Archive divider | ~120 | Medium | Dual layout mode — careful cursor management |
| Sequence grouping | ~80 | Medium | State store query for group ID — needs provider change |
| Input modification | ~60 | Low | Reuse existing textinput patterns |
| Status bar updates | ~30 | Low | Additional conditional branches |

**Total estimated**: ~540 LOC new code + ~200 LOC test code
