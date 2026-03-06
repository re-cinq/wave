# Tasks: TUI Pipeline Launch Flow

**Feature**: #256 — TUI Pipeline Launch Flow
**Branch**: `256-tui-pipeline-launch`
**Generated**: 2026-03-06
**Spec**: `specs/256-tui-pipeline-launch/spec.md`
**Plan**: `specs/256-tui-pipeline-launch/plan.md`

## Phase 1: Foundation — Messages, Types, and Dependencies

### Setup

- [X] T001 [P1] [Setup] Add new message types and data structures to `internal/tui/pipeline_messages.go`
  - Add `LaunchConfig` struct with `PipelineName`, `Input`, `ModelOverride`, `Flags []string`, `DryRun bool`
  - Add `LaunchDependencies` struct with `Manifest *manifest.Manifest`, `Store state.StateStore`, `PipelinesDir string`
  - Add `DetailPaneState` type (int) with constants: `stateEmpty`, `stateLoading`, `stateAvailableDetail`, `stateFinishedDetail`, `stateRunningInfo`, `stateConfiguring`, `stateLaunching`, `stateError`
  - Add `LaunchRequestMsg` struct with `Config LaunchConfig`
  - Add `PipelineLaunchedMsg` struct with `RunID string`, `PipelineName string`
  - Add `PipelineLaunchResultMsg` struct with `RunID string`, `Err error`
  - Add `LaunchErrorMsg` struct with `PipelineName string`, `Err error`
  - Add `ConfigureFormMsg` struct with `PipelineName string`, `InputExample string`
  - Add `FormActiveMsg` struct with `Active bool` (for status bar hint switching)
  - **Verify**: `go build ./internal/tui/...`

### Foundational

- [X] T002 [P1] [Foundation] Refactor `PipelineDetailModel` in `internal/tui/pipeline_detail.go` to use `DetailPaneState`
  - Add `paneState DetailPaneState` field to `PipelineDetailModel`
  - Set `paneState = stateEmpty` in `NewPipelineDetailModel()`
  - Update `PipelineSelectedMsg` handler: set `stateLoading` when fetching, `stateAvailableDetail`/`stateFinishedDetail`/`stateRunningInfo` based on `Kind`
  - Update `DetailDataMsg` handler: set `stateAvailableDetail` or `stateFinishedDetail` based on which detail is non-nil; set `stateError` on error
  - Update `View()` to switch on `m.paneState` instead of nil checks and boolean flags
  - Remove redundant `loading` and `errorMsg` fields (now covered by `paneState`)
  - **Constraint**: Existing behavior must be 100% preserved — this is a pure refactor
  - **Verify**: `go test ./internal/tui/... -run TestPipelineDetail`

- [X] T003 [P1] [Foundation] Add `LoadPipelineByName()` to `internal/tui/pipelines.go`
  - Add `LoadPipelineByName(dir, name string) (*pipeline.Pipeline, error)` function
  - Scan YAML files in `dir`, parse each with `yaml.Unmarshal`, return first match on `p.Metadata.Name == name`
  - Return `fmt.Errorf("pipeline not found: %s", name)` if no match
  - **Verify**: `go test ./internal/tui/... -run TestLoadPipeline`

- [X] T004 [P] [Foundation] Add unit tests for `LoadPipelineByName` in `internal/tui/pipelines_test.go`
  - Test: valid pipeline YAML returns correct `*pipeline.Pipeline`
  - Test: non-existent name returns error
  - Test: empty directory returns error
  - Test: malformed YAML files are skipped gracefully
  - Use `t.TempDir()` with temporary YAML files
  - **Verify**: `go test ./internal/tui/... -run TestLoadPipeline`

## Phase 2: Argument Form — US-1 (Launch Pipeline) & US-2 (Cancel Before Starting)

- [X] T005 [P1] [US-1] Add form fields and creation logic to `PipelineDetailModel` in `internal/tui/pipeline_detail.go`
  - Add fields: `launchForm *huh.Form`, `launchInput string`, `launchModel string`, `launchFlags []string`, `launchError string`
  - Handle `ConfigureFormMsg` in `Update()`: create `huh.Form` with:
    - `huh.NewInput().Title("Input").Placeholder(msg.InputExample).Value(&m.launchInput)` — pipeline input text field
    - `huh.NewInput().Title("Model override (optional)").Value(&m.launchModel)` — model override text field
    - `huh.NewMultiSelect[string]().Title("Options").Options(buildFlagOptions(DefaultFlags())...).Value(&m.launchFlags)` — flag multi-select
  - Apply `WaveTheme()` from `theme.go`, set `.WithWidth(m.width)`, `.WithHeight(m.height)`
  - Set `paneState = stateConfiguring`, call `form.Init()`
  - Reset `launchInput`, `launchModel`, `launchFlags` to zero values on form creation
  - **FR**: FR-001, FR-002, FR-003, FR-014, FR-016
  - **Verify**: `go build ./internal/tui/...`

- [X] T006 [P1] [US-1,US-2] Add form lifecycle handling in `PipelineDetailModel.Update()` in `internal/tui/pipeline_detail.go`
  - When `paneState == stateConfiguring` and msg is `tea.KeyMsg`: forward to `m.launchForm.Update(msg)`
  - After each form update, check `form.State`:
    - `huh.StateCompleted` → extract bound values, build `LaunchConfig`, emit `LaunchRequestMsg` via `tea.Cmd`, set `paneState = stateLaunching`
    - `huh.StateAborted` → set `launchForm = nil`, revert `paneState = stateAvailableDetail`, emit `FocusChangedMsg{Pane: FocusPaneLeft}` (signals content to restore left focus)
  - Handle form resizing in `SetSize()`: call `m.launchForm.WithWidth(w).WithHeight(h)` when form is non-nil
  - Cast `form.Update()` result back to `*huh.Form` (it returns `tea.Model`)
  - **FR**: FR-004, FR-005
  - **Verify**: `go build ./internal/tui/...`

- [X] T007 [P1] [US-1,US-2] Add form rendering in `PipelineDetailModel.View()` in `internal/tui/pipeline_detail.go`
  - `stateConfiguring`: render `m.launchForm.View()` within the pane dimensions
  - `stateLaunching`: render centered "Starting pipeline..." indicator with muted style
  - `stateError`: render error message in red with `m.launchError`, plus hint "[Esc] Back"
  - **FR**: FR-016
  - **Verify**: `go build ./internal/tui/...`

- [X] T008 [P] [US-1,US-2] Add form unit tests in `internal/tui/pipeline_detail_test.go`
  - Test: `ConfigureFormMsg` creates form with correct fields (input, model, flags) and sets `stateConfiguring`
  - Test: form abort (`huh.StateAborted`) reverts to `stateAvailableDetail` and nilifies form
  - Test: form completion (`huh.StateCompleted`) emits `LaunchRequestMsg` with correct `LaunchConfig`
  - Test: `View()` renders form view in `stateConfiguring` state
  - Test: `View()` renders "Starting..." in `stateLaunching` state
  - Test: `View()` renders error message in `stateError` state
  - Test: pane state refactor preserves all existing detail rendering (available, finished, running)
  - **Verify**: `go test ./internal/tui/... -run TestPipelineDetail`

## Phase 3: Pipeline Launcher Component — US-1 (Launch Pipeline) & US-3 (Cancel Running)

- [X] T009 [P1] [US-1,US-3] Create `PipelineLauncher` component in new file `internal/tui/pipeline_launcher.go`
  - Struct: `deps LaunchDependencies`, `cancelFns map[string]context.CancelFunc`, `mu sync.Mutex`
  - `NewPipelineLauncher(deps LaunchDependencies) *PipelineLauncher` constructor
  - `Launch(config LaunchConfig) tea.Cmd`:
    - Load full `pipeline.Pipeline` via `LoadPipelineByName(deps.PipelinesDir, config.PipelineName)`
    - If load fails, return cmd that emits `LaunchErrorMsg`
    - Create `context.WithCancel(context.Background())`
    - Resolve adapter: `adapter.NewMockAdapter()` if `--mock` in flags, else `adapter.ResolveAdapter()` from manifest
    - Create run ID via `deps.Store.CreateRun()` (fall back to `pipeline.GenerateRunID()`)
    - Create workspace manager via `workspace.NewWorkspaceManager()`
    - Build executor options mirroring `runRun()`: `WithEmitter`, `WithRunID`, `WithStateStore`, `WithWorkspaceManager`, `WithDebug`, `WithModelOverride`
    - Create audit logger if manifest audit config says so
    - Store cancel function in `cancelFns[runID]` (mutex-protected)
    - Return `tea.Batch(immediateCmd, executorCmd)`:
      - `immediateCmd`: returns `PipelineLaunchedMsg{RunID, PipelineName}`
      - `executorCmd`: calls `executor.Execute(ctx, p, manifest, input)`, updates run status in store, returns `PipelineLaunchResultMsg{RunID, err}`
  - `Cancel(runID string)`: look up cancel function in map, call if found
  - `CancelAll()`: iterate all cancel functions, call each, clear map
  - `Cleanup(runID string)`: remove entry from `cancelFns` (called when `PipelineLaunchResultMsg` received)
  - **FR**: FR-005, FR-006, FR-010, FR-011, FR-012, FR-017, FR-018, FR-020
  - **Verify**: `go build ./internal/tui/...`

- [X] T010 [P] [US-1,US-3] Add launcher unit tests in new file `internal/tui/pipeline_launcher_test.go`
  - Test: `NewPipelineLauncher` initializes empty cancel map
  - Test: `Cancel` on unknown runID is no-op (no panic)
  - Test: `CancelAll` invokes all stored cancel functions
  - Test: `CancelAll` on empty map is no-op
  - Test: `Cleanup` removes entry from cancel map
  - Use mock dependencies (nil manifest, nil store acceptable for Cancel/CancelAll tests)
  - **Verify**: `go test ./internal/tui/... -run TestPipelineLauncher`

## Phase 4: Content Model Integration — US-1, US-2, US-3

- [X] T011 [P1] [US-1,US-2,US-3] Integrate `PipelineLauncher` into `ContentModel` in `internal/tui/content.go`
  - Add `launcher *PipelineLauncher` field to `ContentModel`
  - Modify `NewContentModel()` signature: add `LaunchDependencies` parameter, create `PipelineLauncher` if deps are non-zero
  - Add `CancelAll()` method on `ContentModel`: delegates to `launcher.CancelAll()` (nil-safe)
  - Modify Enter handling (existing `tea.KeyEnter` block): when cursor is on available item, also emit `ConfigureFormMsg` to detail (using selected pipeline's `InputExample` from available data)
  - Add message routing in `Update()`:
    - `LaunchRequestMsg` → call `launcher.Launch(config)`, set detail `paneState = stateLaunching`
    - `PipelineLaunchedMsg` → forward to list (for running entry insertion), transition focus to left pane, emit `FocusChangedMsg{Pane: FocusPaneLeft}`
    - `PipelineLaunchResultMsg` → call `launcher.Cleanup(msg.RunID)`
    - `LaunchErrorMsg` → forward to detail (set `launchError`, `paneState = stateError`), transition focus to left
    - `FormActiveMsg` → forward to status bar for hint switching
  - Handle `c` key: when `focus == FocusPaneLeft` and cursor on running item, call `launcher.Cancel(runID)` using the `RunID` from the selected running pipeline
  - Handle `FocusChangedMsg` from detail (form abort emits this): transition focus back to left pane
  - **FR**: FR-007, FR-008, FR-010, FR-013, FR-020
  - **Verify**: `go build ./internal/tui/...`

- [X] T012 [P] [US-1,US-2,US-3] Add content model integration tests in `internal/tui/content_test.go`
  - Update all `NewContentModel()` calls to pass zero-value `LaunchDependencies{}`
  - Test: Enter on available item emits `ConfigureFormMsg` and transitions focus right
  - Test: `LaunchRequestMsg` triggers launcher (mock or nil-safe)
  - Test: `PipelineLaunchedMsg` inserts running entry and transitions focus left
  - Test: `LaunchErrorMsg` sets error state on detail and transitions focus left
  - Test: `c` key on running item calls `Cancel(runID)` on launcher
  - Test: `c` key on non-running item (section header, available, finished) is no-op
  - Test: `CancelAll()` is nil-safe when launcher is nil
  - **Verify**: `go test ./internal/tui/... -run TestContentModel`

## Phase 5: Pipeline List — Running Entry Insertion — US-1

- [X] T013 [P1] [US-1] Handle `PipelineLaunchedMsg` in `PipelineListModel` in `internal/tui/pipeline_list.go`
  - Add `PipelineLaunchedMsg` case in `Update()`:
    - Create `RunningPipeline{RunID: msg.RunID, Name: msg.PipelineName, StartedAt: time.Now()}`
    - Prepend to `m.running` slice
    - Call `m.buildNavigableItems()` to rebuild navigation
    - Move cursor to the new running entry (find first `itemKindRunning` in `m.navigable`)
    - Return `tea.Batch` of `RunningCountMsg{Count: len(m.running)}` and `PipelineSelectedMsg{RunID: msg.RunID, Name: msg.PipelineName, Kind: itemKindRunning}`
  - **FR**: FR-007, FR-008
  - **Verify**: `go build ./internal/tui/...`

- [X] T014 [P] [US-1] Add list integration tests in `internal/tui/pipeline_list_test.go`
  - Test: `PipelineLaunchedMsg` prepends new running entry to running slice
  - Test: `PipelineLaunchedMsg` rebuilds navigable items with the new entry
  - Test: `PipelineLaunchedMsg` moves cursor to the new running entry
  - Test: `PipelineLaunchedMsg` emits `RunningCountMsg` with updated count
  - Test: existing running pipelines are preserved when new one is launched
  - **Verify**: `go test ./internal/tui/... -run TestPipelineList`

## Phase 6: App Model and Status Bar — US-1, US-2, US-3

- [X] T015 [P1] [US-1,US-2] Gate `q`-to-quit on left pane focus in `internal/tui/app.go`
  - Modify the `msg.String() == "q"` check: add `&& m.content.focus == FocusPaneLeft`
  - This prevents quitting when `q` is typed in a form text field
  - **FR**: FR-019
  - **Verify**: `go test ./internal/tui/... -run TestAppModel`

- [X] T016 [P1] [US-3] Add `CancelAll()` call on TUI exit in `internal/tui/app.go`
  - Before `tea.Quit` on both `q` and `Ctrl+C` paths, call `m.content.CancelAll()`
  - **FR**: FR-017
  - **Verify**: `go build ./internal/tui/...`

- [X] T017 [P1] [US-1] Update `NewAppModel()` to accept `LaunchDependencies` in `internal/tui/app.go`
  - Add `LaunchDependencies` parameter to `NewAppModel()`
  - Pass through to `NewContentModel()`
  - Update `RunTUI()` to accept `LaunchDependencies` and pass to `NewAppModel()`
  - **Verify**: `go build ./internal/tui/...`

- [X] T018 [P2] [US-1] Add form-context hints to `StatusBarModel` in `internal/tui/statusbar.go`
  - Add `formActive bool` field
  - Handle `FormActiveMsg` in `Update()`: set `m.formActive = msg.Active`
  - In `View()`: when `m.formActive && m.focusPane == FocusPaneRight`, render `"Tab: next  Shift+Tab: prev  Enter: launch  Esc: cancel"` instead of default right-pane hints
  - **FR**: FR-009
  - **Verify**: `go build ./internal/tui/...`

- [X] T019 [P] [US-1,US-2] Add app model and status bar tests
  - In `internal/tui/app_test.go`:
    - Update all `NewAppModel()` calls to pass zero-value `LaunchDependencies{}`
    - Test: `q` key with `focus == FocusPaneRight` does NOT quit (forwarded to content)
    - Test: `q` key with `focus == FocusPaneLeft` still quits
    - Test: `CancelAll()` is called before `tea.Quit` (verify via launcher mock or nil-safe)
  - In `internal/tui/statusbar_test.go`:
    - Test: `FormActiveMsg{Active: true}` + `FocusPaneRight` shows form hints
    - Test: `FormActiveMsg{Active: false}` reverts to default hints
  - **Verify**: `go test ./internal/tui/... -run "TestAppModel|TestStatusBar"`

## Phase 7: CLI Integration

- [X] T020 [P1] [US-1] Update CLI entry points to pass `LaunchDependencies` in `cmd/wave/main.go`
  - Modify `shouldLaunchTUI` block in `rootCmd.RunE`:
    - Load manifest from default path `wave.yaml`
    - Open state store from `.wave/state.db` (nil-safe if unavailable)
    - Determine pipelines directory from manifest or default `.wave/pipelines`
    - Construct `tui.LaunchDependencies{Manifest: &m, Store: store, PipelinesDir: pipelinesDir}`
    - Pass to `tui.RunTUI(deps)`
  - Handle errors gracefully: if manifest/store unavailable, pass zero values (TUI still works for browsing, just can't launch)
  - **Verify**: `go build ./cmd/wave/...`

## Phase 8: Error Handling — US-4 (Launch Errors)

- [X] T021 [P2] [US-4] Handle `LaunchErrorMsg` display in `PipelineDetailModel` in `internal/tui/pipeline_detail.go`
  - In `Update()`, handle `LaunchErrorMsg`: set `launchError = msg.Err.Error()`, `paneState = stateError`, clear form
  - In `View()` stateError case: render error with title "Launch Failed", error message, and "[Esc] Back to detail" hint
  - Handle Esc in `stateError`: revert to `stateAvailableDetail`, clear `launchError`
  - **FR**: FR-013
  - **Verify**: `go build ./internal/tui/...`

- [X] T022 [P] [US-4] Add error handling tests in `internal/tui/pipeline_detail_test.go`
  - Test: `LaunchErrorMsg` sets `paneState = stateError` and stores error message
  - Test: `View()` in `stateError` renders the error message
  - Test: Esc from `stateError` reverts to `stateAvailableDetail`
  - Test: re-opening form after error (Enter on same pipeline) shows fresh form
  - **Verify**: `go test ./internal/tui/... -run TestPipelineDetail`

## Phase 9: Dry-Run Support — US-5

- [X] T023 [P3] [US-5] Add dry-run detection in `PipelineLauncher.Launch()` in `internal/tui/pipeline_launcher.go`
  - Check `config.DryRun` (or `--dry-run` in `config.Flags`)
  - If dry-run: load pipeline, call executor dry-run path (or format step plan manually), return result as `PipelineLaunchResultMsg` without creating a run record
  - The detail pane shows the execution plan in the right pane instead of transitioning to running
  - **FR**: FR-015
  - **Verify**: `go build ./internal/tui/...`

## Phase 10: Polish & Cross-Cutting

- [X] T024 [P] [Polish] Ensure all existing tests pass with updated signatures
  - Update any remaining `NewAppModel()`, `NewContentModel()` calls across all test files in `internal/tui/`
  - Run `go test ./internal/tui/...` and fix any compilation or test failures
  - Run `go test ./...` to ensure no regressions across the entire project
  - **SC**: SC-006
  - **Verify**: `go test ./... -race`

- [X] T025 [P] [Polish] Add form rendering dimension tests in `internal/tui/pipeline_detail_test.go`
  - Test: form renders correctly at width 80 (minimum terminal width)
  - Test: form renders correctly at width 300 (wide terminal)
  - Test: form renders correctly at height 24 (minimum terminal height)
  - Test: resizing while form is active updates form dimensions
  - **SC**: SC-007
  - **Verify**: `go test ./internal/tui/... -run TestPipelineDetail`

## Dependency Graph

```
T001 ──→ T002 ──→ T005 ──→ T006 ──→ T007 ──→ T008
  │        │                                    │
  │        └──→ T003 ──→ T004                   │
  │              │                               │
  │              └──→ T009 ──→ T010              │
  │                     │                        │
  │                     └──→ T011 ──→ T012       │
  │                            │                 │
  │                            └──→ T013 ──→ T014
  │                            │
  │                            └──→ T015 ──→ T016 ──→ T017 ──→ T019
  │                                                     │
  │                                                     └──→ T020
  │
  └──→ T018 ──→ T019
  
T007 ──→ T021 ──→ T022
T009 ──→ T023
T019 ──→ T024 ──→ T025
```

## Parallelization Opportunities

Tasks marked [P] can run in parallel with their siblings at the same phase level:
- T003, T004 can run in parallel with T005–T008 (after T002)
- T008, T010 can run in parallel (independent test files)
- T012, T014, T019 can run in parallel (independent test files after their code deps)
- T024, T025 should run last as cross-cutting validation

## Summary

| Phase | Tasks | Priority | User Stories |
|-------|-------|----------|-------------|
| 1. Foundation | T001–T004 | P1 | Setup |
| 2. Argument Form | T005–T008 | P1 | US-1, US-2 |
| 3. Launcher | T009–T010 | P1 | US-1, US-3 |
| 4. Content Integration | T011–T012 | P1 | US-1, US-2, US-3 |
| 5. List Integration | T013–T014 | P1 | US-1 |
| 6. App & Status Bar | T015–T019 | P1/P2 | US-1, US-2, US-3 |
| 7. CLI Integration | T020 | P1 | US-1 |
| 8. Error Handling | T021–T022 | P2 | US-4 |
| 9. Dry-Run | T023 | P3 | US-5 |
| 10. Polish | T024–T025 | — | Cross-cutting |

**Total**: 25 tasks across 10 phases
**P1 tasks**: 14 (core launch flow)
**P2 tasks**: 3 (error handling, status bar hints)
**P3 tasks**: 1 (dry-run)
**Parallel opportunities**: 7 tasks can execute in parallel with siblings
