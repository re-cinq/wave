# Tasks: TUI Finished Pipeline Interactions

**Feature**: #258 — TUI Finished Pipeline Interactions
**Branch**: `258-tui-pipeline-actions`
**Generated**: 2026-03-06
**Spec**: `specs/258-tui-pipeline-actions/spec.md`
**Plan**: `specs/258-tui-pipeline-actions/plan.md`

## Phase 1: Message Types (Setup)

- [X] T001 [P1] [P] [Setup] Add `ChatSessionEndedMsg` to `internal/tui/pipeline_messages.go` — struct with `Err error` field, sent by `tea.Exec()` callback when Claude Code subprocess exits
- [X] T002 [P1] [P] [Setup] Add `BranchCheckoutMsg` to `internal/tui/pipeline_messages.go` — struct with `BranchName string`, `Success bool`, `Err error` fields, sent by `tea.Cmd` closure after `git checkout`
- [X] T003 [P1] [P] [Setup] Add `DiffViewEndedMsg` to `internal/tui/pipeline_messages.go` — struct with `Err error` field, sent by `tea.Exec()` callback when diff pager exits
- [X] T004 [P2] [P] [Setup] Add `FinishedDetailActiveMsg` to `internal/tui/pipeline_messages.go` — struct with `Active bool` field, following `FormActiveMsg`/`LiveOutputActiveMsg` pattern for status bar hint switching

## Phase 2: Data Provider Extensions (Foundational)

- [X] T005 [P1] [US1] Add `WorkspacePath string` and `BranchDeleted bool` fields to `FinishedDetail` struct in `internal/tui/pipeline_detail_provider.go`
- [X] T006 [P1] [US1] Add local `sanitizeBranch(branchName string) string` function in `internal/tui/pipeline_detail_provider.go` — inline the logic from `internal/pipeline/context.go:sanitizeBranchName()` (replace non-alphanumeric with `-`, collapse consecutive dashes, trim, limit to 50 chars)
- [X] T007 [P1] [US1] Extend `FetchFinishedDetail()` in `internal/tui/pipeline_detail_provider.go` to derive workspace path: construct `.wave/workspaces/<RunID>/__wt_<sanitized_branch>/`, validate with `os.Stat()`, fall back to `filepath.Glob(".wave/workspaces/<RunID>/__wt_*")` if primary path not found, set `WorkspacePath` if directory exists
- [X] T008 [P1] [US2] Extend `FetchFinishedDetail()` in `internal/tui/pipeline_detail_provider.go` to check branch existence: if `BranchName` is non-empty, run `exec.Command("git", "rev-parse", "--verify", branchName)`, set `BranchDeleted = true` on non-zero exit
- [X] T009 [P1] [US1] Add tests for workspace path derivation in `internal/tui/pipeline_detail_provider_test.go` — create temp dirs matching workspace convention, verify path resolution for existing and missing workspaces, and for empty branch name fallback via glob
- [X] T010 [P1] [US2] Add tests for branch existence check in `internal/tui/pipeline_detail_provider_test.go` — test with real git repo in temp dir: create branch → `BranchDeleted = false`, delete branch → `BranchDeleted = true`, empty branch name → not checked

## Phase 3: Detail Model — Action Key Handlers (US1: Chat Session)

- [X] T011 [P1] [US1] Add `actionError string` field to `PipelineDetailModel` in `internal/tui/pipeline_detail.go` for transient error display
- [X] T012 [P1] [US1] Handle Enter key in `PipelineDetailModel.Update()` when `paneState == stateFinishedDetail && focused` in `internal/tui/pipeline_detail.go` — validate `WorkspacePath` is non-empty; if empty, set `actionError = "Workspace directory no longer exists — the worktree may have been cleaned up"`; if exists, construct `exec.Command("claude")` with `Dir = WorkspacePath`, return `tea.Exec(cmd, func(err error) tea.Msg { return ChatSessionEndedMsg{Err: err} })`
- [X] T013 [P1] [US1] Handle `ChatSessionEndedMsg` in `PipelineDetailModel.Update()` in `internal/tui/pipeline_detail.go` — re-fetch finished detail via provider (triggers data refresh), return `tea.Batch` including `GitRefreshTickMsg` emission for header bar to detect any branch changes made during chat
- [X] T014 [P1] [US1] Add tests in `internal/tui/pipeline_detail_test.go` — Enter in `stateFinishedDetail` with valid workspace returns `tea.Exec` command; Enter with empty workspace sets `actionError`; `ChatSessionEndedMsg` triggers re-fetch

## Phase 4: Detail Model — Action Key Handlers (US2: Branch Checkout)

- [X] T015 [P1] [US2] Handle `b` key in `PipelineDetailModel.Update()` when `paneState == stateFinishedDetail && focused` in `internal/tui/pipeline_detail.go` — gate on `!branchDeleted && finishedDetail.BranchName != ""`; return `tea.Cmd` closure that runs `exec.Command("git", "checkout", branch).CombinedOutput()`, returns `BranchCheckoutMsg{BranchName, success, err}`
- [X] T016 [P1] [US2] Handle `BranchCheckoutMsg` in `PipelineDetailModel.Update()` in `internal/tui/pipeline_detail.go` — on success: clear `actionError`, return `GitRefreshTickMsg` command; on failure: set `actionError` to error message (e.g., "Branch checkout failed: <git error>")
- [X] T017 [P1] [US2] Add tests in `internal/tui/pipeline_detail_test.go` — `b` with valid branch returns checkout command; `b` with `branchDeleted=true` is no-op; `b` with empty `BranchName` is no-op; `BranchCheckoutMsg` success clears error and returns git refresh; `BranchCheckoutMsg` failure sets `actionError`

## Phase 5: Detail Model — Action Key Handlers (US3: Diff View)

- [X] T018 [P2] [US3] Handle `d` key in `PipelineDetailModel.Update()` when `paneState == stateFinishedDetail && focused` in `internal/tui/pipeline_detail.go` — gate on `!branchDeleted && finishedDetail.BranchName != ""`; return `tea.Exec(exec.Command("git", "diff", "main..."+branch), func(err error) tea.Msg { return DiffViewEndedMsg{Err: err} })`
- [X] T019 [P2] [US3] Handle `DiffViewEndedMsg` in `PipelineDetailModel.Update()` in `internal/tui/pipeline_detail.go` — no-op (TUI resumes automatically, diff is read-only so no data refresh needed)
- [X] T020 [P2] [US3] Add tests in `internal/tui/pipeline_detail_test.go` — `d` with valid branch returns `tea.Exec` command; `d` with `branchDeleted=true` is no-op; `d` with empty `BranchName` is no-op; `DiffViewEndedMsg` is handled without error

## Phase 6: Detail Model — Transient Error and Rendering (Cross-cutting)

- [X] T021 [P1] [US1] Add transient error clearing logic in `PipelineDetailModel.Update()` for `tea.KeyMsg` in `stateFinishedDetail` in `internal/tui/pipeline_detail.go` — clear `actionError` on any key press before processing the key (so errors dismiss on next interaction)
- [X] T022 [P1] [US1] Update `renderFinishedDetail()` signature in `internal/tui/pipeline_detail.go` to accept `actionError string` and `workspacePath string` parameters (or pass full detail struct); update call site in `updateViewportContent()` to pass these values
- [X] T023 [P2] [US3] Fix `[d]` hint fainting in `renderFinishedDetail()` in `internal/tui/pipeline_detail.go` — apply `Faint(true)` to `diffHint` when `branchDeleted || branchName == ""` (C15 fix, currently only `[b]` is fainted)
- [X] T024 [P1] [US1] Add `[Enter]` hint fainting in `renderFinishedDetail()` in `internal/tui/pipeline_detail.go` — apply `Faint(true)` to `enterHint` when `workspacePath == ""`
- [X] T025 [P1] [US1] Add action error rendering in `renderFinishedDetail()` in `internal/tui/pipeline_detail.go` — when `actionError != ""`, render red-styled error text instead of action hints section
- [X] T026 [P1] [US2] Update `branchDeleted` field sourcing in `PipelineDetailModel` in `internal/tui/pipeline_detail.go` — when `DetailDataMsg` arrives with `FinishedDetail`, also update `m.branchDeleted` from `FinishedDetail.BranchDeleted` (overrides `PipelineSelectedMsg.BranchDeleted` which is currently always false)
- [X] T027 [P1] [US1] Add rendering tests in `internal/tui/pipeline_detail_test.go` — `[d]` fainted when `branchDeleted`; `[Enter]` fainted when workspace path empty; action error rendered in red when set; action error replaces action hints; error clears on next keypress

## Phase 7: Content Model — Message Routing (US4: Dynamic Header + Integration)

- [X] T028 [P2] [US4] Route `ChatSessionEndedMsg` in `ContentModel.Update()` in `internal/tui/content.go` — forward to detail model
- [X] T029 [P2] [US4] Route `BranchCheckoutMsg` in `ContentModel.Update()` in `internal/tui/content.go` — forward to detail model
- [X] T030 [P2] [US4] Route `DiffViewEndedMsg` in `ContentModel.Update()` in `internal/tui/content.go` — forward to detail model
- [X] T031 [P2] [US4] Emit `FinishedDetailActiveMsg{Active: true}` in `ContentModel.Update()` in `internal/tui/content.go` — when Enter focuses right pane on a finished item (add alongside existing `FocusChangedMsg` emission in the Enter handler)
- [X] T032 [P2] [US4] Emit `FinishedDetailActiveMsg{Active: false}` in `ContentModel.Update()` in `internal/tui/content.go` — when Esc returns to left pane (add alongside existing `LiveOutputActiveMsg{Active: false}` emission)
- [X] T033 [P2] [US4] Add tests in `internal/tui/content_test.go` — Enter on finished item emits `FinishedDetailActiveMsg{Active: true}`; Esc from finished detail emits `FinishedDetailActiveMsg{Active: false}`; new message types forwarded to detail model

## Phase 8: Status Bar — Finished Detail Hints (US5: Status Bar)

- [X] T034 [P3] [US5] Add `finishedDetailActive bool` field to `StatusBarModel` in `internal/tui/statusbar.go`
- [X] T035 [P3] [US5] Handle `FinishedDetailActiveMsg` in `StatusBarModel.Update()` in `internal/tui/statusbar.go` — set `finishedDetailActive = msg.Active`
- [X] T036 [P3] [US5] Update hint rendering in `StatusBarModel.View()` in `internal/tui/statusbar.go` — insert after `liveOutputActive` check, before generic right-pane: when `finishedDetailActive && focusPane == FocusPaneRight`, show `"[Enter] Chat  [b] Branch  [d] Diff  [Esc] Back"`
- [X] T037 [P3] [US5] Add tests in `internal/tui/statusbar_test.go` — `FinishedDetailActiveMsg{Active: true}` sets field; status bar renders finished detail hints when active; hint priority chain: form > live output > finished detail > generic

## Phase 9: App Model — Message Forwarding (Integration)

- [X] T038 [P2] [P] [US4] Forward `FinishedDetailActiveMsg` to status bar in `AppModel.Update()` in `internal/tui/app.go` — add to the `switch msg.(type)` case alongside `FocusChangedMsg`, `FormActiveMsg`, `LiveOutputActiveMsg`
- [X] T039 [P2] [US4] Add test in `internal/tui/app_test.go` — `FinishedDetailActiveMsg` forwarded to status bar (verify `finishedDetailActive` state changes)

## Phase 10: Polish & Cross-cutting

- [X] T040 [P1] [US1] Ensure `NO_COLOR` environment variable is passed through to subprocesses in `internal/tui/pipeline_detail.go` — `exec.Command` inherits environment by default (verify no explicit `Env` override strips it); add test assertion if needed (FR-018)
- [X] T041 [P1] [Integration] Run `go test ./internal/tui/...` to verify all existing tests pass after changes — no regressions in pipeline list, detail, header, status bar, launch flow, or live output components (SC-008)
- [X] T042 [P1] [Integration] Run `go test -race ./...` to verify no race conditions introduced by the new action key handlers and background checkout command
