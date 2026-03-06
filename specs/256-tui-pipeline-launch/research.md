# Research: TUI Pipeline Launch Flow

**Date**: 2026-03-06
**Feature**: #256 — TUI Pipeline Launch Flow

## R1: huh.Form Embedding in Bubble Tea

**Decision**: Use `huh.Form` as an embedded `tea.Model` within `PipelineDetailModel`

**Rationale**: The `huh` library (v0.8.0, already a dependency) implements `tea.Model` on its `Form` type — `Init()`, `Update(msg) (tea.Model, tea.Cmd)`, `View() string`. When embedded, the parent model forwards key messages to `form.Update(msg)` and checks `form.State` after each update for `huh.StateCompleted` or `huh.StateAborted`. This is the documented embedding pattern.

Key API surface:
- `huh.NewForm(groups...) *Form` — constructor
- `form.Init() tea.Cmd` — must be called when the form is created
- `form.Update(msg) (tea.Model, tea.Cmd)` — returns `tea.Model`, must cast back to `*huh.Form`
- `form.State` — `huh.StateNormal`, `huh.StateCompleted`, `huh.StateAborted`
- `form.WithWidth(w)` / `form.WithHeight(h)` — sizing for embedded layout
- `form.WithTheme(theme)` — applies `WaveTheme()` for visual consistency
- Value bindings via `huh.NewInput().Value(&str)`, `huh.NewMultiSelect[string]().Value(&slice)`

**Alternatives Rejected**:
- Custom Bubble Tea widgets: Higher implementation cost, no visual consistency with CLI selector
- Full-screen form via `form.Run()`: Blocks the event loop, incompatible with TUI architecture

## R2: Executor Integration via Background Goroutine

**Decision**: Launch executor in a goroutine wrapped in `tea.Cmd`, return result messages

**Rationale**: The executor's `Execute()` method is synchronous and blocking. The Bubble Tea event loop must not block. The standard pattern is to wrap blocking work in a `tea.Cmd` (a `func() tea.Msg`). The approach:

1. `PipelineLauncher.Launch(config)` returns a `tea.Cmd` via `tea.Batch()` combining:
   - An immediate cmd that returns `PipelineLaunchedMsg` (instant UI feedback)
   - A blocking cmd that runs the executor and returns `PipelineLaunchResultMsg`
2. The cancel function from `context.WithCancel()` is stored in a map keyed by run ID
3. `tea.Batch()` runs both commands concurrently — the immediate one resolves instantly, the executor one blocks until completion

**Alternatives Rejected**:
- Channel-based communication: Unnecessary complexity; `tea.Cmd` is idiomatic Bubble Tea
- Polling state store: 5-second delay, poor UX for instant feedback

## R3: LaunchDependencies Injection

**Decision**: Introduce `LaunchDependencies` struct passed through `NewAppModel()` → `NewContentModel()` → `PipelineLauncher`

**Rationale**: The executor requires `*manifest.Manifest`, `state.StateStore`, and the pipelines directory path. These are available at `RunTUI()` call site. Rather than constructing full executor infrastructure at startup, pass lightweight dependencies and construct executor on demand.

`PipelineLauncher` resolves adapter runner, workspace manager, and audit logger per-launch, mirroring `runRun()` in `cmd/wave/commands/run.go`. This keeps TUI startup fast and avoids holding resources for unlaunched pipelines.

**Alternatives Rejected**:
- Pre-constructing executor at startup: Wastes resources, couples adapter resolution to TUI init
- Passing individual dependencies: Too many parameters, struct is cleaner

## R4: Detail Pane State Machine Extension

**Decision**: Add explicit `DetailPaneState` type replacing implicit nil checks

**Rationale**: The current `PipelineDetailModel` infers state from nil checks on `availableDetail`/`finishedDetail`. Adding `configuring`, `launching`, and `error` states makes nil-check approach fragile. An explicit state enum:
- `stateEmpty` — no selection (placeholder message)
- `stateLoading` — fetching data
- `stateAvailableDetail` — showing available pipeline info
- `stateFinishedDetail` — showing finished pipeline results
- `stateRunningInfo` — showing running pipeline brief info
- `stateConfiguring` — argument form is active
- `stateLaunching` — brief "Starting..." indicator
- `stateError` — launch error message

**Alternatives Rejected**:
- Continue with nil checks + booleans: Fragile, hard to reason about transitions

## R5: `q`-to-quit Guard

**Decision**: Gate `q`-to-quit on `m.content.focus == FocusPaneLeft` in `AppModel.Update()`

**Rationale**: When the argument form is active, `q` typed in a text field must not quit the app. The current handler at `app.go:57` checks `!m.content.list.filtering` but not focus state. Adding the focus gate ensures `q` only quits from the left pane, consistent with how Esc is already focus-gated in `content.go:62`.

**Alternatives Rejected**:
- Adding a `formActive` boolean: Exposes internal state; focus pane is already the correct abstraction

## R6: Pipeline Loading for Launch

**Decision**: Load pipeline YAML on demand at launch time via existing `parsePipelineFile()` in `pipelines.go`

**Rationale**: The `PipelineLauncher` needs the full `pipeline.Pipeline` struct for the executor, not just `PipelineInfo`. The `AvailableDetail` already parsed the YAML and has the name. At launch time, re-parse the YAML to get the full `Pipeline` struct. This avoids caching full pipeline objects in memory when most won't be launched.

The `loadPipeline()` function in `cmd/wave/commands/run.go` loads from the manifest's pipeline references. We can either reuse that or scan the pipelines directory. Since the TUI already uses `DiscoverPipelines()` which scans the directory, we add a `LoadPipeline(name string) (*pipeline.Pipeline, error)` to the launcher that does a targeted YAML parse from the pipelines directory.

**Alternatives Rejected**:
- Caching all Pipeline structs at TUI startup: Memory waste for O(N) pipelines when only 1-2 launched
- Passing pre-loaded Pipeline through the form: Form only knows the name, loading happens at launch
