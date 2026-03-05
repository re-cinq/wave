# Research: TUI Header Bar with Animated Logo and Project Metadata

**Date**: 2026-03-05
**Branch**: `253-tui-header-bar`

## Phase 0 — Unknowns & Technology Decisions

### R-001: Bubble Tea Animation Pattern for Logo Color Cycling

**Decision**: Use `tea.Tick` command pattern for 200ms color cycling.

**Rationale**: The existing `display/animation.go` `Spinner` uses goroutines and mutexes (`time.NewTicker` + `sync.Mutex`), which is inappropriate for Bubble Tea's single-threaded message loop. Bubble Tea has a first-class `tea.Tick` command that returns a `tea.Msg` on a timer, which integrates cleanly with `Update()`. The `charmbracelet/bubbletea` package (already in go.mod v1.3.10) provides `tea.Tick(duration, func(time.Time) tea.Msg)` for exactly this purpose.

**Alternatives Rejected**:
- **Reuse `display.Spinner`**: Uses goroutines/mutexes which bypasses Bubble Tea's concurrency model. Would require channel bridge adding complexity without benefit.
- **lipgloss animation library**: No first-party animation support in lipgloss. Third-party libraries would add unnecessary dependencies.

### R-002: Metadata Fetching Strategy (Async I/O in Bubble Tea)

**Decision**: Use `tea.Cmd` functions for async data loading, returning typed messages to `Update()`.

**Rationale**: Bubble Tea commands (`tea.Cmd`) are the idiomatic way to perform I/O. A `tea.Cmd` is a `func() tea.Msg` that runs in a goroutine managed by the Bubble Tea runtime. The header model will return `tea.Cmd` values from `Init()` and `Update()` to trigger git CLI calls, manifest loading, and `gh` CLI calls. Results arrive as typed messages (`GitStateMsg`, `ManifestInfoMsg`, `GitHubInfoMsg`).

**Alternatives Rejected**:
- **Direct subprocess calls in `View()`**: Blocks rendering. Violates Bubble Tea's pure rendering model.
- **Background goroutines with channels**: Bypasses Bubble Tea's message loop. Requires manual synchronization.

### R-003: GitHub Open Issues Count — `gh` CLI vs Go HTTP Client

**Decision**: Use `gh` CLI subprocess via `exec.Command` for GitHub data, consistent with the spec's FR-007.

**Rationale**: The existing `internal/github/` package uses a Go HTTP client that requires a `GITHUB_TOKEN` and direct API calls. The spec explicitly calls for `gh` CLI integration (`gh auth status` for auth check, `gh api` for data). Using `gh` CLI:
1. Inherits the user's existing `gh auth` session — no token management needed
2. Handles auth, caching, and rate limiting transparently
3. Consistent with the spec's three-state model: no auth → "—", auth but unreachable → "[offline]", working → count

**Alternatives Rejected**:
- **`internal/github.Client`**: Requires explicit token configuration. Not suitable for TUI — users shouldn't need to configure API tokens to see issue counts.
- **GraphQL API**: More efficient but adds complexity. The REST endpoint `GET /repos/{owner}/{repo}` already returns `open_issues_count` which is sufficient.

### R-004: State DB Schema Change — `BranchName` on `pipeline_run`

**Decision**: Add migration #7 with `ALTER TABLE pipeline_run ADD COLUMN branch_name TEXT DEFAULT ''`.

**Rationale**: `RunRecord` currently has no branch field (confirmed by grep of `internal/state/types.go`). The pipeline executor knows the branch at worktree creation time (`executor.go:978-994`). A new migration following the established pattern (see `migration_definitions.go`, currently at v6) will add the column. The executor's worktree setup section will be updated to call a new `UpdateRunBranch(runID, branch)` method on `StateStore`.

**Alternatives Rejected**:
- **Store branch in event log**: Event logs are append-only and not efficiently queryable by run.
- **Derive from workspace path**: Workspace paths are implementation details; branch names are sometimes sanitized differently.

### R-005: MetadataProvider Interface Design

**Decision**: Define a `MetadataProvider` interface in the TUI package with four methods matching the spec's four data sources.

**Rationale**: The spec defines four data sources: git CLI, manifest, state DB, and GitHub CLI. An interface with `FetchGitState()`, `FetchManifestInfo()`, `FetchGitHubInfo()`, and `FetchPipelineHealth()` enables:
1. Unit testing with mock providers (no subprocess calls in tests)
2. Clear separation between data fetching and rendering
3. Each method can fail independently — the header degrades gracefully per source

**Alternatives Rejected**:
- **Single `FetchAll()` method**: Blocks on the slowest source. Can't refresh sources independently.
- **Direct CLI calls in the model**: Untestable without integration tests. Violates separation of concerns.

### R-006: Responsive Column Layout Strategy

**Decision**: Compute visible columns at render time based on terminal width, using priority-ordered column definitions.

**Rationale**: FR-009 specifies columns with priority order: logo > branch > health > repo > dirty > remote > issues > commit hash. At render time, the `View()` method will:
1. Start with the logo (always shown) and calculate remaining width
2. Add columns in priority order, stopping when remaining width is insufficient
3. Use lipgloss `Width()` to measure actual rendered width including ANSI codes

This is the simplest approach — no breakpoint tables needed. The rendering loop naturally handles any width.

**Alternatives Rejected**:
- **Fixed breakpoints**: Inflexible, doesn't adapt to varying column content lengths.
- **CSS-like flex model**: Over-engineered for a fixed set of columns with known priorities.

### R-007: NO_COLOR Compliance via lipgloss

**Decision**: Rely on lipgloss's built-in `NO_COLOR` support — no custom handling needed.

**Rationale**: FR-008 explicitly states that lipgloss handles `NO_COLOR` automatically via `ColorProfile()`. When `NO_COLOR` is set, `lipgloss.NewStyle().Foreground(...)` becomes a no-op. The header MUST NOT override this behavior. Tests can verify by setting `NO_COLOR=1` and checking output for absence of `\x1b[` escape sequences.

**Alternatives Rejected**: None — this is the correct approach per the spec.

### R-008: Header State Update Architecture

**Decision**: Use typed Bubble Tea messages for all header state changes.

**Rationale**: FR-010 specifies that the header accepts external messages for state updates. Define message types:
- `RunningCountMsg{Count int}` — pipeline running count changed
- `PipelineSelectedMsg{RunID string, BranchName string}` — finished pipeline selected
- `MetadataRefreshMsg{}` — trigger a metadata refresh
- `GitStateMsg{...}` / `ManifestInfoMsg{...}` / `GitHubInfoMsg{...}` — async fetch results

The `AppModel.Update()` will forward relevant messages to `HeaderModel.Update()`, which returns `(HeaderModel, tea.Cmd)`.

**Alternatives Rejected**:
- **Direct method calls**: Breaks Bubble Tea's immutable update model.
- **Shared state via pointers**: Race conditions, testing difficulty.
