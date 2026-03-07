# Implementation Plan: Pipeline Composition UI (#261)

**Branch**: `261-tui-compose-ui` | **Date**: 2026-03-07 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/261-tui-compose-ui/spec.md`

## Summary

Build the pipeline composition UI enabling users to chain multiple pipelines into sequences with artifact flow visualization and compatibility validation. The feature adds compose mode (modal state within Pipelines view), artifact matching logic, a `wave compose` CLI command, and status bar integration. Execution is gated on #249 (cross-pipeline artifact handoff) — the UI shows an informational message until that dependency is available.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: Bubble Tea (`github.com/charmbracelet/bubbletea`), lipgloss, huh (forms), viewport (scrolling), cobra (CLI)
**Storage**: N/A (compose is transient — no persistence needed)
**Testing**: `go test ./...` with table-driven tests, `-race` required for PR
**Target Platform**: Linux/macOS terminals, 80+ column width
**Project Type**: Single Go binary (existing project)
**Performance Goals**: All compose mode interactions within a single Bubble Tea frame update (SC-004)
**Constraints**: Graceful degradation below 120 columns (FR-014), single binary (Constitution P1)
**Scale/Scope**: ~10 new files, ~1500-2000 LOC (including tests)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ PASS | No new dependencies — uses existing Bubble Tea/lipgloss/huh/cobra |
| P2: Manifest as Truth | ✅ PASS | Pipeline definitions read from manifest; no new config files |
| P3: Persona-Scoped | ✅ N/A | TUI feature, not a pipeline step |
| P4: Fresh Memory | ✅ N/A | TUI feature, not a pipeline step |
| P5: Navigator-First | ✅ N/A | TUI feature, not a pipeline step |
| P6: Contracts at Handover | ✅ N/A | TUI feature, not a pipeline step |
| P7: Relay via Summarizer | ✅ N/A | TUI feature, not a pipeline step |
| P8: Ephemeral Workspaces | ✅ N/A | TUI feature, not a pipeline step |
| P9: Credentials Never Disk | ✅ PASS | No credential handling |
| P10: Observable Progress | ✅ PASS | Compose mode emits structured messages via Bubble Tea bus |
| P11: Bounded Recursion | ✅ N/A | No meta-pipeline or recursion |
| P12: Minimal State Machine | ✅ PASS | Adds `stateComposing` as a new detail pane state — consistent with existing pattern |
| P13: Test Ownership | ✅ PASS | All new code will have table-driven tests; `go test ./...` required |

No violations found. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/261-tui-compose-ui/
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
├── contracts/           # Phase 1 contracts
└── tasks.md             # Phase 2 output (not created by /speckit.plan)
```

### Source Code (repository root)

```
internal/tui/
├── compose.go              # NEW: Sequence, ArtifactFlow, CompatibilityResult types + ValidateSequence()
├── compose_test.go          # NEW: Unit tests for validation logic
├── compose_list.go          # NEW: ComposeListModel (sequence builder left pane)
├── compose_list_test.go     # NEW: Unit tests for compose list
├── compose_detail.go        # NEW: ComposeDetailModel (artifact flow right pane)
├── compose_detail_test.go   # NEW: Unit tests for compose detail
├── compose_messages.go      # NEW: Compose-mode message types
├── content.go               # MODIFIED: Compose mode integration
├── pipeline_messages.go     # MODIFIED: Add stateComposing
├── statusbar.go             # MODIFIED: Compose mode hints
├── app.go                   # MODIFIED: Forward ComposeActiveMsg to status bar
└── pipelines.go             # MODIFIED: LoadPipelineByName used by compose

cmd/wave/commands/
├── compose.go               # NEW: wave compose CLI command
└── compose_test.go          # NEW: CLI command tests
```

**Structure Decision**: All compose UI code lives in `internal/tui/` following the established single-package pattern. The compose-specific types (`Sequence`, `ArtifactFlow`, `CompatibilityResult`, `ValidateSequence`) are in `internal/tui/compose.go` since they need access to `pipeline.Pipeline` and `PipelineInfo` types already imported by the TUI package. The CLI command lives in `cmd/wave/commands/compose.go` following existing command structure.

## Implementation Phases

### Phase A: Core Data Types & Validation Logic
**Files**: `internal/tui/compose.go`, `internal/tui/compose_test.go`, `internal/tui/compose_messages.go`

1. Define `Sequence`, `SequenceEntry`, `ArtifactFlow`, `FlowMatch`, `MatchStatus`, `CompatibilityResult`, `CompatibilityStatus` types
2. Implement `ValidateSequence(seq Sequence) CompatibilityResult`
   - Iterate adjacent pairs in sequence
   - Extract last step `OutputArtifacts` from pipeline N
   - Extract first step `Memory.InjectArtifacts` from pipeline N+1
   - Match by `ArtifactDef.Name` == `ArtifactRef.Artifact`
   - Mark `MatchCompatible`, `MatchMissing`, `MatchUnmatched`
   - Respect `ArtifactRef.Optional` for warning vs error status
3. Define message types: `ComposeActiveMsg`, `ComposeSequenceChangedMsg`, `ComposeStartMsg`, `ComposeCancelMsg`
4. Write table-driven tests covering:
   - Two compatible pipelines (all inputs satisfied)
   - Two incompatible pipelines (required input missing)
   - Optional input missing (warning, not error)
   - Pipeline with no outputs
   - Pipeline with no inputs
   - Three-pipeline chain with mixed compatibility
   - Single pipeline (no boundaries to validate)
   - Empty sequence

### Phase B: Compose List Model (Left Pane)
**Files**: `internal/tui/compose_list.go`, `internal/tui/compose_list_test.go`

1. Create `ComposeListModel` with `Sequence`, cursor, picker state, available pipelines
2. Implement `Init()`, `Update()`, `View()` following Bubble Tea conventions
3. Key handling:
   - `↑`/`↓`: cursor navigation
   - `shift+↑`/`shift+↓`: reorder (swap entry at cursor with adjacent)
   - `a`: open pipeline picker (inline `huh.Select` or custom overlay)
   - `x`: remove entry at cursor
   - `Enter`: emit `ComposeStartMsg` (with confirmation if incompatible)
   - `Esc`: emit `ComposeCancelMsg`
4. Pipeline picker: when `a` is pressed, show a filterable list of available pipelines. After selection, append to sequence and re-validate
5. Render: numbered list with cursor indicator, visual markers for compatibility status
6. Tests:
   - Navigation moves cursor
   - Reorder swaps entries
   - Remove deletes entry
   - Add appends entry
   - Esc emits cancel
   - Enter on empty/single sequence (edge cases)

### Phase C: Compose Detail Model (Right Pane)
**Files**: `internal/tui/compose_detail.go`, `internal/tui/compose_detail_test.go`

1. Create `ComposeDetailModel` with viewport, validation result, width/height
2. Implement `renderArtifactFlow(result CompatibilityResult, width int) string`
   - Full mode (width >= 120): box-drawing characters with connection lines
   - Compact mode (width < 120): text-only summary with status indicators
3. Color coding: green ✓ for compatible, yellow ⚠ for optional mismatch, red ✗ for missing required
4. Update viewport content when `ComposeSequenceChangedMsg` is received
5. Tests:
   - Render with compatible flows shows green indicators
   - Render with missing required shows red indicators
   - Render degrades gracefully below 120 columns
   - Empty sequence renders placeholder

### Phase D: Content Model Integration
**Files**: `internal/tui/content.go`, `internal/tui/pipeline_messages.go`

1. Add `stateComposing` to `DetailPaneState` enum
2. Add `composing bool`, `composeList *ComposeListModel`, `composeDetail *ComposeDetailModel` to `ContentModel`
3. Handle `s` key in `ContentModel.Update()`:
   - Only when `currentView == ViewPipelines`, `focus == FocusPaneLeft`, cursor on available item
   - Create `ComposeListModel` with selected pipeline as first entry
   - Create `ComposeDetailModel`
   - Set `composing = true`
   - Block `Tab` cycling while composing (same as `stateConfiguring`)
4. Route key messages to compose models when `composing == true`
5. Handle `ComposeCancelMsg`: tear down compose models, set `composing = false`, restore normal view
6. Handle `ComposeStartMsg`: if #249 available, invoke; else show informational message
7. Handle `ComposeSequenceChangedMsg`: update both compose models with new validation
8. Update `View()`: when `composing`, render `composeList.View()` in left pane, `composeDetail.View()` in right pane
9. Gate focus transitions: `Enter` on sequence item focuses right pane for detail inspection, `Esc` returns to left pane (second `Esc` exits compose)

### Phase E: Status Bar Integration
**Files**: `internal/tui/statusbar.go`, `internal/tui/app.go`

1. Add `composeActive bool` to `StatusBarModel`
2. Handle `ComposeActiveMsg` in `StatusBarModel.Update()`
3. Add compose-mode hint text: `"a: add  x: remove  Shift+↑↓: reorder  Enter: start  Esc: cancel"`
4. Forward `ComposeActiveMsg` from `AppModel.Update()` to `StatusBarModel`

### Phase F: CLI `wave compose` Command
**Files**: `cmd/wave/commands/compose.go`, `cmd/wave/commands/compose_test.go`

1. Create `NewComposeCmd() *cobra.Command`
   - `Use: "compose [pipelines...]"`
   - `Args: cobra.MinimumNArgs(2)`
   - `--validate-only` flag
   - `--manifest` flag (default "wave.yaml")
2. Load pipeline definitions for each named pipeline
3. Build `Sequence` and call `ValidateSequence()`
4. If `--validate-only`: print compatibility report and exit
5. If valid: show informational message that sequential execution requires #249
6. Register in root command

### Phase G: Edge Cases & Polish
**Files**: Various, based on edge case testing

1. `s` key has no effect on running/finished items or non-Pipelines views (FR-015)
2. Single-pipeline sequence delegates to normal `PipelineLauncher` (edge case)
3. Removing all pipelines keeps compose mode open with empty list, `Enter` disabled
4. Middle removal re-validates adjacent pipelines
5. Duplicate pipelines allowed with notice
6. No available pipelines to add → `a` disabled
7. Terminal too narrow → text-only degradation

## Complexity Tracking

No constitution violations. No complexity exceptions needed.
