# Implementation Plan: TUI Header Bar with Animated Logo and Project Metadata

**Branch**: `253-tui-header-bar` | **Date**: 2026-03-05 | **Spec**: `specs/253-tui-header-bar/spec.md`
**Input**: Feature specification from `/specs/253-tui-header-bar/spec.md`

## Summary

Replace the stub `HeaderModel` in `internal/tui/header.go` with a full-featured header bar component that displays the Wave ASCII logo (with color-cycling animation during pipeline execution) alongside responsive metadata columns showing project/git state, pipeline health, and GitHub info. Requires a prerequisite schema migration to add `branch_name` to `pipeline_run`, a `MetadataProvider` interface for testable async data fetching, and integration with Bubble Tea's message-passing architecture for real-time updates.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/lipgloss` v1.1.0, `modernc.org/sqlite` (existing)
**Storage**: SQLite via `internal/state/` — migration #7 adds `branch_name` column
**Testing**: `go test -race ./...` — table-driven tests with mock `MetadataProvider`
**Target Platform**: Linux/macOS terminals, 80+ column width
**Project Type**: Single Go binary (single project structure)
**Performance Goals**: <16ms render (SC-001), <50ms animation jitter (SC-002), >90% coverage (SC-005)
**Constraints**: 3-line fixed header height, NO_COLOR compliance, no new Go dependencies

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ PASS | No new dependencies. Uses existing bubbletea/lipgloss. `gh` CLI is optional external tool. |
| P2: Manifest as Source of Truth | ✅ PASS | Project metadata sourced from `wave.yaml` via `manifest.Load()`. |
| P3: Persona-Scoped Execution | N/A | TUI is user-facing, not a persona execution context. |
| P4: Fresh Memory at Step Boundary | N/A | Not a pipeline step. |
| P5: Navigator-First Architecture | N/A | Not a pipeline. |
| P6: Contracts at Every Handover | N/A | Not a pipeline. |
| P7: Relay via Summarizer | N/A | Not a pipeline. |
| P8: Ephemeral Workspaces | N/A | Not a pipeline. |
| P9: Credentials Never Touch Disk | ✅ PASS | GitHub auth handled by `gh` CLI. No tokens stored. |
| P10: Observable Progress | ✅ PASS | Header displays pipeline health status — enhances observability. |
| P11: Bounded Recursion | N/A | Not a pipeline. |
| P12: Minimal Step State Machine | ✅ PASS | Schema migration adds field, doesn't modify state machine. |
| P13: Test Ownership | ✅ PASS | Existing header_test.go will be updated. New tests added. `go test -race ./...` required. |

**Re-check after Phase 1**: ✅ All principles still satisfied. No violations.

## Project Structure

### Documentation (this feature)

```
specs/253-tui-header-bar/
├── plan.md              # This file
├── research.md          # Phase 0 output — technology decisions
├── data-model.md        # Phase 1 output — entity definitions
├── checklists/          # Checklist artifacts
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/
├── tui/
│   ├── header.go            # MODIFY: Replace stub with full HeaderModel
│   ├── header_test.go       # MODIFY: Replace stub tests with comprehensive tests
│   ├── header_metadata.go   # NEW: MetadataProvider interface + implementations
│   ├── header_messages.go   # NEW: Bubble Tea message types for header
│   ├── header_logo.go       # NEW: LogoAnimator with tea.Tick integration
│   ├── app.go               # MODIFY: Forward messages to header, wire Init/Update
│   ├── app_test.go          # MODIFY: Update for new header interface
│   └── theme.go             # REFERENCE: WaveLogo() character art (reuse)
├── state/
│   ├── types.go             # MODIFY: Add BranchName to RunRecord
│   ├── store.go             # MODIFY: Add UpdateRunBranch method, update queries
│   ├── migration_definitions.go  # MODIFY: Add migration #7
│   ├── schema.sql           # MODIFY: Add branch_name column
│   └── migrations_test.go   # MODIFY: Add migration #7 test
└── pipeline/
    └── executor.go          # MODIFY: Persist branch name on run creation
```

**Structure Decision**: All new code fits within existing package structure. The header is split into focused files (metadata, messages, logo) to keep each file under ~200 lines and maintain single-responsibility. No new packages needed.

## Implementation Phases

### Phase A: Schema Migration (Prerequisite)

**Goal**: Add `branch_name` field to `pipeline_run` table and `RunRecord` type.

1. Add migration #7 to `migration_definitions.go`:
   - `ALTER TABLE pipeline_run ADD COLUMN branch_name TEXT DEFAULT ''`
   - Down migration uses table recreation pattern (SQLite < 3.35 compat)
2. Update `schema.sql` to include `branch_name` column
3. Add `BranchName string` field to `RunRecord` in `types.go`
4. Add `UpdateRunBranch(runID, branch string) error` to `StateStore` interface
5. Implement `UpdateRunBranch` in `store.go`
6. Update `GetRun`, `GetRunningRuns`, `ListRuns`, `queryRunsWithArgs` to scan `branch_name`
7. Update `pipeline/executor.go` to call `UpdateRunBranch` when worktree branch is resolved
8. Add migration #7 tests

### Phase B: Header Data Types & Messages

**Goal**: Define all types and messages the header component needs.

1. Create `header_messages.go` with all Bubble Tea message types:
   - `GitStateMsg`, `ManifestInfoMsg`, `GitHubInfoMsg`, `PipelineHealthMsg`
   - `RunningCountMsg`, `PipelineSelectedMsg`
   - `LogoTickMsg`, `GitRefreshTickMsg`
2. Create `header_metadata.go` with:
   - `HeaderMetadata` struct (all displayable fields)
   - `HealthStatus` enum (OK, Warn, Err)
   - `GitHubAuthState` enum (NotConfigured, Offline, Connected)
   - `GitState`, `ManifestInfo`, `GitHubInfo` value types
   - `MetadataProvider` interface
   - `DefaultMetadataProvider` implementation (git CLI, manifest loader, gh CLI, state DB)

### Phase C: Logo Animator

**Goal**: Implement color-cycling animation using `tea.Tick`.

1. Create `header_logo.go` with `LogoAnimator` struct:
   - Color palette: `[]lipgloss.Color{"6", "4", "5"}` (cyan, blue, magenta)
   - `Tick() tea.Cmd` — returns `tea.Tick(200ms, LogoTickMsg)`
   - `Update(LogoTickMsg)` — advances `colorIndex`
   - `SetActive(bool)` — starts/stops tick scheduling
   - `View(logoText string) string` — renders logo with current palette color
2. Reuse `WaveLogo()` character art from `theme.go` (without margins/styling)

### Phase D: Header Model Rewrite

**Goal**: Replace the stub header with the full implementation.

1. Rewrite `header.go`:
   - `HeaderModel` struct with `width`, `metadata`, `logo`, `provider`, `refreshTimer`
   - `NewHeaderModel(provider MetadataProvider) HeaderModel`
   - `Init() tea.Cmd` — returns batch of async fetch commands + git refresh tick
   - `Update(msg tea.Msg) (HeaderModel, tea.Cmd)`:
     - Handle `GitStateMsg` → update metadata.Branch, CommitHash, IsDirty, RemoteName
     - Handle `ManifestInfoMsg` → update metadata.ProjectName, RepoName
     - Handle `GitHubInfoMsg` → update metadata.IssuesCount, GitHubState
     - Handle `PipelineHealthMsg` → update metadata.HealthStatus
     - Handle `RunningCountMsg` → update metadata.RunningCount, toggle logo animation
     - Handle `PipelineSelectedMsg` → update metadata.OverrideBranch
     - Handle `LogoTickMsg` → advance logo color, schedule next tick if active
     - Handle `GitRefreshTickMsg` → trigger git state refetch, schedule next tick
   - `View() string`:
     - Render logo with current animation color
     - Build metadata columns in priority order
     - Measure and fit columns within `m.width`
     - Join horizontally with lipgloss
   - `SetWidth(w int)` — update width for responsive reflow

2. Update `app.go`:
   - `NewAppModel` takes `MetadataProvider` parameter (or uses default)
   - `Init()` returns `m.header.Init()` (not nil)
   - `Update()` passes messages to `m.header.Update()` and collects commands
   - Merge header commands with app commands using `tea.Batch`

### Phase E: Tests

**Goal**: >90% coverage, race-safe, NO_COLOR compliance verified.

1. Rewrite `header_test.go`:
   - Mock `MetadataProvider` that returns canned data
   - Test `View()` at widths 80, 120, 200 — verify column priority degradation
   - Test `Update()` with each message type — verify state transitions
   - Test logo animation: active/inactive toggle, color cycling
   - Test NO_COLOR mode: set `NO_COLOR=1`, verify no ANSI escapes in output
   - Test edge cases: no git, no manifest, no GitHub auth, deleted branch
   - Test placeholder rendering before async data arrives
2. Update `app_test.go`:
   - Update `NewAppModel()` calls to pass mock provider
   - Verify `Init()` returns commands (not nil)
   - Verify message forwarding to header
3. Add state migration test for migration #7
4. Run `go test -race ./internal/tui/... ./internal/state/... ./internal/pipeline/...`

## Key Design Decisions

1. **Header is a sub-model, not a standalone `tea.Model`**: The header doesn't implement `tea.Model` directly — it has `Init()`, `Update()`, `View()` methods but returns `(HeaderModel, tea.Cmd)` from Update, not `(tea.Model, tea.Cmd)`. The parent `AppModel` owns the Bubble Tea lifecycle. This matches the existing pattern in the codebase.

2. **File split by concern**: Rather than a single 500+ line `header.go`, the implementation is split into `header.go` (model/view), `header_messages.go` (message types), `header_metadata.go` (data provider), `header_logo.go` (animation). Each file stays focused and testable.

3. **Graceful degradation over hard failures**: Every data source can fail independently. The header never blocks on I/O and always renders something useful — placeholders initially, error states when data sources fail.

4. **No new packages**: All header code lives in `internal/tui/`. The `MetadataProvider` implementation uses existing packages (`internal/manifest`, `internal/state`) directly rather than creating wrapper packages.

## Complexity Tracking

_No constitution violations to justify._
