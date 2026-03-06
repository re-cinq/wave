# Implementation Plan: TUI Pipeline Launch Flow

**Branch**: `256-tui-pipeline-launch` | **Date**: 2026-03-06 | **Spec**: `specs/256-tui-pipeline-launch/spec.md`
**Input**: Feature specification from `/specs/256-tui-pipeline-launch/spec.md`

## Summary

Add pipeline launch capability to the TUI. When a user selects an available pipeline and presses Enter, the right pane transitions from detail preview to an embedded `huh.Form` argument configuration form (input, model override, flag toggles). Submitting launches the pipeline executor in a background goroutine via `tea.Cmd`. The pipeline immediately appears in the Running section. Esc cancels at any point, `c` cancels a running pipeline. A `PipelineLauncher` component on `ContentModel` manages executor lifecycle and cancel functions. `LaunchDependencies` struct flows through the component hierarchy to provide executor construction dependencies on demand.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/huh` v0.8.0 (both existing)
**Storage**: SQLite via `internal/state` (existing — used for run record creation and status updates)
**Testing**: `go test` with `testify/assert`, `testify/require`
**Target Platform**: Linux/macOS terminal (80–300 columns, 24–100 rows)
**Project Type**: Single Go binary — changes in `internal/tui/` and `cmd/wave/`
**Constraints**: No new external dependencies; must not break existing tests (`go test ./...`)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. Uses existing `huh` v0.8.0 form embedding. |
| P2: Manifest as SSOT | ✅ Pass | Pipeline loaded from YAML files discovered via manifest pipelines directory. |
| P3: Persona-Scoped Execution | ✅ Pass | TUI constructs executor with full persona scoping — same path as CLI. |
| P4: Fresh Memory at Step Boundary | ✅ Pass | Executor constructed per-launch — no state leakage between launches. |
| P5: Navigator-First Architecture | ✅ Pass | Executor runs the full pipeline DAG including navigator steps. |
| P6: Contracts at Every Handover | ✅ Pass | Executor handles contract validation — TUI doesn't bypass it. |
| P7: Relay via Dedicated Summarizer | N/A | TUI component, no context compaction. |
| P8: Ephemeral Workspaces | ✅ Pass | Executor creates workspaces as normal — TUI doesn't interfere. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling in TUI layer. Executor inherits env vars. |
| P10: Observable Progress | ✅ Pass | Executor emitter is initialized per-launch. Running section shows status. |
| P11: Bounded Recursion | ✅ Pass | Executor enforces bounds — TUI doesn't modify executor behavior. |
| P12: Minimal Step State Machine | ✅ Pass | Uses existing step state machine via executor. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests updated for modified signatures. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/256-tui-pipeline-launch/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/tui/
├── app.go                    # MODIFY — q-quit guard, CancelAll on exit, LaunchDependencies param
├── app_test.go               # MODIFY — update for new AppModel signature, q-quit focus tests
├── content.go                # MODIFY — add PipelineLauncher, route launch messages, CancelAll()
├── content_test.go           # MODIFY — update for new ContentModel signature, launch message tests
├── pipeline_detail.go        # MODIFY — add DetailPaneState, form embedding, form lifecycle
├── pipeline_detail_test.go   # MODIFY — add form rendering, state transition, form abort tests
├── pipeline_list.go          # MODIFY — handle PipelineLaunchedMsg (insert running entry)
├── pipeline_list_test.go     # MODIFY — test PipelineLaunchedMsg handling
├── pipeline_messages.go      # MODIFY — add LaunchRequestMsg, PipelineLaunchedMsg, etc.
├── pipeline_launcher.go      # NEW — PipelineLauncher component
├── pipeline_launcher_test.go # NEW — launcher unit tests
├── statusbar.go              # MODIFY — add form-context hints
├── statusbar_test.go         # MODIFY — test form-context hints
├── run_selector.go           # READ-ONLY — reuse WaveTheme(), DefaultFlags(), buildFlagOptions()
├── pipelines.go              # MODIFY — add LoadPipelineByName() for full pipeline loading
├── pipelines_test.go         # MODIFY — test LoadPipelineByName()
├── theme.go                  # READ-ONLY — WaveTheme() reused by embedded form
├── header_messages.go        # READ-ONLY — FocusPane, FocusChangedMsg reused

cmd/wave/commands/
├── tui.go                    # MODIFY — pass LaunchDependencies to RunTUI()
```

**Structure Decision**: All changes within the existing `internal/tui/` package plus one new file (`pipeline_launcher.go`). The launcher is a component, not a separate package, because it operates within the TUI's message-passing architecture and shares types with other TUI components.

## Implementation Approach

### Phase 1: Foundation — Messages, Types, and State Machine

**Files**: `pipeline_messages.go`, `pipeline_detail.go`

1. Add new message types to `pipeline_messages.go`:
   - `LaunchRequestMsg{Config LaunchConfig}`
   - `PipelineLaunchedMsg{RunID, PipelineName string}`
   - `PipelineLaunchResultMsg{RunID string, Err error}`
   - `LaunchErrorMsg{PipelineName string, Err error}`
   - `LaunchConfig` struct
   - `LaunchDependencies` struct
   - `DetailPaneState` type with constants

2. Refactor `PipelineDetailModel` to use `DetailPaneState`:
   - Add `paneState DetailPaneState` field
   - Set state in `Update()` instead of relying on nil checks
   - Update `View()` to switch on `paneState`
   - This is a refactor — existing behavior must be preserved

### Phase 2: Argument Form — Creation and Lifecycle

**Files**: `pipeline_detail.go`, `pipeline_detail_test.go`

1. Add form fields to `PipelineDetailModel`:
   - `launchForm *huh.Form`
   - `launchInput, launchModel string` (value bindings)
   - `launchFlags []string` (value binding)
   - `launchError string`

2. Implement form creation when `stateConfiguring` is entered:
   - Triggered by a new `ConfigureFormMsg` from `ContentModel`
   - Create `huh.Form` with input field (placeholder from `InputExample`), model override field, flag multi-select
   - Apply `WaveTheme()`, set width/height
   - Call `form.Init()`

3. Implement form lifecycle in `Update()`:
   - Forward key messages to `form.Update(msg)` when `stateConfiguring`
   - Check `form.State` after each update:
     - `huh.StateCompleted` → extract values, emit `LaunchRequestMsg`
     - `huh.StateAborted` → revert to `stateAvailableDetail`
   - Handle form resizing on `SetSize()`

4. Implement form rendering in `View()`:
   - `stateConfiguring` renders `form.View()` within the pane
   - `stateLaunching` renders "Starting..." indicator
   - `stateError` renders error message with action hints

### Phase 3: PipelineLauncher Component

**Files**: `pipeline_launcher.go`, `pipeline_launcher_test.go`

1. Implement `PipelineLauncher` struct:
   - `deps LaunchDependencies`
   - `cancelFns map[string]context.CancelFunc`
   - `mu sync.Mutex`

2. Implement `Launch(config LaunchConfig) tea.Cmd`:
   - Load full `pipeline.Pipeline` from YAML
   - Create `context.WithCancel(context.Background())`
   - Resolve adapter (mock if `--mock` flag, else `adapter.ResolveAdapter()`)
   - Create run ID via state store
   - Build executor with options (mirroring `runRun()`)
   - Store cancel function in map
   - Return `tea.Batch(immediateCmd, executorCmd)`:
     - `immediateCmd` returns `PipelineLaunchedMsg`
     - `executorCmd` calls `executor.Execute()`, returns `PipelineLaunchResultMsg`

3. Implement `Cancel(runID string)`:
   - Look up cancel function in map, call it if found

4. Implement `CancelAll()`:
   - Iterate map, call all cancel functions

5. Add `LoadPipelineByName(name string) (*pipeline.Pipeline, error)` to `pipelines.go`

### Phase 4: Content Model Integration

**Files**: `content.go`, `content_test.go`

1. Add `launcher *PipelineLauncher` field to `ContentModel`
2. Modify `NewContentModel()` to accept `LaunchDependencies` and create launcher
3. Add message routing in `Update()`:
   - `LaunchRequestMsg` → call `launcher.Launch(config)`, transition to `stateLaunching`
   - `PipelineLaunchedMsg` → forward to list (insert running entry), transition focus to left
   - `PipelineLaunchResultMsg` → forward to launcher (cleanup cancel map)
   - `LaunchErrorMsg` → forward to detail (show error), focus left
4. Modify Enter handling: when focus transitions right on available item, emit `ConfigureFormMsg` to trigger form creation
5. Expose `CancelAll()` method that delegates to launcher
6. Handle `c` key when left pane focused and cursor on running item: call `launcher.Cancel(runID)`

### Phase 5: Pipeline List — Running Entry Insertion

**Files**: `pipeline_list.go`, `pipeline_list_test.go`

1. Handle `PipelineLaunchedMsg` in `PipelineListModel.Update()`:
   - Create synthetic `RunningPipeline{RunID, Name, StartedAt: time.Now()}`
   - Prepend to `m.running`
   - Rebuild navigable items
   - Move cursor to the new running entry
   - Return `RunningCountMsg` and `PipelineSelectedMsg`

### Phase 6: App Model and Status Bar

**Files**: `app.go`, `app_test.go`, `statusbar.go`, `statusbar_test.go`

1. Modify `AppModel`:
   - Gate `q`-to-quit on `m.content.focus == FocusPaneLeft`
   - Call `m.content.CancelAll()` before `tea.Quit` on exit (both `q` and `Ctrl+C`)
   - Update `NewAppModel()` to accept `LaunchDependencies`

2. Modify `RunTUI()`:
   - Accept `LaunchDependencies` parameter
   - Pass through to `NewAppModel()`

3. Modify `StatusBarModel`:
   - Detect configuring state (new `FormActiveMsg` or check focus + pane state)
   - Show form-specific hints: `"Tab: next  Shift+Tab: prev  Enter: launch  Esc: cancel"`

### Phase 7: CLI Integration

**Files**: `cmd/wave/commands/tui.go` (or equivalent)

1. Modify the TUI command to construct `LaunchDependencies`:
   - Load manifest
   - Open state store
   - Determine pipelines directory
   - Pass to `tui.RunTUI(deps)`

### Phase 8: Test Updates

**Files**: All `*_test.go` files in `internal/tui/`

1. Update all test functions that call `NewAppModel()`, `NewContentModel()` to pass `nil` or zero-value `LaunchDependencies`
2. Add new test cases:
   - Form creation on Enter for available item
   - Form abort on Esc
   - Form completion triggers LaunchRequestMsg
   - PipelineLaunchedMsg inserts running entry
   - `q` during form does not quit
   - `c` on running pipeline calls cancel
   - CancelAll on exit
   - Form renders correctly at boundary dimensions

## Complexity Tracking

_No constitution violations. No complexity tracking entries needed._
