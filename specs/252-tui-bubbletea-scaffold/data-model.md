# Data Model: TUI Bubble Tea Scaffold

**Feature**: #252 — Bubble Tea TUI scaffold (part 1 of 10, parent: #251)
**Date**: 2026-03-05

## Entities

### AppModel (`internal/tui/app.go`)

The root Bubble Tea model implementing `tea.Model`. Owns all application state and child component models.

```go
type AppModel struct {
    // Window dimensions (updated via tea.WindowSizeMsg)
    width  int
    height int

    // Child component models
    header    HeaderModel
    content   ContentModel
    statusBar StatusBarModel

    // Shutdown state
    shuttingDown bool

    // Layout constants
    ready bool // true after first WindowSizeMsg received
}
```

**Implements**: `tea.Model` (Init, Update, View)

**Lifecycle**:
- `Init()` → returns `nil` (no startup commands; waits for `WindowSizeMsg`)
- `Update(msg)` → handles `tea.KeyMsg` (q, Ctrl-C), `tea.WindowSizeMsg`, delegates to children
- `View()` → composes header + content + statusbar using Lip Gloss vertical join

**Layout calculation in View()**:
- Header height: fixed (3 lines — logo + padding)
- Status bar height: fixed (1 line)
- Content height: `height - headerHeight - statusBarHeight`
- All widths: `width` (full terminal width)

### HeaderModel (`internal/tui/header.go`)

Component model for the header bar. Renders Wave branding and placeholder metadata.

```go
type HeaderModel struct {
    width int
}
```

**Methods**:
- `Update(msg tea.Msg)` → handles `tea.WindowSizeMsg` to update width
- `View() string` → renders header bar with Wave logo, version placeholder
- `SetWidth(w int)` → updates width for reflow

**Visual layout**:
```
╦ ╦╔═╗╦  ╦╔═╗  │  Pipeline Orchestrator    [no pipeline running]
║║║╠═╣╚╗╔╝║╣   │
╚╩╝╩ ╩ ╚╝ ╚═╝  │
```

### ContentModel (`internal/tui/content.go`)

Component model for the main content area. Displays placeholder text; will be replaced by real views in subsequent issues.

```go
type ContentModel struct {
    width  int
    height int
}
```

**Methods**:
- `Update(msg tea.Msg)` → handles `tea.WindowSizeMsg`
- `View() string` → renders centered placeholder text
- `SetSize(w, h int)` → updates dimensions for reflow

**Placeholder content**: Centered text: "Wave TUI — Pipelines view coming soon"

### StatusBarModel (`internal/tui/statusbar.go`)

Component model for the bottom status/keybindings bar.

```go
type StatusBarModel struct {
    width       int
    contextLabel string // e.g., "Dashboard"
}
```

**Methods**:
- `Update(msg tea.Msg)` → handles `tea.WindowSizeMsg`
- `View() string` → renders context label (left) + keybinding hints (right)
- `SetWidth(w int)` → updates width for reflow

**Visual layout**:
```
 Dashboard                                          q: quit  ctrl+c: exit
```

## Key Messages

| Message | Source | Handler | Effect |
|---------|--------|---------|--------|
| `tea.WindowSizeMsg` | Bubble Tea runtime | `AppModel.Update` | Updates dimensions, propagates to children |
| `tea.KeyMsg` "q" | User input | `AppModel.Update` | Returns `tea.Quit` |
| `tea.KeyMsg` Ctrl-C | User input / SIGINT | `AppModel.Update` | First: `shuttingDown=true` + `tea.Quit`; Second: `os.Exit(0)` |

## Functions (cmd/wave/main.go)

### shouldLaunchTUI()

```go
func shouldLaunchTUI(cmd *cobra.Command) bool
```

Determines whether to launch the Bubble Tea TUI. Returns `true` when:
1. `--no-tui` flag is NOT set
2. stdout is a TTY (via `term.IsTerminal(int(os.Stdout.Fd()))`)
3. `WAVE_FORCE_TTY` is not `"0"` or `"false"`
4. `TERM` is not `"dumb"`

If `WAVE_FORCE_TTY` is `"1"` or `"true"`, forces `true` regardless of actual TTY status.

### RunTUI()

```go
func RunTUI() error
```

Public function in `internal/tui/` package. Creates the Bubble Tea program with `tea.WithAltScreen()` and runs it. Called by `rootCmd.RunE`.

## File Map

| File | Purpose | New/Modified |
|------|---------|--------------|
| `cmd/wave/main.go` | Root command `RunE`, `--no-tui` flag, `shouldLaunchTUI()` | Modified |
| `internal/tui/app.go` | `AppModel` — root tea.Model | New |
| `internal/tui/header.go` | `HeaderModel` — header bar component | New |
| `internal/tui/content.go` | `ContentModel` — main area component | New |
| `internal/tui/statusbar.go` | `StatusBarModel` — status bar component | New |
| `internal/tui/app_test.go` | Tests for AppModel init/update/view | New |
| `internal/tui/header_test.go` | Tests for HeaderModel | New |
| `internal/tui/content_test.go` | Tests for ContentModel | New |
| `internal/tui/statusbar_test.go` | Tests for StatusBarModel | New |
| `cmd/wave/main_test.go` | Tests for shouldLaunchTUI, --no-tui flag | New |

## Dependencies

All dependencies already present in `go.mod`:
- `github.com/charmbracelet/bubbletea v1.3.10`
- `github.com/charmbracelet/lipgloss v1.1.0`
- `github.com/charmbracelet/bubbles v0.21.1` (indirect, for potential spinner use)
- `golang.org/x/term v0.39.0` (for TTY detection)
- `github.com/spf13/cobra v1.8.0` (existing CLI framework)

No new dependencies required (SC-012).
