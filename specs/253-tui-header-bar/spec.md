# Feature Specification: TUI Header Bar with Animated Logo and Project Metadata

**Feature Branch**: `253-tui-header-bar`  
**Created**: 2026-03-05  
**Status**: Draft  
**Input**: GitHub Issue #253 — part of #251 (full-screen TUI implementation)

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Static Header with Project Metadata (Priority: P1)

A developer launches the Wave TUI and immediately sees the header bar displaying the Wave ASCII logo on the left and key project metadata on the right — health status, repository name, current git branch, clean/dirty state, and latest commit hash. This gives at-a-glance project awareness without leaving the TUI.

**Why this priority**: The static header with metadata is the core value proposition. Without it, the header is just branding with no utility. Every other story builds on this foundation.

**Independent Test**: Can be fully tested by launching the TUI in a git repository and verifying that the header renders correct metadata. Delivers immediate value as a project status dashboard.

**Acceptance Scenarios**:

1. **Given** the TUI is launched in a git repository, **When** the header renders, **Then** it displays the Wave ASCII logo on the left side and project metadata columns on the right including: health status indicator, GitHub repository name (from manifest or git remote), current git branch name, git remote name, clean/dirty working tree indicator, open issues count, and abbreviated latest commit hash.
2. **Given** the TUI is launched in a directory with a valid `wave.yaml`, **When** the header reads project metadata, **Then** it displays the project name from the manifest metadata section.
3. **Given** the terminal width is 80 columns (minimum supported), **When** the header renders, **Then** all metadata columns fit without truncation or overlap, gracefully omitting lower-priority columns if space is insufficient.

---

### User Story 2 - Animated Logo During Pipeline Execution (Priority: P2)

When one or more pipelines are actively running, the Wave ASCII logo in the header animates by cycling the foreground color through a palette on a timer tick. When all pipelines finish, the animation stops and the logo returns to its static cyan color. This provides a subtle, always-visible indicator of system activity.

**Why this priority**: Logo animation is the primary visual signal that work is happening. It differentiates idle from active states without requiring the user to inspect pipeline details.

**Independent Test**: Can be tested by starting a pipeline run and observing the logo color cycling, then verifying it stops when no pipelines are running.

**Acceptance Scenarios**:

1. **Given** no pipelines are running (`runningCount == 0`), **When** the header renders, **Then** the Wave logo is displayed in its static cyan color with no animation.
2. **Given** one or more pipelines are running (`runningCount > 0`), **When** the header renders on each tick, **Then** the logo cycles its foreground color through a palette (cyan → blue → magenta → cyan) at a fixed 200ms interval.
3. **Given** all running pipelines complete, **When** the running count drops to zero, **Then** the logo color animation stops and the logo returns to static cyan within one tick interval.

---

### User Story 3 - Dynamic Branch Display on Pipeline Selection (Priority: P2)

When the user selects a finished pipeline in the main content area, the header bar's branch display updates to show the branch that pipeline ran on (its worktree branch), rather than the current checked-out branch. This helps the user understand which branch contains a pipeline's output.

**Why this priority**: Dynamic branch display is a key interaction between the header and the content area. It provides critical context when reviewing finished pipeline results and deciding whether to checkout or merge.

**Independent Test**: Can be tested by selecting a finished pipeline in the pipeline list and verifying the header's branch field updates to show that pipeline's worktree branch, then selecting a different item and confirming it reverts.

**Acceptance Scenarios**:

1. **Given** a finished pipeline is selected in the left pane, **When** the selection changes, **Then** the header's branch display updates to show the branch name from that pipeline's worktree (e.g., `wave/speckit-flow-abc123`).
2. **Given** no pipeline or a running/available pipeline is selected, **When** the header renders, **Then** the branch display shows the current repository's checked-out branch (e.g., `main`).
3. **Given** the user navigates away from the finished pipeline selection, **When** focus returns to the available pipelines section, **Then** the branch display reverts to the repository's current branch.

---

### User Story 4 - NO_COLOR and Accessibility Compliance (Priority: P3)

When the `NO_COLOR` environment variable is set, all header bar styling (colors, bold) is disabled, rendering plain text that works in any terminal. The header also adapts to narrow terminals by gracefully degrading metadata display.

**Why this priority**: Accessibility and terminal compatibility are important for CI environments, screen readers, and constrained terminal sessions, but most users will see the styled version.

**Independent Test**: Can be tested by setting `NO_COLOR=1` and launching the TUI, verifying no ANSI escape codes appear in the header output.

**Acceptance Scenarios**:

1. **Given** `NO_COLOR` environment variable is set, **When** the header renders, **Then** no ANSI color or styling escape sequences are present in the output.
2. **Given** the terminal width is less than the space needed for all metadata columns, **When** the header renders, **Then** lower-priority metadata columns are progressively hidden (commit hash first, then issues count, then remote info) while preserving the logo and branch display.
3. **Given** a terminal width of exactly 80 columns, **When** the header renders, **Then** the logo and at minimum the branch name and health status are visible.

---

### Edge Cases

- What happens when git is not available or the directory is not a git repository? The header should display "[no git]" for branch and commit fields.
- What happens when `wave.yaml` is missing or malformed? The header should display "[no project]" for project-related fields and continue operating.
- What happens when the GitHub API is unreachable? The header should display cached or "[offline]" values without blocking the TUI startup.
- What happens when the terminal is resized mid-render? The header should reflow metadata columns based on the new width on the next render cycle.
- What happens when animation tick fires but the TUI is shutting down? The tick should be ignored and no further ticks scheduled.
- What happens when a finished pipeline's worktree branch has been deleted? The header should display the branch name with a visual indicator that the branch no longer exists (e.g., strikethrough or "[deleted]" suffix).
- What happens when GitHub auth is not configured (no `gh` CLI or `GITHUB_TOKEN`)? The header should display "—" for issues count and not attempt GitHub API calls. This is distinct from "unreachable" (auth exists but network fails → "[offline]").

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: Header bar MUST render at a fixed height of 3 lines across the full terminal width at the top of the TUI layout.
- **FR-002**: Wave ASCII logo MUST be displayed on the left side of the header, using the same character art as the existing `WaveLogo()` function in `theme.go`.
- **FR-003**: Logo MUST animate by cycling the foreground color through a palette (cyan → blue → magenta → cyan) at a fixed 200ms tick interval when `runningCount > 0`. The ASCII character art remains unchanged between frames — only the lipgloss foreground color shifts. This keeps the animation subtle, avoids defining multiple ASCII art variants, and respects the 3-line height constraint.
- **FR-004**: Logo MUST display in its static default frame (cyan foreground) when `runningCount == 0`.
- **FR-005**: Project metadata MUST be displayed as columns to the right of the logo, including: health status indicator, repository name, git branch, git remote name, clean/dirty state, open issues count, and abbreviated commit hash.
- **FR-006**: Branch display MUST update dynamically when a finished pipeline is selected, showing that pipeline's worktree branch instead of the current checked-out branch. The branch name MUST be sourced from a `BranchName` field stored on the pipeline run record in the state database (populated by the executor when the worktree is created).
- **FR-007**: Metadata MUST be sourced from: git state (branch, clean/dirty, commit via `git` CLI subprocess), manifest (`wave.yaml` metadata via `manifest.Load()`), state database (pipeline run status and branch via `StateStore`), and GitHub (open issues via `gh` CLI — requires `gh auth status` to succeed; skipped entirely when `gh` is not authenticated).
- **FR-008**: Header MUST respect the `NO_COLOR` environment variable by disabling all ANSI color and styling escape sequences. Lipgloss handles this automatically via `lipgloss.HasDarkBackground()` / `lipgloss.ColorProfile()` when `NO_COLOR` is set — the header MUST NOT override this behavior.
- **FR-009**: Header MUST adapt gracefully to terminal widths from 80 columns upward, progressively hiding lower-priority metadata columns when space is insufficient. Column priority order (highest to lowest): logo, branch name, health status, repo name, clean/dirty, remote name, open issues count, commit hash.
- **FR-010**: Header MUST accept external messages to update its state (running count changes, pipeline selection changes, metadata refreshes) via Bubble Tea's message-passing architecture.
- **FR-011**: Health status indicator MUST show a visual status derived from the aggregate state of pipeline runs known to the TUI: `● OK` when no pipelines have failed, `▲ WARN` when any pipeline has soft-failure warnings, `✗ ERR` when any pipeline has hard-failed. When no pipeline runs exist, health defaults to `● OK`.
- **FR-012**: Header MUST not block TUI startup — metadata that requires I/O (git commands, `gh` API calls) MUST be loaded asynchronously via Bubble Tea commands (`tea.Cmd`) and the header MUST render with placeholder values ("…" for text fields, `● …` for health) until data arrives.
- **FR-013**: Metadata MUST refresh on two triggers: (1) event-driven — when pipeline state changes are received via Bubble Tea messages (running count changes, pipeline completion), and (2) periodic — a background timer refreshes git state (branch, dirty status, commit hash) every 30 seconds to catch external changes.

### Key Entities

- **HeaderModel**: The Bubble Tea component responsible for rendering the header bar. Holds animation state, metadata values, terminal width, and responds to update messages.
- **HeaderMetadata**: A data structure containing all project metadata fields displayed in the header (branch, repo, commit hash, health status, clean/dirty, running count, issues count).
- **LogoAnimator**: Manages logo color animation state including the current color index, color palette (e.g., `[]lipgloss.Color{"6", "4", "5"}` for cyan/blue/magenta), tick interval (200ms), and active/idle transitions. Uses `tea.Tick` for Bubble Tea-compatible timing.
- **MetadataProvider**: An interface for fetching project metadata from various sources (git CLI, manifest loader, state DB, `gh` CLI) — allows testability via dependency injection. Methods: `FetchGitState() (GitState, error)`, `FetchManifestInfo() (ManifestInfo, error)`, `FetchGitHubInfo() (GitHubInfo, error)`, `FetchPipelineHealth() (HealthStatus, error)`.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Header bar renders within 16ms (single frame budget at 60fps) with all metadata fields populated from cached values.
- **SC-002**: Logo animation transitions between colors at consistent intervals (jitter < 50ms) when pipelines are running.
- **SC-003**: Branch display updates within one render cycle (< 100ms) of a finished pipeline being selected.
- **SC-004**: Header renders correctly at terminal widths of 80, 120, and 200+ columns without layout breakage.
- **SC-005**: All header rendering is covered by unit tests with > 90% line coverage of the header package.
- **SC-006**: NO_COLOR mode produces output with zero ANSI escape sequences, verified by automated test.
- **SC-007**: Header startup renders placeholder values within 100ms, with real metadata populated asynchronously within 2 seconds.
- **SC-008**: `go test -race ./internal/tui/...` passes with no data races in header animation or metadata update paths.

## Clarifications

The following ambiguities were identified and resolved during the clarify step:

### C-001: Logo Animation Frame Definition

**Question**: The spec referenced "distinct visual frames" for logo animation but did not define what the frames look like. Options: (a) multiple ASCII art variants with character mutations, (b) color cycling through a palette, (c) wave-like character shift effects.

**Resolution**: **Color cycling** — the logo foreground color shifts through a palette (cyan → blue → magenta → cyan) at 200ms intervals while the ASCII art stays unchanged. This is the simplest, most robust approach: it avoids maintaining multiple 3-line ASCII art variants, keeps the animation within the fixed 3-line height, and follows the existing pattern in `theme.go` where `WaveLogo()` is pure character art with a single foreground color. The existing `display/animation.go` `Spinner` pattern is not reused because it operates at character-level granularity, not multi-line logo level.

### C-002: Health Status Data Source

**Question**: FR-011 said health is "derived from preflight check results or system state" but preflight results are ephemeral (only run before pipeline execution in `internal/preflight/`). There is no persistent health state in the codebase. What does "health" mean in the dashboard context?

**Resolution**: **Aggregate pipeline run status** — health reflects the state of pipeline runs known to the TUI session: `OK` when no failures, `WARN` for soft failures, `ERR` for hard failures. This is the most actionable real-time signal since the TUI is primarily a pipeline monitoring tool. Preflight results are not cached or reused. When no runs exist, health defaults to `OK`.

### C-003: Open Issues Count — Auth and Data Source

**Question**: FR-005 lists "open issues count" but didn't specify the authentication mechanism, data source, or behavior when GitHub auth is unavailable vs. unreachable.

**Resolution**: Use the **`gh` CLI** for GitHub data, consistent with `internal/github/` patterns in the codebase. The repo is sourced from `manifest.Metadata.Repo` (e.g., `"re-cinq/wave"`), falling back to parsing the git remote origin URL. Authentication is validated via `gh auth status`. Three states: (1) auth not configured → display "—" and skip all GitHub calls, (2) auth configured but API unreachable → display "[offline]" with last cached value, (3) auth configured and reachable → display count. Added as a new edge case.

### C-004: RunRecord Lacks Worktree Branch Field

**Question**: US3 requires showing a finished pipeline's worktree branch, but `state.RunRecord` has no branch field. The executor creates worktrees but doesn't persist the branch name.

**Resolution**: A **`BranchName` field must be added to `RunRecord`** and the state schema. The pipeline executor already knows the branch when creating worktrees (`internal/worktree/`). This is a clean schema addition with a migration. FR-006 updated to reference this field explicitly. This is a prerequisite change that must be implemented before the header can display dynamic branch names.

### C-005: Metadata Refresh Strategy

**Question**: FR-012 specifies async initial load but doesn't define when metadata refreshes after startup. Should git state refresh periodically, on events, or only once?

**Resolution**: **Event-driven + periodic refresh**. Pipeline state changes (running count, completion) trigger immediate metadata refresh via Bubble Tea messages. Git state (branch, dirty, commit) also refreshes on a 30-second background timer via `tea.Tick` to catch external changes (e.g., user commits in another terminal). GitHub data refreshes only on pipeline events (not periodic) to avoid API rate limits. Added as FR-013.
