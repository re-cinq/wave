# Implementation Plan: TUI Finished Pipeline Interactions

**Branch**: `258-tui-pipeline-actions` | **Date**: 2026-03-06 | **Spec**: `specs/258-tui-pipeline-actions/spec.md`
**Input**: Feature specification from `/specs/258-tui-pipeline-actions/spec.md`

## Summary

Implement interactive actions for finished pipelines in the TUI. When a finished pipeline's detail view is focused, Enter opens an interactive Claude Code session in the pipeline's workspace (via `tea.Exec()`), `b` checks out the pipeline's branch (`git checkout` as a background command), and `d` opens a diff view (`git diff main...<branch>` through the user's pager via `tea.Exec()`). Extends `FinishedDetail` with workspace path (derived from RunID + sanitized branch) and branch existence (via `git rev-parse`). Adds status bar context hints and transient error display for failed actions.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `charmbracelet/bubbletea` v1.3.10, `charmbracelet/bubbles/viewport` (existing)
**Storage**: SQLite via `internal/state` (existing — RunRecord lookup for workspace path derivation)
**Testing**: `go test` with `testify/assert`, `testify/require`
**Target Platform**: Linux/macOS terminal (80–300 columns, 24–100 rows)
**Project Type**: Single Go binary — changes in `internal/tui/`
**Constraints**: No new external dependencies; must not break existing tests (`go test ./...`)

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | ✅ Pass | No new runtime dependencies. Uses existing bubbletea, `os/exec` stdlib. |
| P2: Manifest as SSOT | ✅ Pass | No manifest changes. Pipeline data flows from state store. |
| P3: Persona-Scoped Execution | ✅ Pass | Chat session spawns raw `claude` binary, not a persona execution. TUI is a display/interaction layer. |
| P4: Fresh Memory at Step Boundary | ✅ Pass | Chat session is an independent invocation, not a pipeline step. No context inheritance. |
| P5: Navigator-First Architecture | N/A | TUI actions are user-initiated, not pipeline execution. |
| P6: Contracts at Every Handover | N/A | No pipeline step handovers involved. |
| P7: Relay via Dedicated Summarizer | N/A | TUI component, no context compaction. |
| P8: Ephemeral Workspaces | ✅ Pass | Uses existing workspace directories — read-only access for path derivation. |
| P9: Credentials Never Touch Disk | ✅ Pass | No credential handling. Subprocess inherits env vars via standard exec. |
| P10: Observable Progress | ✅ Pass | Actions are user-initiated with clear feedback (TUI suspend/resume, error messages). |
| P11: Bounded Recursion | N/A | No pipeline execution involved. |
| P12: Minimal Step State Machine | N/A | No step state transitions. |
| P13: Test Ownership | ✅ Pass | All new code will have tests; existing tests must continue to pass. |

No violations. No complexity tracking entries needed.

## Project Structure

### Documentation (this feature)

```
specs/258-tui-pipeline-actions/
├── plan.md              # This file
├── spec.md              # Feature specification
├── research.md          # Phase 0 research output
├── data-model.md        # Phase 1 data model output
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```
internal/tui/
├── pipeline_messages.go       # MODIFY — add ChatSessionEndedMsg, BranchCheckoutMsg, DiffViewEndedMsg, FinishedDetailActiveMsg
├── pipeline_detail.go         # MODIFY — add action key handlers, actionError state, chat/checkout/diff commands
├── pipeline_detail_test.go    # MODIFY — test action key handlers, error display, state gating
├── pipeline_detail_provider.go # MODIFY — extend FinishedDetail with WorkspacePath, BranchDeleted; add workspace derivation + git rev-parse
├── pipeline_detail_provider_test.go # MODIFY — test workspace path derivation, branch existence check
├── content.go                 # MODIFY — route new message types, emit FinishedDetailActiveMsg on Enter for finished items
├── content_test.go            # MODIFY — test finished detail focus, message routing
├── statusbar.go               # MODIFY — add finishedDetailActive state, finished detail hints
├── statusbar_test.go          # MODIFY — test FinishedDetailActiveMsg handling, hint text
├── app.go                     # MODIFY — forward FinishedDetailActiveMsg to status bar
├── app_test.go                # MODIFY — test FinishedDetailActiveMsg forwarding
```

**Structure Decision**: No new files needed. All changes extend existing files following established patterns from #252-#257. The feature adds action handlers to `pipeline_detail.go`, data provider extensions to `pipeline_detail_provider.go`, and message routing to `content.go`/`app.go`.

## Implementation Approach

### Phase 1: New Message Types and FinishedDetail Extensions

**Files**: `pipeline_messages.go`, `pipeline_detail_provider.go`, `pipeline_detail_provider_test.go`

1. Add new message types to `pipeline_messages.go`:
   - `ChatSessionEndedMsg{Err error}`
   - `BranchCheckoutMsg{BranchName string, Success bool, Err error}`
   - `DiffViewEndedMsg{Err error}`
   - `FinishedDetailActiveMsg{Active bool}`

2. Extend `FinishedDetail` struct in `pipeline_detail_provider.go`:
   - Add `WorkspacePath string` field
   - Add `BranchDeleted bool` field

3. Modify `FetchFinishedDetail()` in `DefaultDetailDataProvider`:
   - After existing data fetch, derive workspace path:
     - Import `internal/pipeline` for `sanitizeBranchName` access (or inline the logic)
     - Note: `sanitizeBranchName` is unexported in `internal/pipeline/context.go`
     - **Approach**: Inline a local `sanitizeBranch()` function in the TUI package to avoid exporting internal pipeline functions. The sanitization logic is simple (regex replace, collapse dashes, trim, limit 50 chars).
     - Construct candidate path: `.wave/workspaces/<RunID>/__wt_<sanitized_branch>/`
     - If candidate doesn't exist, try `filepath.Glob(".wave/workspaces/<RunID>/__wt_*")` and use first match
     - Validate with `os.Stat()` — set `WorkspacePath` if exists, leave empty if not
   - Check branch existence:
     - If `BranchName` is non-empty, run `exec.Command("git", "rev-parse", "--verify", branchName).Run()`
     - Non-zero exit → `BranchDeleted = true`

4. Tests for workspace path derivation:
   - Create temp dirs matching workspace convention, verify path resolution
   - Test branch existence check with real git repo in test (or mock)

### Phase 2: Action Key Handlers in Detail Model

**Files**: `pipeline_detail.go`, `pipeline_detail_test.go`

1. Add `actionError string` field to `PipelineDetailModel`.

2. Handle `tea.KeyMsg` in `Update()` when `paneState == stateFinishedDetail && focused`:
   - **Enter** (chat session):
     - Check `m.finishedDetail.WorkspacePath` is non-empty
     - If empty: set `m.actionError = "Workspace directory no longer exists — the worktree may have been cleaned up"`
     - If exists: return `tea.Exec(exec.Command("claude"), func(err error) tea.Msg { return ChatSessionEndedMsg{Err: err} })`
     - Set working directory: `cmd.Dir = m.finishedDetail.WorkspacePath`
   - **`b`** (branch checkout):
     - Check `!m.branchDeleted && m.finishedDetail != nil && m.finishedDetail.BranchName != ""`
     - If gated: no-op (return m, nil)
     - Return `tea.Cmd`: run `git checkout <branch>`, return `BranchCheckoutMsg`
   - **`d`** (diff view):
     - Check `!m.branchDeleted && m.finishedDetail != nil && m.finishedDetail.BranchName != ""`
     - If gated: no-op (return m, nil)
     - Return `tea.Exec(exec.Command("git", "diff", "main..."+branch), func(err error) tea.Msg { return DiffViewEndedMsg{Err: err} })`

3. Handle result messages:
   - **`ChatSessionEndedMsg`**: Re-fetch finished detail (data refresh), emit `GitRefreshTickMsg`
   - **`BranchCheckoutMsg`**: On success → clear `actionError`, emit `GitRefreshTickMsg`. On failure → set `actionError`
   - **`DiffViewEndedMsg`**: No-op (just resume TUI)

4. Clear `actionError` on any key press in `stateFinishedDetail` (before processing the key).

5. Update `renderFinishedDetail()`:
   - Accept `actionError` and `workspacePath` parameters (or access from detail struct)
   - When `actionError != ""`: render red error text instead of action hints
   - Faint `[d]` when `branchDeleted` (existing code only faints `[b]`) — C15 fix
   - Faint `[Enter]` when workspace path is empty

6. Tests:
   - Enter in `stateFinishedDetail` with valid workspace → returns `tea.Exec` command
   - Enter with empty workspace → sets `actionError`
   - `b` with valid branch → returns checkout command
   - `b` with deleted branch → no-op
   - `d` with valid branch → returns `tea.Exec` command
   - `d` with deleted branch → no-op
   - Action error clears on next key press
   - `BranchCheckoutMsg` success clears error
   - `BranchCheckoutMsg` failure sets error

### Phase 3: Content Model — Message Routing and Focus Integration

**Files**: `content.go`, `content_test.go`

1. Route new message types in `ContentModel.Update()`:
   - `ChatSessionEndedMsg` → forward to detail model
   - `BranchCheckoutMsg` → forward to detail model; on success, emit `GitRefreshTickMsg`
   - `DiffViewEndedMsg` → forward to detail model

2. Emit `FinishedDetailActiveMsg`:
   - When Enter focuses right pane on a finished item: emit `FinishedDetailActiveMsg{Active: true}`
   - When Esc returns to left pane: emit `FinishedDetailActiveMsg{Active: false}` (alongside existing `LiveOutputActiveMsg{Active: false}`)

3. Tests:
   - Enter on finished item focuses right pane and emits `FinishedDetailActiveMsg{Active: true}`
   - Esc from finished detail emits `FinishedDetailActiveMsg{Active: false}`
   - New message types forwarded to detail model

### Phase 4: Status Bar — Finished Detail Hints

**Files**: `statusbar.go`, `statusbar_test.go`

1. Add `finishedDetailActive bool` field to `StatusBarModel`.

2. Handle `FinishedDetailActiveMsg` in `Update()`.

3. Update hint rendering in `View()`:
   - Insert after `liveOutputActive` check, before generic right-pane:
   - When `finishedDetailActive && focusPane == FocusPaneRight`:
     - Show: `"[Enter] Chat  [b] Branch  [d] Diff  [Esc] Back"`

4. Tests:
   - `FinishedDetailActiveMsg{Active: true}` sets `finishedDetailActive = true`
   - Status bar renders finished detail hints when active
   - Hints priority: form > live output > finished detail > generic

### Phase 5: App Model — Message Forwarding

**Files**: `app.go`, `app_test.go`

1. Forward `FinishedDetailActiveMsg` to status bar:
   - Add `FinishedDetailActiveMsg` to the `switch msg.(type)` case alongside `FocusChangedMsg`, `FormActiveMsg`, `LiveOutputActiveMsg`

2. Tests:
   - `FinishedDetailActiveMsg` forwarded to status bar

### Phase 6: Integration and Rendering Fixes

**Files**: `pipeline_detail.go`, `pipeline_detail_test.go`

1. Update `renderFinishedDetail()` signature:
   - Add `actionError string` and `workspacePath string` parameters (or pass full detail struct)
   - Handle `actionError` rendering (red text replacing action hints)
   - Fix `[d]` fainting for consistency with `[b]` (C15)
   - Add `[Enter]` fainting when workspace path is empty

2. Update `updateViewportContent()` to pass new parameters to `renderFinishedDetail()`.

3. `branchDeleted` field on `PipelineDetailModel`:
   - Currently set from `PipelineSelectedMsg.BranchDeleted` — update to also use `FinishedDetail.BranchDeleted` when detail data arrives
   - Precedence: `FinishedDetail.BranchDeleted` (from git rev-parse) overrides `PipelineSelectedMsg.BranchDeleted` (currently always false)

4. Tests:
   - `[d]` fainted when branch deleted (C15 fix)
   - `[Enter]` fainted when workspace path empty
   - Action error rendered in red when set
   - Action error replaces action hints

## Complexity Tracking

_No constitution violations. No complexity tracking entries needed._
