# Research: TUI Bubble Tea Scaffold

**Feature**: #252 — Bubble Tea TUI scaffold (part 1 of 10, parent: #251)
**Date**: 2026-03-05

## Phase 0 — Unknowns & Research

### Unknown 1: Root Command Behavior Change

**Question**: How to add `RunE` to the root Cobra command without breaking `--help`, `--version`, and existing subcommands?

**Decision**: Add `RunE` to `rootCmd` in `cmd/wave/main.go`. Cobra processes `--help`/`-h` and `--version` flags *before* calling `RunE`, so the TUI only launches when `RunE` is actually reached (bare `wave` with no flags). Subcommands have their own `RunE` and are unaffected.

**Rationale**: This is standard Cobra behavior. The `--help` and `--version` flags are intercepted by Cobra's built-in preprocessing. When any subcommand is invoked (e.g., `wave run`), the root `RunE` is not called. This is confirmed by reading `cmd/wave/main.go:17-28` — the root command currently has no `RunE`, so adding one only activates for the bare `wave` invocation.

**Alternatives rejected**:
- Wrapper function around `rootCmd.Execute()` — adds complexity, breaks Cobra conventions
- Custom `PersistentPreRunE` — would run for all subcommands, requiring guard logic

### Unknown 2: TTY Detection — Reuse vs. New Function

**Question**: Should the TUI reuse `display.isTerminal()` (unexported) or `isInteractive()` from `run.go` (checks stdin)?

**Decision**: Create a new `shouldLaunchTUI()` function in `cmd/wave/main.go` that checks **stdout** using `term.IsTerminal(int(os.Stdout.Fd()))` and respects `WAVE_FORCE_TTY`. Also check `TERM=dumb` to fall back (per edge case).

**Rationale**: 
- `display.isTerminal()` checks stdout but is unexported and lives in the `display` package
- `isInteractive()` in `run.go` checks **stdin** (for huh form input) — wrong check for TUI rendering
- Bubble Tea renders to stdout, so stdout must be the terminal. A new function in `main.go` keeps it co-located with the root command
- The `WAVE_FORCE_TTY` env var override is already established pattern (see `display/terminal.go:101-105` and `run.go:457-461`)

**Alternatives rejected**:
- Exporting `display.isTerminal()` — would add coupling between cmd and display packages for one function
- Reusing `isInteractive()` — checks stdin, not stdout; different semantic meaning

### Unknown 3: Bubble Tea Model Architecture — Flat vs. Nested

**Question**: Should the root model contain all state directly, or compose child models?

**Decision**: Compose child models. `AppModel` owns `HeaderModel`, `ContentModel`, and `StatusBarModel` as struct fields. Each child implements its own `Update()` and `View()` methods (not `tea.Model` — they're internal components, not standalone programs). `AppModel` delegates to children in its `Update` and `View`.

**Rationale**: The spec explicitly defines 4 entities (AppModel, HeaderModel, ContentModel, StatusBarModel) in separate files. Composition is idiomatic Bubble Tea for multi-region layouts. Future issues (#251 series) will replace `ContentModel`'s placeholder with real views — a composed architecture makes this a drop-in replacement without touching `AppModel`.

**Alternatives rejected**:
- Flat model with all view logic in one file — violates single-responsibility, makes future issues harder
- Full `tea.Model` interface on children — over-abstraction for components that don't run as standalone programs

### Unknown 4: Graceful Shutdown & Double Ctrl-C

**Question**: How to implement double Ctrl-C for force-exit in Bubble Tea?

**Decision**: Track shutdown state in `AppModel` (`shuttingDown bool`). On first `tea.KeyCtrlC`, set `shuttingDown = true` and return `tea.Quit`. On second `tea.KeyCtrlC` while `shuttingDown` is true, call `os.Exit(0)`. In practice, Bubble Tea's `tea.Quit` is fast enough that the second Ctrl-C scenario is mainly a safety net.

**Rationale**: Bubble Tea intercepts SIGINT and converts it to `tea.KeyCtrlC` messages. The standard pattern is to return `tea.Quit` which triggers the alternate screen restore and cleanup. For the double-Ctrl-C case, since `tea.Quit` is near-instant for our simple scaffold, the force exit is mostly a UX guarantee. Using `os.Exit(0)` on the second signal is the standard TUI pattern (vim, lazygit do this).

**Alternatives rejected**:
- Custom signal handler outside Bubble Tea — conflicts with Bubble Tea's own signal handling
- `context.WithCancel` pattern — over-engineering for a UI exit path

### Unknown 5: Existing `internal/tui/` Package Compatibility

**Question**: Will new Bubble Tea full-screen app code conflict with existing `huh`-based code in the same package?

**Decision**: No conflict. The existing code (`run_selector.go`, `pipelines.go`, `theme.go`) uses `charmbracelet/huh` for form-based prompts. The new code uses `charmbracelet/bubbletea` for a full-screen app. Both are in the same `tui` package but have no type name collisions. The `WaveTheme()` from `theme.go` is a `*huh.Theme` (not directly usable for Lip Gloss), but the color constants (cyan=6, white=7, muted=244) should be extracted into shared constants or simply duplicated as Lip Gloss styles.

**Rationale**: Reading `theme.go`, the color values are hardcoded as Lip Gloss colors (`lipgloss.Color("6")`, etc.). The new Bubble Tea code can reference the same color values directly. No shared state, no import cycles, no naming conflicts.

### Unknown 6: `--no-tui` Flag Registration

**Question**: Where to register the `--no-tui` persistent flag?

**Decision**: Register on `rootCmd` in `cmd/wave/main.go` `init()`, alongside existing persistent flags like `--manifest`, `--debug`, `--output`. Being persistent means Cobra won't reject `wave --no-tui run pipeline` — the flag propagates but only the root `RunE` inspects it.

**Rationale**: This is the established pattern in `main.go:34-37`. Persistent flags on root propagate to all subcommands. The `run` command already has its own output modes and never checks `--no-tui`.

## Summary of Decisions

| Decision | Choice | Key Factor |
|----------|--------|------------|
| Root command change | Add `RunE` to `rootCmd` | Standard Cobra pattern; help/version unaffected |
| TTY detection | New `shouldLaunchTUI()` checking stdout | Bubble Tea needs stdout; different from stdin check |
| Model architecture | Composed child models | Future extensibility for 9 remaining issues |
| Shutdown handling | Track state, `tea.Quit` + `os.Exit(0)` | Standard Bubble Tea pattern |
| Package structure | New files in existing `internal/tui/` | No conflicts with huh-based code |
| `--no-tui` flag | Persistent flag on root command | Cobra propagation prevents subcommand errors |
| Color constants | Reuse same lipgloss.Color values from theme.go | Consistent design language |
