# Tasks: TUI Header Bar with Animated Logo and Project Metadata

**Branch**: `253-tui-header-bar` | **Date**: 2026-03-05  
**Source**: `specs/253-tui-header-bar/plan.md`, `specs/253-tui-header-bar/spec.md`

## Phase 1: Setup

_No project initialization needed — existing project structure._

## Phase 2: Foundational Prerequisites

These tasks add the schema migration and define all types/messages the header needs.

- [X] T001 [P1] [Prereq] Add migration #7 `ALTER TABLE pipeline_run ADD COLUMN branch_name TEXT DEFAULT ''` to `internal/state/migration_definitions.go` following the v6 migration pattern (Up + Down with table recreation)
- [X] T002 [P1] [Prereq] Add `BranchName string` field to `RunRecord` in `internal/state/types.go`
- [X] T003 [P1] [Prereq] Add `branch_name` column to the `pipeline_run` CREATE TABLE in `internal/state/schema.sql`
- [X] T004 [P1] [Prereq] Update `GetRun`, `queryRunsWithArgs` SELECT queries in `internal/state/store.go` to include `branch_name` in the column list and scan it into `record.BranchName`
- [X] T005 [P1] [Prereq] Add `UpdateRunBranch(runID string, branch string) error` method to `StateStore` interface and implement it in `internal/state/store.go` with `UPDATE pipeline_run SET branch_name = ? WHERE run_id = ?`
- [X] T006 [P1] [Prereq] Update `CreateRun` in `internal/state/store.go` to accept and persist `branch_name` (or keep it empty and rely on `UpdateRunBranch` called later by the executor — match the plan's approach of calling `UpdateRunBranch` when worktree branch is resolved)
- [X] T007 [P1] [Prereq] Update `internal/pipeline/executor.go` to call `store.UpdateRunBranch(runID, branch)` after the worktree branch is resolved (around line 991 where `execution.Context.BranchName` is set)
- [X] T008 [P1] [P] [Prereq] Create `internal/tui/header_messages.go` with all Bubble Tea message types: `GitStateMsg`, `ManifestInfoMsg`, `GitHubInfoMsg`, `PipelineHealthMsg`, `RunningCountMsg`, `PipelineSelectedMsg`, `LogoTickMsg`, `GitRefreshTickMsg` — as defined in `specs/253-tui-header-bar/data-model.md`
- [X] T009 [P1] [P] [Prereq] Create `internal/tui/header_metadata.go` with: `HeaderMetadata` struct, `HealthStatus` enum (OK/Warn/Err), `GitHubAuthState` enum (NotConfigured/Offline/Connected), `GitState`/`ManifestInfo`/`GitHubInfo` value types, and `MetadataProvider` interface with four methods (`FetchGitState`, `FetchManifestInfo`, `FetchGitHubInfo`, `FetchPipelineHealth`) — as defined in `specs/253-tui-header-bar/data-model.md`

## Phase 3: US1 — Static Header with Project Metadata (P1)

Core header rewrite with metadata columns and responsive layout.

- [X] T010 [P1] [US1] [P] Implement `DefaultMetadataProvider.FetchGitState()` in `internal/tui/header_metadata.go` — runs `git rev-parse --abbrev-ref HEAD`, `git rev-parse --short HEAD`, `git status --porcelain`, `git remote` via `exec.Command`, returns `GitState` struct. Handle no-git case with `"[no git]"` fallbacks
- [X] T011 [P1] [US1] [P] Implement `DefaultMetadataProvider.FetchManifestInfo()` in `internal/tui/header_metadata.go` — loads `wave.yaml` via `manifest.Load()` from `internal/manifest`, extracts `ProjectName` and `RepoName` from metadata. Handle missing manifest with `"[no project]"` fallback
- [X] T012 [P1] [US1] [P] Implement `DefaultMetadataProvider.FetchGitHubInfo()` in `internal/tui/header_metadata.go` — runs `gh auth status` to check auth, then `gh api repos/{owner}/{repo}` to get `open_issues_count`. Three states: not configured → `GitHubNotConfigured`, unreachable → `GitHubOffline`, working → `GitHubConnected` with count
- [X] T013 [P1] [US1] [P] Implement `DefaultMetadataProvider.FetchPipelineHealth()` in `internal/tui/header_metadata.go` — accepts a `StateStore` dependency, queries runs via `ListRuns`, aggregates to `HealthOK`/`HealthWarn`/`HealthErr` based on run statuses
- [X] T014 [P1] [US1] Create `internal/tui/header_logo.go` with `LogoAnimator` struct: color palette `[]lipgloss.Color{"6", "4", "5"}`, `colorIndex int`, `active bool`, `View(logoText string) string` method that applies current palette color. Static mode only (animation added in Phase 4). Reuse `WaveLogo()` character art from `internal/tui/theme.go` (without margins)
- [X] T015 [P1] [US1] Rewrite `HeaderModel` in `internal/tui/header.go`: add fields for `metadata HeaderMetadata`, `logo LogoAnimator`, `provider MetadataProvider`, `refreshTimer time.Duration`. Implement `NewHeaderModel(provider MetadataProvider) HeaderModel`. Implement `Init() tea.Cmd` returning a batch of async fetch commands (FetchGitState, FetchManifestInfo, FetchGitHubInfo, FetchPipelineHealth) plus a 30s git refresh tick
- [X] T016 [P1] [US1] Implement `HeaderModel.Update(msg tea.Msg) (HeaderModel, tea.Cmd)` in `internal/tui/header.go` — handle `GitStateMsg`, `ManifestInfoMsg`, `GitHubInfoMsg`, `PipelineHealthMsg`, `GitRefreshTickMsg` to update metadata fields and schedule next refresh tick
- [X] T017 [P1] [US1] Implement `HeaderModel.View() string` in `internal/tui/header.go` — render logo via `LogoAnimator.View()`, build metadata columns in priority order (FR-009: logo > branch > health > repo > dirty > remote > issues > commit), measure and fit within `m.width`, join horizontally with lipgloss. Render placeholder `"…"` for unfetched fields
- [X] T018 [P1] [US1] Update `NewAppModel()` in `internal/tui/app.go` to accept `MetadataProvider` parameter (or construct `DefaultMetadataProvider` internally). Wire `Init()` to return `m.header.Init()`. Update `Update()` to forward messages to `m.header.Update()` and batch returned commands

## Phase 4: US2 — Animated Logo During Pipeline Execution (P2)

Logo color-cycling animation keyed to running pipeline count.

- [X] T019 [P2] [US2] Add `Tick() tea.Cmd` to `LogoAnimator` in `internal/tui/header_logo.go` — returns `tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return LogoTickMsg{} })`. Add `SetActive(active bool)` to start/stop tick scheduling. When deactivated, reset `colorIndex` to 0 (static cyan)
- [X] T020 [P2] [US2] Add `LogoTickMsg` handling in `HeaderModel.Update()` in `internal/tui/header.go` — advance `logo.colorIndex = (colorIndex + 1) % len(palette)`, schedule next tick via `logo.Tick()` if `logo.active`
- [X] T021 [P2] [US2] Add `RunningCountMsg` handling in `HeaderModel.Update()` in `internal/tui/header.go` — update `metadata.RunningCount`, call `logo.SetActive(count > 0)`. If transitioning to active, return `logo.Tick()` cmd. If transitioning to inactive, return nil (stops ticks)

## Phase 5: US3 — Dynamic Branch Display on Pipeline Selection (P2)

Header branch updates when a finished pipeline is selected.

- [X] T022 [P2] [US3] Add `PipelineSelectedMsg` handling in `HeaderModel.Update()` in `internal/tui/header.go` — set `metadata.OverrideBranch` from msg. When `OverrideBranch != ""`, `View()` displays it instead of `metadata.Branch`. When empty, reverts to current branch
- [X] T023 [P2] [US3] Add edge case handling for deleted worktree branch in `View()` in `internal/tui/header.go` — if `OverrideBranch` is set but the branch no longer exists (indicated by a `BranchDeleted bool` field on `PipelineSelectedMsg`), append `" [deleted]"` suffix to the displayed branch name

## Phase 6: US4 — NO_COLOR and Accessibility (P3)

- [X] T024 [P3] [US4] [P] Verify NO_COLOR compliance in `HeaderModel.View()` in `internal/tui/header.go` — ensure no manual ANSI codes are written; all styling goes through lipgloss which respects `NO_COLOR` automatically (FR-008). No custom handling needed per research R-007
- [X] T025 [P3] [US4] Implement progressive column degradation in `HeaderModel.View()` in `internal/tui/header.go` — when width is insufficient, hide columns in reverse priority order: commit hash first, then issues count, remote, dirty state, repo name, health. Logo and branch always visible (FR-009). This is part of the responsive layout logic in T017 but verify it works at exactly 80 columns

## Phase 7: Tests & Polish

- [X] T026 [P1] [Test] [P] Create mock `MetadataProvider` in `internal/tui/header_test.go` returning canned data for all four methods. Table-driven tests for `Update()` with each message type — verify correct state transitions
- [X] T027 [P1] [Test] [P] Write `View()` rendering tests in `internal/tui/header_test.go` at widths 80, 120, 200 — verify column priority degradation at 80 columns, all columns visible at 200, correct logo rendering at all widths
- [X] T028 [P2] [Test] [P] Write logo animation tests in `internal/tui/header_test.go` — test `SetActive(true)` starts ticks, `SetActive(false)` stops and resets to cyan, color index cycles correctly through palette
- [X] T029 [P2] [Test] [P] Write `PipelineSelectedMsg` tests in `internal/tui/header_test.go` — verify branch override displays correctly, empty override reverts to current branch, deleted branch shows `[deleted]` suffix
- [X] T030 [P3] [Test] Write NO_COLOR test in `internal/tui/header_test.go` — set `NO_COLOR=1` via `t.Setenv`, render header, verify output contains zero `\x1b[` escape sequences (SC-006)
- [X] T031 [P1] [Test] Write edge case tests in `internal/tui/header_test.go` — test no git available (`FetchGitState` returns error → "[no git]"), no manifest (`FetchManifestInfo` returns error → "[no project]"), no GitHub auth (`FetchGitHubInfo` returns `GitHubNotConfigured` → "—"), placeholder rendering before async data arrives
- [X] T032 [P1] [Test] [P] Add migration #7 test in `internal/state/migrations_test.go` — verify Up creates `branch_name` column, Down recreates table without it, following existing test patterns
- [X] T033 [P1] [Test] Update `internal/tui/app_test.go` — update `NewAppModel()` calls to pass mock provider, verify `Init()` returns non-nil cmd (async fetches), verify `Update()` forwards messages to header and batches commands
- [X] T034 [P1] [Test] Run `go test -race ./internal/tui/... ./internal/state/... ./internal/pipeline/...` — verify no data races in header animation or metadata update paths (SC-008)

## Dependency Graph

```
T001 → T002 → T003 → T004 → T005 → T006 → T007  (schema migration chain)
T008, T009  (parallel, no dependencies)
T010, T011, T012, T013  (parallel, depend on T009)
T014  (depends on T008)
T015  (depends on T008, T009, T014)
T016  (depends on T015)
T017  (depends on T015, T014)
T018  (depends on T015)
T019  (depends on T014)
T020, T021  (depend on T019, T016)
T022  (depends on T016, T005)
T023  (depends on T022)
T024  (depends on T017)
T025  (depends on T017)
T026-T034  (depend on all implementation tasks)
```

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 34 |
| Phase 2 (Prerequisites) | 9 tasks |
| Phase 3 (US1 — Static Header) | 9 tasks |
| Phase 4 (US2 — Animation) | 3 tasks |
| Phase 5 (US3 — Dynamic Branch) | 2 tasks |
| Phase 6 (US4 — NO_COLOR) | 2 tasks |
| Phase 7 (Tests & Polish) | 9 tasks |
| Parallelizable tasks | 14 |
