# Tasks: Pipeline Composition UI (#261)

**Branch**: `261-tui-compose-ui` | **Generated**: 2026-03-07
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md) | **Data Model**: [data-model.md](data-model.md)

## Phase 1: Setup & Message Types

- [X] T001 [P1] [Setup] Define compose-mode Bubble Tea message types in `internal/tui/compose_messages.go`
  - `ComposeActiveMsg{Active bool}` — status bar hint switching
  - `ComposeSequenceChangedMsg{Sequence, CompatibilityResult}` — sequence modified
  - `ComposeStartMsg{Sequence}` — user pressed Enter to start
  - `ComposeCancelMsg{}` — user pressed Esc to exit compose mode
  - `ComposeFocusDetailMsg{}` — Enter on boundary focuses right pane detail
  - Follow existing message patterns in `internal/tui/pipeline_messages.go`

- [X] T002 [P1] [Setup] Add `stateComposing` to `DetailPaneState` enum in `internal/tui/pipeline_messages.go`
  - Add after `stateError` (line ~35): `stateComposing // Compose mode artifact flow`
  - This is a single-line addition to the existing `const` block

## Phase 2: Core Data Types & Validation Logic

- [X] T003 [P1] [US1] Define `Sequence`, `SequenceEntry`, `ArtifactFlow`, `FlowMatch`, `MatchStatus`, `CompatibilityResult`, `CompatibilityStatus` types in `internal/tui/compose.go`
  - `Sequence` struct with `Entries []SequenceEntry`
  - `SequenceEntry` struct with `PipelineName string`, `Pipeline *pipeline.Pipeline`
  - `ArtifactFlow` struct with `SourcePipeline`, `TargetPipeline string`, `Outputs []pipeline.ArtifactDef`, `Inputs []pipeline.ArtifactRef`, `Matches []FlowMatch`
  - `FlowMatch` struct with `OutputName`, `InputName`, `InputAs string`, `Status MatchStatus`, `Optional bool`
  - `MatchStatus` iota: `MatchCompatible`, `MatchMissing`, `MatchUnmatched`
  - `CompatibilityResult` struct with `Flows []ArtifactFlow`, `Status CompatibilityStatus`, `Diagnostics []string`
  - `CompatibilityStatus` iota: `CompatibilityValid`, `CompatibilityWarning`, `CompatibilityError`
  - `IsReady() bool` on `CompatibilityResult`
  - Sequence methods: `Add()`, `Remove()`, `MoveUp()`, `MoveDown()`, `Len()`, `IsEmpty()`, `IsSingle()`
  - See data-model.md for exact struct definitions and behavior spec

- [X] T004 [P1] [US3] Implement `ValidateSequence(seq Sequence) CompatibilityResult` in `internal/tui/compose.go`
  - Iterate adjacent pairs: `seq.Entries[i]` and `seq.Entries[i+1]`
  - Extract outputs from `source.Pipeline.Steps[last].OutputArtifacts` (`ArtifactDef.Name`)
  - Extract inputs from `target.Pipeline.Steps[0].Memory.InjectArtifacts` (`ArtifactRef.Artifact`)
  - Match by name: `output.Name == input.Artifact` → `MatchCompatible`
  - Missing required input (`!input.Optional`) → `MatchMissing`, `CompatibilityError`
  - Missing optional input (`input.Optional`) → `MatchMissing`, `CompatibilityWarning`
  - Unmatched output (produced but not consumed) → `MatchUnmatched` (informational, no status change)
  - Empty/single-entry sequence → `CompatibilityValid` with no flows
  - Generate diagnostic strings: `"speckit-flow → wave-evolve: missing required input 'review_input'"`
  - See contracts/compose-validation.md for status rules and match rules

- [X] T005 [P1] [US3] Write table-driven tests for `ValidateSequence` in `internal/tui/compose_test.go`
  - Test: two compatible pipelines (all inputs satisfied) → `CompatibilityValid`
  - Test: two incompatible pipelines (required input missing) → `CompatibilityError`
  - Test: optional input missing → `CompatibilityWarning`
  - Test: pipeline with no output artifacts → warning for next pipeline's inputs
  - Test: pipeline with no input artifacts → `CompatibilityValid` (nothing to match)
  - Test: three-pipeline chain with mixed compatibility at different boundaries
  - Test: single pipeline → `CompatibilityValid`, zero flows
  - Test: empty sequence → `CompatibilityValid`, zero flows
  - Test: unmatched outputs (output produced but not consumed) → informational, not error
  - Create test helper to build `pipeline.Pipeline` with configurable steps/artifacts

## Phase 3: Compose List Model (Left Pane — US1: Sequence Builder)

- [X] T006 [P1] [US1] Create `ComposeListModel` struct and constructor in `internal/tui/compose_list.go`
  - Fields: `width`, `height`, `focused`, `sequence Sequence`, `cursor int`, `picking bool`, `picker *huh.Form`, `pickerValue string`, `available []PipelineInfo`, `validation CompatibilityResult`
  - `NewComposeListModel(initial PipelineInfo, initialPipeline *pipeline.Pipeline, available []PipelineInfo) ComposeListModel`
  - Initialize with `initial` as first entry, `cursor = 0`

- [X] T007 [P1] [US1] Implement `Init()`, `Update()`, `View()` for `ComposeListModel` in `internal/tui/compose_list.go`
  - `Init()`: return nil (no async init needed)
  - `Update()` key handling:
    - `↑`/`↓` (`tea.KeyUp`/`tea.KeyDown`): cursor navigation within sequence
    - `shift+↑`/`shift+↓` (`tea.KeyShiftUp`/`tea.KeyShiftDown`): reorder — swap `sequence.Entries[cursor]` with adjacent, update cursor
    - `a`: set `picking = true`, create `huh.Select` form from `available` pipelines (allow duplicates)
    - `x`: remove entry at cursor via `sequence.Remove(cursor)`, adjust cursor. Disable on empty sequence
    - `Enter`: if `picking`, complete picker selection. If not picking: if sequence is empty → no-op; if single → emit `ComposeStartMsg` (delegate to normal launch); if multi → emit `ComposeStartMsg` (with confirmation if `!validation.IsReady()`)
    - `Esc`: if `picking`, cancel picker. If not picking → emit `ComposeCancelMsg`
  - After any sequence mutation (add/remove/reorder): call `ValidateSequence()` and emit `ComposeSequenceChangedMsg`
  - `View()`: render numbered list with cursor indicator (`▸`), pipeline name, compatibility status icons per data-model.md. Show picker when `picking == true`

- [X] T008 [P1] [US1] Write unit tests for `ComposeListModel` in `internal/tui/compose_list_test.go`
  - Test: cursor navigation (up/down within bounds)
  - Test: reorder swaps entries and updates cursor position
  - Test: remove deletes entry and adjusts cursor (edge: remove last item)
  - Test: add via picker appends entry and re-validates
  - Test: Esc during picker cancels picker, Esc outside picker emits `ComposeCancelMsg`
  - Test: Enter on empty sequence is no-op
  - Test: Enter on single-entry sequence emits `ComposeStartMsg`
  - Test: duplicate pipeline allowed (same pipeline added twice)
  - Use `tea.KeyMsg` simulation pattern from `internal/tui/pipeline_list_test.go`

## Phase 4: Compose Detail Model (Right Pane — US2: Artifact Flow Visualization)

- [X] T009 [P2] [US2] Create `ComposeDetailModel` struct and constructor in `internal/tui/compose_detail.go`
  - Fields: `width`, `height`, `focused`, `viewport viewport.Model`, `validation CompatibilityResult`, `focusedIdx int`
  - `NewComposeDetailModel() ComposeDetailModel`
  - Initialize viewport with zero size (will be set via `SetSize`)

- [X] T010 [P2] [US2] Implement `renderArtifactFlow(result CompatibilityResult, width int) string` in `internal/tui/compose_detail.go`
  - **Full mode** (width >= 120): box-drawing characters with `┌─┐`, `│`, `└─┘`, connection lines with `┬`/`┴`
    - Each pipeline shown as a box with name and its artifacts listed
    - Connections between boxes showing match status: `✓` (green), `⚠` (yellow/optional), `✗` (red/missing)
  - **Compact mode** (width < 120): text-only summary per research.md R3
    - `speckit-flow → wave-evolve`
    - `  ✓ spec-status → spec_info (match)`
    - `  ✗ review_input (missing — no matching output)`
  - Use lipgloss for color styling: green for compatible, yellow for optional warning, red for missing required
  - Handle empty validation result: show "Add pipelines to see artifact flow"

- [X] T011 [P2] [US2] Implement `Init()`, `Update()`, `View()` for `ComposeDetailModel` in `internal/tui/compose_detail.go`
  - `Init()`: return nil
  - `Update()`: handle `ComposeSequenceChangedMsg` → re-render viewport content via `renderArtifactFlow`. Handle viewport scroll keys when focused (`↑`/`↓`, `pgup`/`pgdn`)
  - `View()`: render viewport with border styling matching existing detail pane
  - `SetSize(w, h int)`: update viewport dimensions
  - `SetFocused(focused bool)`: toggle focus state

- [X] T012 [P2] [US2] Write unit tests for `ComposeDetailModel` in `internal/tui/compose_detail_test.go`
  - Test: render with compatible flows shows green `✓` indicators
  - Test: render with missing required input shows red `✗` indicators
  - Test: render with optional mismatch shows yellow `⚠` indicators
  - Test: render degrades to text-only below 120 columns
  - Test: empty validation renders placeholder text
  - Test: viewport scrolling works with content taller than viewport

## Phase 5: Content Model Integration (US1+US2+US3 Wiring)

- [X] T013 [P1] [US1] Add compose mode fields to `ContentModel` in `internal/tui/content.go`
  - Add fields: `composing bool`, `composeList *ComposeListModel`, `composeDetail *ComposeDetailModel`
  - These are nil when compose mode is inactive (same pattern as lazy-init alternative views)

- [X] T014 [P1] [US1] Handle `s` key in `ContentModel.Update()` to enter compose mode in `internal/tui/content.go`
  - Gate: `currentView == ViewPipelines && focus == FocusPaneLeft && !list.filtering`
  - Gate: cursor must be on an `itemKindAvailable` item (FR-015)
  - Load full pipeline via `LoadPipelineByName(launcher.deps.PipelinesDir, selectedPipeline.Name)`
  - Create `ComposeListModel` with selected pipeline as first entry
  - Create `ComposeDetailModel`
  - Set `composing = true`
  - Emit `ComposeActiveMsg{Active: true}` and `ComposeSequenceChangedMsg` (initial validation)
  - Block `Tab` view cycling when `composing == true` (same as `stateConfiguring` pattern at line ~240)

- [X] T015 [P1] [US1] Route messages to compose models when `composing == true` in `internal/tui/content.go`
  - When `composing && focus == FocusPaneLeft`: forward key messages to `composeList.Update()`
  - When `composing && focus == FocusPaneRight`: forward key messages to `composeDetail.Update()`
  - Handle `ComposeCancelMsg`: set `composing = false`, nil compose models, restore focus to left pane, emit `ComposeActiveMsg{Active: false}`
  - Handle `ComposeStartMsg`: if single-entry → delegate to `PipelineLauncher` (normal launch). If multi-entry → show informational message about #249 dependency. Set `composing = false`
  - Handle `ComposeSequenceChangedMsg`: update both `composeList.validation` and `composeDetail` with new validation result
  - Handle `ComposeFocusDetailMsg`: switch focus to right pane for artifact flow scrolling. First `Esc` returns to left pane (second `Esc` exits compose mode entirely)

- [X] T016 [P1] [US1] Update `ContentModel.View()` to render compose models when `composing` in `internal/tui/content.go`
  - When `composing == true`: render `composeList.View()` in left pane slot, `composeDetail.View()` in right pane slot
  - Use same `lipgloss.JoinHorizontal` layout as normal view rendering
  - Propagate `SetSize` to compose models when `composing` is active

- [X] T017 [P1] [US1] Write integration tests for compose mode entry/exit in `internal/tui/content_test.go`
  - Test: pressing `s` on available pipeline enters compose mode
  - Test: pressing `s` on running/finished pipeline does nothing (FR-015)
  - Test: pressing `s` when not in ViewPipelines does nothing (FR-015)
  - Test: pressing `s` when right pane is focused does nothing (FR-015)
  - Test: `ComposeCancelMsg` exits compose mode and restores normal view
  - Test: Tab key is blocked during compose mode
  - Test: `ComposeStartMsg` with single entry delegates to normal launch

## Phase 6: Status Bar Integration (US1: Keybinding Hints)

- [X] T018 [P2] [US1] Add compose mode hint switching to `StatusBarModel` in `internal/tui/statusbar.go`
  - Add `composeActive bool` field to `StatusBarModel` struct
  - Handle `ComposeActiveMsg` in `Update()`: set `composeActive = msg.Active`
  - Add compose mode hint text in `View()`: `"a: add  x: remove  Shift+↑↓: reorder  Enter: start  Esc: cancel"`
  - Insert before other hint checks (after `formActive` check) so compose hints take priority when active

- [X] T019 [P2] [US1] Forward `ComposeActiveMsg` from `AppModel.Update()` to `StatusBarModel` in `internal/tui/app.go`
  - Add `ComposeActiveMsg` to the message forwarding in `AppModel.Update()` (same pattern as `FormActiveMsg`, `LiveOutputActiveMsg`, `FinishedDetailActiveMsg`)
  - The message is already forwarded to content via existing routing — just ensure status bar also receives it

- [X] T020 [P2] [US1] Write tests for status bar compose mode hints in `internal/tui/statusbar_test.go`
  - Test: `ComposeActiveMsg{Active: true}` causes compose hints to render
  - Test: `ComposeActiveMsg{Active: false}` restores default hints
  - Test: compose hints contain all expected keybindings (`a`, `x`, `Shift+↑↓`, `Enter`, `Esc`)

## Phase 7: CLI `wave compose` Command (US5)

- [X] T021 [P3] [US5] Create `NewComposeCmd() *cobra.Command` in `cmd/wave/commands/compose.go`
  - `Use: "compose [pipelines...]"`, `Short: "Validate and execute a pipeline sequence"`
  - `Args: cobra.MinimumNArgs(2)` — at least 2 pipelines for a sequence
  - `--validate-only` flag (bool, default false): check compatibility without executing
  - `--manifest` flag: inherited from root persistent flag
  - Load manifest, resolve `pipelines_dir` from manifest or default `.wave/pipelines`
  - For each pipeline name arg: call `tui.LoadPipelineByName(pipelinesDir, name)` to get `*pipeline.Pipeline`
  - Build `tui.Sequence` from loaded pipelines
  - Call `tui.ValidateSequence(seq)` to get `CompatibilityResult`
  - If `--validate-only`: print compatibility report to stdout (format per contracts/compose-validation.md) and exit 0 (valid) or exit 1 (error)
  - If not `--validate-only` and valid: print informational message that sequential execution requires #249
  - If not `--validate-only` and invalid: print errors to stderr and exit 1
  - Call `checkOnboarding()` before execution (same as other commands)

- [X] T022 [P3] [US5] Register `compose` command in `cmd/wave/main.go`
  - Add `rootCmd.AddCommand(commands.NewComposeCmd())` after existing `AddCommand` calls (around line 112)

- [X] T023 [P3] [US5] Write tests for `wave compose` CLI command in `cmd/wave/commands/compose_test.go`
  - Test: `compose p1 p2` with valid pipelines → exit 0
  - Test: `compose p1 p2` with incompatible artifacts → exit 1 with error message
  - Test: `compose --validate-only p1 p2` → prints compatibility report
  - Test: `compose p1` (only one arg) → error "requires at least 2 arg(s)"
  - Test: `compose nonexistent p1` → error "pipeline not found"
  - Create test pipeline YAML fixtures in `cmd/wave/commands/testdata/`

## Phase 8: Edge Cases & Polish

- [X] T024 [P2] [US1] Handle edge case: removing all pipelines from sequence in `internal/tui/compose_list.go`
  - When sequence becomes empty after `x` key: keep compose mode open
  - Disable `Enter` (start) action when sequence is empty
  - Show "No pipelines in sequence" placeholder text in View()

- [X] T025 [P2] [US1] Handle edge case: duplicate pipeline notice in `internal/tui/compose_list.go`
  - When adding a pipeline that already exists in the sequence: allow it (FR-001 spec allows duplicates)
  - Show a subtle "(duplicate)" indicator next to the duplicated pipeline name in the list view

- [X] T026 [P2] [US3] Handle edge case: confirmation prompt for incompatible sequences in `internal/tui/compose_list.go`
  - When `Enter` is pressed and `!validation.IsReady()`: show a confirmation prompt (FR-008)
  - Use `huh.Confirm` inline form: "Sequence has artifact incompatibilities. Start anyway?"
  - If confirmed → emit `ComposeStartMsg`. If cancelled → return to compose mode

- [X] T027 [P2] [US2] Handle edge case: middle pipeline removal re-validates adjacent flows in `internal/tui/compose.go`
  - `Sequence.Remove(index)` should trigger re-validation of the now-adjacent pipelines
  - Already handled by `ComposeSequenceChangedMsg` → `ValidateSequence()` pipeline — verify with test

- [X] T028 [P3] [US1] Handle edge case: no available pipelines to add in `internal/tui/compose_list.go`
  - When `a` is pressed but all pipelines are already in the sequence: still show picker (duplicates allowed per spec)
  - When there are zero available pipelines total: disable `a` key, show "(no pipelines available)" in status

- [X] T029 [P1] [US4] Handle edge case: single-pipeline sequence delegates to normal launch in `internal/tui/content.go`
  - When `ComposeStartMsg` received with `sequence.IsSingle()`: extract the single pipeline entry and delegate to `PipelineLauncher` via `LaunchRequestMsg`
  - This avoids requiring #249 for single-pipeline sequences

- [X] T030 [P3] [US4] Stub grouped running display for future #249 integration in `internal/tui/pipeline_list.go`
  - Define `RunningSequence` struct: `Label string` (e.g. "speckit-flow → wave-evolve"), `Entries []RunningPipeline`, `ActiveIndex int`
  - Add `sequences []RunningSequence` field to `PipelineListModel` (unused until #249)
  - Add `// TODO(#249): Render grouped sequence items in Running section` comment
  - Do NOT implement rendering — just define the data structure for future use

- [X] T031 [P2] [US4] Show informational message when #249 is not available in `internal/tui/content.go`
  - When `ComposeStartMsg` received with multi-pipeline sequence: display a styled message in the right pane
  - Message: "Sequential pipeline execution requires cross-pipeline artifact handoff (#249). Build and validate your sequence now — execution will be enabled in a future release."
  - Use lipgloss styled block with info icon

- [X] T032 [P1] [Verify] Run `go test ./...` and `go vet ./...` to verify all changes compile and pass
  - Run full test suite after all tasks complete
  - Fix any compilation errors or test failures
  - Ensure no new `go vet` warnings

## Dependency Graph

```
T001, T002 (Setup — no dependencies)
  ↓
T003 (types, depends on T001 for message types)
  ↓
T004 (validation, depends on T003)
  ↓
T005 (tests, depends on T004)
  ↓
T006 → T007 → T008 (compose list model, depends on T003+T004)
T009 → T010 → T011 → T012 (compose detail model, depends on T003+T004) [P]
  ↓
T013 → T014 → T015 → T016 → T017 (content integration, depends on T007+T011)
  ↓
T018 → T019 → T020 (status bar, depends on T001) [P with Phase 3-5]
T021 → T022 → T023 (CLI command, depends on T003+T004) [P with Phase 3-6]
  ↓
T024-T031 (edge cases, depends on respective phase completion)
  ↓
T032 (final verification, depends on all)
```

[P] = can be parallelized with adjacent phases

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 32 |
| P1 (Critical) | 16 |
| P2 (Important) | 12 |
| P3 (Nice-to-have) | 4 |
| New files | 10 |
| Modified files | 5 |
| Phases | 8 |
| Parallel opportunities | 6 (T001/T002, T006-T008 ∥ T009-T012, T018-T020 ∥ T021-T023, various edge cases) |
