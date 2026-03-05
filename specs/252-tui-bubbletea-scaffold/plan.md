# Implementation Plan: TUI Bubble Tea Scaffold

**Branch**: `252-tui-bubbletea-scaffold` | **Date**: 2026-03-05 | **Spec**: [specs/252-tui-bubbletea-scaffold/spec.md](spec.md)
**Input**: Feature specification from `/specs/252-tui-bubbletea-scaffold/spec.md`

## Summary

Bootstrap the full-screen Bubble Tea TUI application with a 3-row layout (header, content, status bar), wire TTY detection on the root `wave` command, add `--no-tui` flag, and implement graceful Ctrl-C / `q` exit handling. This is part 1 of 10 for the complete TUI implementation (parent: #251).

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea v1.3.10`, `charmbracelet/lipgloss v1.1.0`, `spf13/cobra v1.8.0`, `golang.org/x/term v0.39.0` — all already in `go.mod`
**Storage**: N/A (no persistence in this feature)
**Testing**: `go test ./...` with `testify/assert`
**Target Platform**: Linux, macOS (terminal emulators)
**Project Type**: Single Go binary (existing)
**Performance Goals**: First frame render within 100ms of `tea.Program.Run()`
**Constraints**: No new dependencies; existing tests must continue passing; existing `internal/tui/` code untouched
**Scale/Scope**: 4 new source files + 4 test files in `internal/tui/`, 1 modified file in `cmd/wave/`

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies; all Go deps compile into binary |
| P2: Manifest as SSOT | ✅ Pass | TUI doesn't add configuration; reads existing manifest only when needed |
| P3: Persona-Scoped Execution | ✅ N/A | No persona/agent changes |
| P4: Fresh Memory at Boundaries | ✅ N/A | No pipeline step changes |
| P5: Navigator-First | ✅ N/A | No pipeline changes |
| P6: Contracts at Handovers | ✅ N/A | No pipeline changes |
| P7: Relay via Summarizer | ✅ N/A | No relay changes |
| P8: Ephemeral Workspaces | ✅ N/A | No workspace changes |
| P9: Credentials Never Touch Disk | ✅ Pass | TUI displays no credentials |
| P10: Observable Progress | ✅ Pass | TUI is the future home of progress visualization |
| P11: Bounded Recursion | ✅ N/A | No meta-pipeline changes |
| P12: Minimal Step State Machine | ✅ N/A | No state machine changes |
| P13: Test Ownership | ✅ Pass | New code comes with tests; existing tests must pass |

**Result**: All applicable principles pass. No violations requiring justification.

## Project Structure

### Documentation (this feature)

```
specs/252-tui-bubbletea-scaffold/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
├── checklists/
│   └── requirements.md  # Requirements checklist
└── tasks.md             # Phase 2 output (not created by plan)
```

### Source Code (repository root)

```
cmd/wave/
├── main.go              # MODIFIED: Add RunE, --no-tui flag, shouldLaunchTUI()
└── main_test.go         # NEW: Tests for shouldLaunchTUI, --no-tui flag

internal/tui/
├── theme.go             # EXISTING (unchanged) — color constants reference
├── run_selector.go      # EXISTING (unchanged) — huh-based pipeline selector
├── pipelines.go         # EXISTING (unchanged) — pipeline discovery
├── app.go               # NEW: AppModel — root tea.Model
├── app_test.go          # NEW: Tests for AppModel
├── header.go            # NEW: HeaderModel component
├── header_test.go       # NEW: Tests for HeaderModel
├── content.go           # NEW: ContentModel component
├── content_test.go      # NEW: Tests for ContentModel
├── statusbar.go         # NEW: StatusBarModel component
└── statusbar_test.go    # NEW: Tests for StatusBarModel
```

**Structure Decision**: All new code fits within the existing `internal/tui/` package and `cmd/wave/` directory. No new packages needed. The 4 new component files follow one-type-per-file Go convention.

## Implementation Plan

### Phase A: Component Models (`internal/tui/`)

**Task A1**: Create `internal/tui/app.go` — AppModel

The root Bubble Tea model. Implements `tea.Model` with:
- `Init()` → returns `nil`
- `Update(msg)` → handles `tea.KeyMsg` (q, Ctrl-C), `tea.WindowSizeMsg`, propagates to children
- `View()` → composes 3-row layout with Lip Gloss; shows degradation message for terminals < 80×24
- Shutdown state tracking for double Ctrl-C
- `NewAppModel()` constructor
- `RunTUI() error` — public entry point that creates `tea.NewProgram` with `tea.WithAltScreen()` and runs it

Key design points:
- Layout: header (fixed 3 lines) + content (fills remaining) + status bar (fixed 1 line)
- Uses `lipgloss.JoinVertical` for row composition
- `ready` flag: false until first `WindowSizeMsg` received (Bubble Tea convention)
- Color constants: cyan (`lipgloss.Color("6")`), white (`lipgloss.Color("7")`), muted (`lipgloss.Color("244")`) — matching `theme.go`

**Task A2**: Create `internal/tui/header.go` — HeaderModel

Header bar component:
- `SetWidth(w int)` for reflow
- `View()` renders: Wave ASCII logo (compact inline) + "Pipeline Orchestrator" + placeholder status
- Uses Lip Gloss horizontal join for logo + text columns
- Fixed 3-line height

**Task A3**: Create `internal/tui/content.go` — ContentModel

Main content area:
- `SetSize(w, h int)` for reflow
- `View()` renders centered placeholder: "Wave TUI — Pipelines view coming soon"
- Uses `lipgloss.Place()` for centering within the content area dimensions

**Task A4**: Create `internal/tui/statusbar.go` — StatusBarModel

Status/keybindings bar:
- `SetWidth(w int)` for reflow
- `View()` renders: context label "Dashboard" (left-aligned) + "q: quit  ctrl+c: exit" (right-aligned)
- Single line, full width, uses Lip Gloss for left/right alignment
- Background color for visual separation (muted gray)

### Phase B: Root Command Integration (`cmd/wave/`)

**Task B1**: Modify `cmd/wave/main.go`

1. Add `--no-tui` persistent flag in `init()`:
   ```go
   rootCmd.PersistentFlags().Bool("no-tui", false, "Disable TUI and print help text")
   ```

2. Add `shouldLaunchTUI(cmd *cobra.Command) bool`:
   - Check `--no-tui` flag
   - Check `WAVE_FORCE_TTY` env var (`"1"`/`"true"` → force true, `"0"`/`"false"` → force false)
   - Check `term.IsTerminal(int(os.Stdout.Fd()))` for stdout TTY
   - Check `TERM` env var (reject `"dumb"`)

3. Add `RunE` to `rootCmd`:
   ```go
   RunE: func(cmd *cobra.Command, args []string) error {
       if shouldLaunchTUI(cmd) {
           return tui.RunTUI()
       }
       return cmd.Help()
   },
   ```

### Phase C: Tests

**Task C1**: `internal/tui/app_test.go`
- Test `NewAppModel()` returns valid initial state
- Test `Update` with `tea.WindowSizeMsg` sets dimensions and `ready` flag
- Test `Update` with `tea.KeyMsg` "q" returns `tea.Quit`
- Test `Update` with `tea.KeyMsg` Ctrl-C sets `shuttingDown` and returns `tea.Quit`
- Test `View()` before ready returns loading indicator
- Test `View()` after ready returns composed 3-row layout
- Test `View()` with dimensions < 80×24 returns degradation message

**Task C2**: `internal/tui/header_test.go`
- Test `View()` contains Wave branding text
- Test `SetWidth()` updates width
- Test output respects width boundaries

**Task C3**: `internal/tui/content_test.go`
- Test `View()` contains placeholder text
- Test `SetSize()` updates dimensions
- Test output dimensions match set size

**Task C4**: `internal/tui/statusbar_test.go`
- Test `View()` contains keybinding hints (q, ctrl+c)
- Test `View()` contains context label
- Test `SetWidth()` updates width

**Task C5**: `cmd/wave/main_test.go`
- Test `shouldLaunchTUI` with `--no-tui` flag returns false
- Test `shouldLaunchTUI` with `WAVE_FORCE_TTY=1` returns true
- Test `shouldLaunchTUI` with `WAVE_FORCE_TTY=0` returns false
- Test `shouldLaunchTUI` with `TERM=dumb` returns false
- Test `--no-tui` flag is registered as persistent

### Phase D: Validation

**Task D1**: Run full test suite
- `go test ./...` — all existing tests must pass
- `go vet ./...` — no static analysis issues
- Verify no changes to `go.mod` beyond what's already present (SC-012)

## Complexity Tracking

No constitution violations. No complexity justifications needed.
