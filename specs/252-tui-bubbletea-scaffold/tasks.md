# Tasks: TUI Bubble Tea Scaffold (#252)

**Feature**: `252-tui-bubbletea-scaffold` (part 1 of 10, parent: #251)
**Generated**: 2026-03-05
**Total Tasks**: 16
**Parallel Opportunities**: 6

## Phase 1: Setup

- [X] T001 [P1] [Setup] Verify Bubble Tea dependencies exist in `go.mod` and run `go mod tidy` — confirm `bubbletea v1.3.10`, `lipgloss v1.1.0`, `bubbles v0.21.1`, `golang.org/x/term` are present. No new deps allowed (SC-012). File: `go.mod`

## Phase 2: Foundational — Component Models (Story 1 prerequisite)

These tasks create the individual TUI components. T002–T004 can be done in parallel; T005 depends on all three.

- [X] T002 [P1] [Story1] [P] Create `HeaderModel` in `internal/tui/header.go` — struct with `width int`, `SetWidth(w int)`, `View() string` rendering Wave logo (reuse `WaveLogo()` from `theme.go` or inline matching style), "Pipeline Orchestrator" text, and placeholder status. Fixed 3-line height. Use `lipgloss.Color("6")` (cyan), `lipgloss.Color("7")` (white), `lipgloss.Color("244")` (muted) matching existing theme. File: `internal/tui/header.go`
- [X] T003 [P1] [Story1] [P] Create `ContentModel` in `internal/tui/content.go` — struct with `width int`, `height int`, `SetSize(w, h int)`, `View() string` rendering centered placeholder text "Wave TUI — Pipelines view coming soon" using `lipgloss.Place()`. File: `internal/tui/content.go`
- [X] T004 [P1] [Story1] [P] Create `StatusBarModel` in `internal/tui/statusbar.go` — struct with `width int`, `contextLabel string` (default "Dashboard"), `SetWidth(w int)`, `View() string` rendering context label left-aligned and keybinding hints "q: quit  ctrl+c: exit" right-aligned on a single line with muted background. File: `internal/tui/statusbar.go`
- [X] T005 [P1] [Story1] Create `AppModel` in `internal/tui/app.go` — root `tea.Model` composing `HeaderModel`, `ContentModel`, `StatusBarModel`. Fields: `width`, `height`, `header`, `content`, `statusBar`, `shuttingDown bool`, `ready bool`. Implement `NewAppModel()` constructor, `Init() tea.Cmd` (returns nil), `Update(msg tea.Msg) (tea.Model, tea.Cmd)` handling `tea.WindowSizeMsg`/`tea.KeyMsg`, `View() string` using `lipgloss.JoinVertical`. Layout: header (3 lines) + content (fills remaining) + statusbar (1 line). Show "Initializing..." before first `WindowSizeMsg`. Add public `RunTUI() error` that creates `tea.NewProgram(NewAppModel(), tea.WithAltScreen())` and calls `Run()`. File: `internal/tui/app.go`

## Phase 3: Story 1 — Launch TUI from Terminal (P1)

- [X] T006 [P1] [Story1] Add `shouldLaunchTUI(cmd *cobra.Command) bool` to `cmd/wave/main.go` — checks in order: (1) `--no-tui` flag → false, (2) `WAVE_FORCE_TTY` env var (`"1"`/`"true"` → true, `"0"`/`"false"` → false), (3) `TERM=dumb` → false, (4) `term.IsTerminal(int(os.Stdout.Fd()))`. Import `golang.org/x/term` and `github.com/recinq/wave/internal/tui`. File: `cmd/wave/main.go`
- [X] T007 [P1] [Story1] Add `RunE` to `rootCmd` in `cmd/wave/main.go` — calls `shouldLaunchTUI(cmd)`, if true returns `tui.RunTUI()`, else returns `cmd.Help()`. File: `cmd/wave/main.go`

## Phase 4: Story 2 — Non-TTY Fallback & `--no-tui` Flag (P1)

- [X] T008 [P1] [Story2] Register `--no-tui` persistent flag on `rootCmd` in `init()` — `rootCmd.PersistentFlags().Bool("no-tui", false, "Disable TUI and print help text")`. Must be persistent so Cobra doesn't reject `wave --no-tui run ...`. Only root `RunE` inspects it. File: `cmd/wave/main.go`

## Phase 5: Story 3 — Graceful Shutdown (P2)

- [X] T009 [P2] [Story3] Implement Ctrl-C and `q` key handling in `AppModel.Update` — on `tea.KeyMsg` type `tea.KeyCtrlC`: if `shuttingDown` is already true, call `os.Exit(0)` (force exit); else set `shuttingDown = true` and return `tea.Quit`. On key "q": return `tea.Quit`. Both exit with code 0. File: `internal/tui/app.go`

## Phase 6: Story 4 — Terminal Resize Handling (P3)

- [X] T010 [P3] [Story4] Implement resize handling and degradation in `AppModel` — on `tea.WindowSizeMsg`, update `width`/`height`, set `ready = true`, propagate via `SetWidth`/`SetSize` to children. In `View()`, if `width < 80 || height < 24`, render degradation message: "Terminal too small. Minimum: 80×24. Current: {w}×{h}". File: `internal/tui/app.go`

## Phase 7: Tests

T011–T013 can be done in parallel. T014 depends on T005+T009+T010. T015 depends on T006–T008.

- [X] T011 [P1] [Story1] [P] Create `internal/tui/header_test.go` — table-driven tests: `View()` contains Wave branding text, `SetWidth()` updates width, output respects width boundaries. Use `testify/assert`. File: `internal/tui/header_test.go`
- [X] T012 [P1] [Story1] [P] Create `internal/tui/content_test.go` — table-driven tests: `View()` contains placeholder text "Pipelines view coming soon", `SetSize()` updates dimensions. Use `testify/assert`. File: `internal/tui/content_test.go`
- [X] T013 [P1] [Story1] [P] Create `internal/tui/statusbar_test.go` — table-driven tests: `View()` contains keybinding hints ("q: quit", "ctrl+c: exit"), `View()` contains context label "Dashboard", `SetWidth()` updates width. Use `testify/assert`. File: `internal/tui/statusbar_test.go`
- [X] T014 [P1] [Story1-3] Create `internal/tui/app_test.go` — tests: `NewAppModel()` returns valid initial state (ready=false, shuttingDown=false), `Update` with `tea.WindowSizeMsg` sets dimensions and `ready=true`, `Update` with `tea.KeyMsg` "q" returns `tea.Quit`, `Update` with Ctrl-C sets `shuttingDown` and returns `tea.Quit`, `View()` before ready returns loading text, `View()` after ready returns composed layout containing header/content/statusbar text, `View()` with dimensions < 80×24 returns degradation message. File: `internal/tui/app_test.go`
- [X] T015 [P1] [Story1-2] Create `cmd/wave/main_test.go` — tests: `shouldLaunchTUI` with `--no-tui=true` returns false, with `WAVE_FORCE_TTY=1` returns true, with `WAVE_FORCE_TTY=0` returns false, with `TERM=dumb` returns false, `--no-tui` flag is registered as persistent. Use `testify/assert`, `t.Setenv()` for env vars. File: `cmd/wave/main_test.go`

## Phase 8: Validation & Polish

- [X] T016 [P1] [Validation] Run `go test ./...` and `go vet ./...` — all existing and new tests must pass. Verify no changes to `go.mod` beyond what's already present (SC-012). Verify existing `internal/tui/` tests still pass (run_selector_test.go, pipelines_test.go). File: N/A (validation step)

## Dependency Graph

```
T001 (setup)
  └─► T002, T003, T004 (components, parallel)
        └─► T005 (AppModel, depends on all components)
              ├─► T006 (shouldLaunchTUI)
              │     └─► T007 (RunE on rootCmd)
              │           └─► T008 (--no-tui flag, depends on T007 for integration)
              ├─► T009 (shutdown handling, depends on T005)
              └─► T010 (resize handling, depends on T005)
  T011, T012, T013 (component tests, parallel after T002-T004)
  T014 (app test, after T005+T009+T010)
  T015 (main_test, after T006-T008)
  T016 (validation, after all)
```

## Story-to-Task Mapping

| Story | Priority | Tasks |
|-------|----------|-------|
| Story 1: Launch TUI | P1 | T002, T003, T004, T005, T006, T007, T011, T012, T013, T014 |
| Story 2: Non-TTY Fallback | P1 | T008, T015 |
| Story 3: Graceful Shutdown | P2 | T009, T014 |
| Story 4: Terminal Resize | P3 | T010, T014 |
| Cross-cutting | — | T001, T016 |
