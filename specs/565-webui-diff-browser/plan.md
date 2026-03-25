# Implementation Plan: WebUI Changed-Files Browser with Diff Views

**Branch**: `565-webui-diff-browser` | **Date**: 2026-03-25 | **Spec**: `specs/565-webui-diff-browser/spec.md`
**Input**: Feature specification from `/specs/565-webui-diff-browser/spec.md`

## Summary

Add a changed-files browser with diff views to the Wave webui run detail page. Two new API endpoints (`GET /api/runs/{id}/diff` for file summary, `GET /api/runs/{id}/diff/{path...}` for single-file diff) compute diffs by shelling out to `git diff` against the run's branch. The frontend renders a file list panel and a multi-mode diff viewer (unified, side-by-side, raw) using vanilla JavaScript with CSS-based syntax highlighting and viewport-based virtualization for large diffs.

## Technical Context

**Language/Version**: Go 1.25+ (backend), vanilla JavaScript ES5+ (frontend)
**Primary Dependencies**: `os/exec` (git subprocess), existing `internal/webui` package, existing `internal/state` package
**Storage**: SQLite (existing — `RunRecord.BranchName` read-only), filesystem (git repo)
**Testing**: `go test` with `httptest` (existing pattern in `handlers_test.go`)
**Target Platform**: Linux/macOS server, all modern browsers
**Project Type**: Single Go binary with embedded web assets
**Performance Goals**: <3s for 100-file summary (SC-001), <1s for 100KB file diff (SC-002), <500ms file list render (SC-003)
**Constraints**: <200MB browser memory for 5000-line diffs (SC-004), no npm/build step, no external JS libraries
**Scale/Scope**: Pipeline runs typically touch 5-30 files; edge case up to 100+ files

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new dependencies. Uses `os/exec` for git (already a runtime dependency). Embedded assets via `embed.FS`. |
| P2: Manifest as SSOT | PASS | No manifest changes required. Feature is entirely webui. |
| P3: Persona-Scoped Execution | N/A | Not a persona feature. |
| P4: Fresh Memory | N/A | Not a pipeline step feature. |
| P5: Navigator-First | N/A | Not a pipeline feature. |
| P6: Contracts at Handover | N/A | Not a pipeline feature. |
| P7: Relay | N/A | Not applicable. |
| P8: Ephemeral Workspaces | PASS | Reads from git repo (read-only). No workspace mutation. |
| P9: Credentials Never Touch Disk | PASS | No credentials involved. Git operations use local repo. |
| P10: Observable Progress | PASS | HTTP request logging via existing middleware. |
| P11: Bounded Recursion | N/A | Not applicable. |
| P12: Minimal Step State Machine | N/A | Not a pipeline feature. |
| P13: Test Ownership | PASS | All new code gets handler tests following existing `handlers_test.go` patterns. |

**Post-Phase 1 Re-check**: All principles still PASS. No violations.

## Project Structure

### Documentation (this feature)

```
specs/565-webui-diff-browser/
├── plan.md              # This file
├── research.md          # Phase 0 — technology decisions and patterns
├── data-model.md        # Phase 1 — entity definitions and data flow
├── contracts/           # Phase 1 — API response schemas
│   ├── diff-summary-api.json
│   └── file-diff-api.json
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
internal/webui/
├── handlers_diff.go         # NEW — diff API handlers (summary + single-file)
├── handlers_diff_test.go    # NEW — handler tests
├── diff.go                  # NEW — git diff computation logic (os/exec)
├── diff_test.go             # NEW — diff computation unit tests
├── types.go                 # MODIFIED — add DiffSummary, FileSummary, FileDiff types
├── routes.go                # MODIFIED — register new API routes
├── templates/
│   └── run_detail.html      # MODIFIED — add diff panel section
├── static/
│   ├── diff-viewer.js       # NEW — diff parser, renderer, syntax highlighting, virtualization
│   └── style.css            # MODIFIED — add diff viewer CSS
```

**Structure Decision**: All new code lives in the existing `internal/webui/` package, following the established pattern of one handler file per domain (e.g., `handlers_artifacts.go`, `handlers_runs.go`). The git diff logic gets its own file (`diff.go`) to separate git subprocess concerns from HTTP handler concerns. Frontend code follows the existing pattern of one JS file per feature module (`log-viewer.js`, `dag.js`, `sse.js`).

## Implementation Phases

### Phase A: Backend — Git Diff Engine (`diff.go`)

**Files**: `internal/webui/diff.go`, `internal/webui/diff_test.go`

1. Implement `resolveBaseBranch()` — resolves base branch via `git symbolic-ref` → `main` → `master` fallback
2. Implement `computeDiffSummary(baseBranch, headBranch string)` — runs `git diff --stat --numstat base...head`, parses output into `DiffSummary`
3. Implement `computeFileDiff(baseBranch, headBranch, filePath string)` — runs `git diff base...head -- path`, returns `FileDiff`
4. Implement path sanitization: reject `..`, absolute paths, validate against `--stat` output
5. Implement truncation: cap single-file diff at 100KB, set `Truncated` flag
6. Binary detection: parse `--numstat` output (binary files show `-` for counts)

### Phase B: Backend — API Handlers (`handlers_diff.go`)

**Files**: `internal/webui/handlers_diff.go`, `internal/webui/handlers_diff_test.go`, `internal/webui/types.go`, `internal/webui/routes.go`

1. Add `DiffSummary`, `FileSummary`, `FileDiff` types to `types.go`
2. Implement `handleAPIDiffSummary(w, r)` for `GET /api/runs/{id}/diff`:
   - Get `RunRecord` from state store
   - Validate `BranchName` is non-empty (FR-006)
   - Call `computeDiffSummary()`
   - Handle errors: branch deleted, no base branch, git errors
3. Implement `handleAPIDiffFile(w, r)` for `GET /api/runs/{id}/diff/{path...}`:
   - Get `RunRecord`, validate branch
   - Extract `{path...}` parameter, sanitize (FR-013)
   - Call `computeFileDiff()`
   - Apply truncation (FR-005)
4. Register routes in `routes.go`
5. Write handler tests following `handlers_runs_test.go` pattern (SC-007)

### Phase C: Frontend — Diff Viewer (`diff-viewer.js`)

**Files**: `internal/webui/static/diff-viewer.js`

1. Implement `DiffViewer` class:
   - `loadFileList(runID)` — fetches `/api/runs/{id}/diff`, renders file list panel
   - `loadFileDiff(runID, path)` — fetches `/api/runs/{id}/diff/{path}`, renders diff
   - `renderUnified(content)` — parse + render unified diff view
   - `renderSideBySide(content)` — parse + render side-by-side view
   - `renderRaw(content, mode)` — render before/after raw file content
2. Implement diff parser: split unified diff into hunks, classify lines
3. Implement syntax highlighter: regex-based token matching for Go, JS, TS, YAML, JSON, MD, HTML, CSS, SQL, Shell
4. Implement viewport virtualization for >500 line diffs
5. Implement view mode persistence via `localStorage` (FR-012)

### Phase D: Frontend — UI Integration

**Files**: `internal/webui/templates/run_detail.html`, `internal/webui/static/style.css`

1. Add diff panel section to `run_detail.html` (below steps, above events)
2. Add CSS for file list, diff viewer, syntax highlighting, view mode toggles
3. Initialize `DiffViewer` in the run detail page script block
4. Wire up SSE updates for running pipelines (refresh file list on step completion)

## Complexity Tracking

_No constitution violations._

| Item | Notes |
|------|-------|
| No new packages | All code in `internal/webui/`, no new Go modules |
| No build step | Vanilla JS, CSS embedded via `embed.FS` |
| No external deps | Git subprocess via `os/exec`, existing pattern |
