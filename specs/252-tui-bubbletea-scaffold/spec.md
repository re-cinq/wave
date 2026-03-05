# Feature Specification: TUI Bubble Tea Scaffold

**Feature Branch**: `252-tui-bubbletea-scaffold`  
**Created**: 2026-03-05  
**Status**: Draft  
**Issue**: [#252](https://github.com/re-cinq/wave/issues/252) (part 1 of 10, parent: [#251](https://github.com/re-cinq/wave/issues/251))  
**Input**: Bootstrap the full-screen TUI application using Bubble Tea with the Elm architecture. Establish the foundational 3-row layout, wire TTY detection, add `--no-tui` flag, and implement graceful Ctrl-C handling.

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Launch TUI from Terminal (Priority: P1)

A developer opens their terminal in a Wave project directory and runs `wave` with no arguments. The application detects that stdout is a TTY and launches the full-screen Bubble Tea TUI instead of printing help text. They see a 3-row layout: a header bar at the top, a main content area filling the middle, and a status/keybindings bar at the bottom. All three rows display placeholder content appropriate to their role.

**Why this priority**: This is the foundational interaction — the TUI must launch correctly for any subsequent feature to work. Without this, there is no TUI.

**Independent Test**: Run `wave` in a terminal emulator and verify the full-screen TUI appears with 3 distinct rows. Verify it does not crash and renders within 100ms.

**Acceptance Scenarios**:

1. **Given** a Wave project directory with `wave.yaml`, **When** a user runs `wave` with no arguments in an interactive terminal, **Then** a full-screen TUI launches showing header bar, main content area, and status bar.
2. **Given** the TUI is launched, **When** the terminal window is at least 80×24, **Then** all three rows render correctly with placeholder content visible.
3. **Given** the TUI is launched, **When** the user presses `q`, **Then** the TUI exits cleanly and returns the user to their shell prompt.

---

### User Story 2 - Non-TTY Fallback to Help (Priority: P1)

A developer pipes `wave` output or runs it in a non-interactive context (e.g., `wave | cat`, cron job, CI pipeline). The application detects stdout is not a TTY and prints standard help text instead of launching the TUI.

**Why this priority**: Equally critical to P1-Story-1 — incorrect TTY detection would break scripting, CI, and piped workflows. The TUI must never launch in non-interactive contexts.

**Independent Test**: Run `wave | cat` and verify help text is printed to stdout. Run `wave --no-tui` and verify help text is printed.

**Acceptance Scenarios**:

1. **Given** stdout is not a TTY (e.g., piped), **When** a user runs `wave` with no arguments, **Then** standard Cobra help text is printed to stdout.
2. **Given** stdout is a TTY, **When** a user runs `wave --no-tui`, **Then** standard Cobra help text is printed instead of launching the TUI.
3. **Given** the environment variable `WAVE_FORCE_TTY=0` is set, **When** a user runs `wave`, **Then** help text is printed regardless of actual TTY status.

---

### User Story 3 - Graceful Shutdown with Ctrl-C (Priority: P2)

A developer is viewing the TUI and presses Ctrl-C. The application begins graceful shutdown — cleaning up any resources, saving state if needed, and displaying a brief exit status message. If the user presses Ctrl-C a second time before cleanup completes, the application force-exits immediately.

**Why this priority**: Graceful shutdown is essential for data safety and user trust, but depends on the TUI being launchable first (P1 stories).

**Independent Test**: Launch TUI, press Ctrl-C once and verify graceful exit message appears. Launch TUI, press Ctrl-C twice rapidly and verify immediate exit.

**Acceptance Scenarios**:

1. **Given** the TUI is running, **When** the user presses Ctrl-C once, **Then** the application initiates graceful shutdown, displays an exit status message, and exits with code 0.
2. **Given** a graceful shutdown is in progress, **When** the user presses Ctrl-C a second time, **Then** the application exits immediately with code 0 (user-initiated exit).
3. **Given** the TUI is running, **When** the user presses `q`, **Then** the application exits cleanly with code 0 (same behavior as single Ctrl-C).

---

### User Story 4 - Terminal Resize Handling (Priority: P3)

A developer resizes their terminal window while the TUI is running. The layout reflows to fill the new dimensions, maintaining the 3-row structure. If the terminal shrinks below the minimum supported size (80×24), the TUI degrades gracefully.

**Why this priority**: Resize handling is a polish feature — the TUI works at its initial size without it, but professional TUIs must handle resize.

**Independent Test**: Launch TUI, resize the terminal window and verify the layout reflows. Shrink below 80×24 and verify graceful degradation message.

**Acceptance Scenarios**:

1. **Given** the TUI is running at 120×40, **When** the terminal is resized to 100×30, **Then** all three rows reflow to fill the new dimensions.
2. **Given** the TUI is running, **When** the terminal is resized below 80×24, **Then** a graceful degradation message is shown (e.g., "Terminal too small. Minimum: 80×24").
3. **Given** the TUI is showing the degradation message, **When** the terminal is resized back above 80×24, **Then** the normal 3-row layout is restored.

---

### Edge Cases

- What happens when the terminal has exactly 80×24 dimensions? The layout should render correctly at the minimum size.
- What happens when `TERM=dumb` is set? The application should fall back to help text (non-ANSI terminal).
- What happens when `NO_COLOR` is set? The TUI should render without any color/styling but still function.
- What happens when the user runs `wave --no-tui run pipeline-name`? The `--no-tui` flag is only inspected by the root command's `RunE`; subcommands like `run` ignore it and use their own output modes.
- What happens when `wave` is run outside a git repository? The TUI should still launch (it doesn't depend on git), but the header placeholder may show reduced info.
- What happens when `wave.yaml` doesn't exist? The TUI should still launch and show an appropriate message in the main content area.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: The root `wave` command (no subcommand, no arguments) MUST launch the full-screen Bubble Tea TUI when stdout is detected as an interactive TTY. Explicit `--help`, `-h`, and `--version` flags MUST continue to print their respective text output without launching the TUI. This is implemented by adding a `RunE` handler to the root Cobra command.
- **FR-002**: The root `wave` command MUST print standard Cobra help text when stdout is NOT a TTY, or when the `--no-tui` flag is passed.
- **FR-003**: The TUI MUST render a 3-row layout: header bar (fixed height), main content area (fills remaining space), and status/keybindings bar (single line, fixed).
- **FR-004**: The header bar MUST display placeholder content (e.g., "Wave" branding text and placeholder metadata columns).
- **FR-005**: The main content area MUST display placeholder content (e.g., "Main content area — pipelines view coming soon").
- **FR-006**: The status bar MUST display the current view context label on the left and context-sensitive keybinding hints on the right.
- **FR-007**: The TUI MUST respond to terminal resize events (Bubble Tea's `tea.WindowSizeMsg`) and reflow the layout.
- **FR-008**: The TUI MUST show a degradation message when the terminal is smaller than 80 columns or 24 rows.
- **FR-009**: Pressing `q` MUST exit the TUI cleanly with exit code 0.
- **FR-010**: The first Ctrl-C signal MUST trigger graceful shutdown (cleanup and exit with status message, exit code 0).
- **FR-011**: A second Ctrl-C during graceful shutdown MUST force-exit immediately with exit code 0.
- **FR-012**: The `--no-tui` persistent flag MUST be registered on the root Cobra command. The flag is persistent so Cobra does not reject it on subcommands, but only the root command's `RunE` inspects it. Subcommands ignore it — they have their own output modes.
- **FR-013**: TTY detection for the root command MUST check **stdout** (not stdin) using `term.IsTerminal(int(os.Stdout.Fd()))`, consistent with the existing `display.isTerminal()` function and Bubble Tea's own requirements. The `WAVE_FORCE_TTY` env var override MUST be respected (values `1`/`true` force TUI, `0`/`false` force help text).
- **FR-014**: New Bubble Tea application code MUST be added to the existing `internal/tui/` package as new files (e.g., `app.go`, `header.go`, `content.go`, `statusbar.go`). The existing `internal/tui/` code (`run_selector.go`, `theme.go`, `pipelines.go`) MUST remain untouched and functional. The `WaveTheme()` from `theme.go` SHOULD be reused for consistent styling.
- **FR-015**: The TUI MUST render its first frame within 100ms of launch (measured from `tea.Program.Run()` to first `View()` render). If initialization takes longer, a spinner MUST be shown.
- **FR-016**: The Bubble Tea, Bubbles, and Lip Gloss dependencies MUST be available in `go.mod` (they already are: `bubbletea v1.3.10`, `bubbles v0.21.1`, `lipgloss v1.1.0`).

### Key Entities

- **AppModel**: The root Bubble Tea model implementing `tea.Model` (Init/Update/View). Contains the current window size, layout dimensions, shutdown state, and child component models for each row. Located in `internal/tui/app.go`.
- **HeaderModel**: Component model for the header bar row. Renders placeholder branding and metadata. Located in `internal/tui/header.go`.
- **ContentModel**: Component model for the main content area. Renders placeholder text; designed to be replaced by real views in subsequent issues. Located in `internal/tui/content.go`.
- **StatusBarModel**: Component model for the status/keybindings bar. Shows current context and available key actions. Located in `internal/tui/statusbar.go`.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `wave` with no args in a TTY launches a full-screen TUI that renders 3 distinct rows — verifiable by visual inspection and automated test.
- **SC-002**: `wave | cat` prints help text to stdout — verifiable by piping output and checking for Cobra help format.
- **SC-003**: `wave --no-tui` prints help text — verifiable by flag test.
- **SC-004**: The TUI first frame renders within 100ms of launch — measurable by timing test or manual stopwatch.
- **SC-005**: Terminal resize from 120×40 to 80×24 results in correct reflow — verifiable by resize test.
- **SC-006**: Terminals smaller than 80×24 show degradation message instead of broken layout — verifiable by resize test.
- **SC-007**: Single Ctrl-C triggers graceful exit (exit code 0) — verifiable by signal test.
- **SC-008**: Double Ctrl-C triggers immediate exit (exit code 0) — verifiable by signal test.
- **SC-009**: `q` key exits the TUI cleanly (exit code 0) — verifiable by key input test.
- **SC-010**: All existing `go test ./...` tests continue to pass after the change — verifiable by CI.
- **SC-011**: New `internal/tui/` package has test coverage for model initialization, update handling, and view rendering.
- **SC-012**: No new dependencies beyond what is already in `go.mod` — verifiable by diffing `go.mod`.

## Clarifications

The following ambiguities were identified during spec review and resolved based on codebase context:

### C-001: TTY Detection — stdout vs stdin

**Ambiguity**: FR-001 referenced "stdout is a TTY" but FR-013 originally referenced `isInteractive()` which checks `os.Stdin.Fd()` (see `cmd/wave/commands/run.go:457`). The `display` package's `isTerminal()` checks `os.Stdout.Fd()` (see `internal/display/terminal.go:105`).

**Resolution**: The TUI must check **stdout**, not stdin. Bubble Tea renders to stdout, so stdout must be a terminal for the TUI to function. The existing `isInteractive()` in `run.go` checks stdin because it guards interactive *input* (huh forms). The new root command TUI guard should use stdout detection, consistent with `display.isTerminal()`. FR-013 has been updated accordingly.

**Rationale**: Checking stdout aligns with Bubble Tea's requirements, the existing `display` package pattern, and standard TUI behavior. A separate `shouldLaunchTUI()` function in `cmd/wave/main.go` can encapsulate this logic to avoid conflating it with the existing `isInteractive()`.

### C-002: `--no-tui` Flag Scope with Subcommands

**Ambiguity**: FR-012 specified `--no-tui` as a persistent flag, but edge case 4 said it should be "ignored for subcommands." In Cobra, persistent flags propagate to all child commands, which could cause confusion.

**Resolution**: The flag is persistent so Cobra accepts it on any command without erroring, but only the root `RunE` handler inspects it. Subcommands already have their own output modes (`--output auto|json|text|quiet`) and never check `--no-tui`. FR-012 and edge case 4 have been clarified.

**Rationale**: Making it non-persistent would cause `wave --no-tui run ...` to error, which is a poor UX. Persistent + root-only-inspection is the standard Cobra pattern for flags that modify default behavior.

### C-003: File Structure in Existing `internal/tui/` Package

**Ambiguity**: FR-014 said "create or extend" `internal/tui/` but the existing package contains `run_selector.go`, `theme.go`, and `pipelines.go` using `huh` forms — not Bubble Tea full-screen. The file layout for new code was unspecified.

**Resolution**: New Bubble Tea app code goes in the same `internal/tui/` package as new files: `app.go` (AppModel), `header.go` (HeaderModel), `content.go` (ContentModel), `statusbar.go` (StatusBarModel). Existing files are not modified. The `WaveTheme()` from `theme.go` can be adapted for Lip Gloss styles in the full-screen TUI since it already defines Wave's color palette (cyan primary, white text, gray muted).

**Rationale**: Single package avoids import cycles and keeps TUI code cohesive. Separate files per model follows Go convention and makes the codebase navigable for the remaining 9 issues in the series.

### C-004: Exit Codes for All User-Initiated Exits

**Ambiguity**: Story 3 specified exit code 0 for single Ctrl-C but left double Ctrl-C and `q` unspecified. Unix convention uses exit code 130 for SIGINT, which would conflict.

**Resolution**: All user-initiated exits (`q`, single Ctrl-C, double Ctrl-C) return exit code 0. The TUI intercepts SIGINT via Bubble Tea's built-in signal handling rather than letting Go's default handler terminate with 130. This is consistent with how interactive TUI applications (vim, htop, lazygit) behave — the user is *requesting* exit, not encountering an error. SC-007, SC-008, SC-009, and the acceptance scenarios have been updated to be explicit.

**Rationale**: Exit code 0 for intentional user exit is standard TUI practice. Shell scripts that invoke `wave` (non-TTY) get help text, not the TUI, so there is no conflict with signal conventions in scripted contexts.

### C-005: Root Command Behavior Change — Help and Version Flags

**Ambiguity**: Currently the root command has no `RunE` and falls through to Cobra help. Adding `RunE` to launch the TUI could break `wave --help` and `wave --version`.

**Resolution**: FR-001 has been updated to explicitly state that `--help`, `-h`, and `--version` continue to work normally. Cobra handles help and version flags *before* invoking `RunE`, so no special logic is needed — the TUI launch only triggers when `RunE` is actually reached (i.e., bare `wave` with no help/version flags).

**Rationale**: This is standard Cobra behavior. Help and version flags are intercepted by Cobra's built-in preprocessing, so `RunE` is never called when these flags are present. No special handling required in the implementation.
