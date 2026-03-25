# Tasks: WebUI Changed-Files Browser with Diff Views

**Feature Branch**: `565-webui-diff-browser`
**Generated**: 2026-03-25
**Spec**: `specs/565-webui-diff-browser/spec.md`
**Plan**: `specs/565-webui-diff-browser/plan.md`

## Phase 1: Setup

- [X] T001 [P1] Add DiffSummary, FileSummary, and FileDiff Go types to `internal/webui/types.go`
  - Add `DiffSummary` struct: `Files []FileSummary`, `TotalFiles int`, `TotalAdditions int`, `TotalDeletions int`, `BaseBranch string`, `HeadBranch string`, `Available bool`, `Message string` (omitempty). JSON tags must match `specs/565-webui-diff-browser/contracts/diff-summary-api.json` schema
  - Add `FileSummary` struct: `Path string`, `Status string`, `Additions int`, `Deletions int`, `Binary bool`. Status values: "added", "modified", "deleted", "renamed"
  - Add `FileDiff` struct: `Path string`, `Status string`, `Additions int`, `Deletions int`, `Content string`, `Truncated bool`, `Size int`, `Binary bool`, `OldPath string` (omitempty). JSON tags must match `specs/565-webui-diff-browser/contracts/file-diff-api.json` schema
  - File: `internal/webui/types.go`

## Phase 2: Backend — Git Diff Engine

- [X] T002 [P1] Create `internal/webui/diff.go` with `resolveBaseBranch` function
  - Implement `resolveBaseBranch() (string, error)` that resolves the base branch for diff comparison
  - Resolution order: (1) `git symbolic-ref refs/remotes/origin/HEAD` → strip `refs/remotes/origin/` prefix, (2) check if `main` branch exists via `git rev-parse --verify main`, (3) check if `master` exists, (4) return error "no base branch could be determined"
  - Use `os/exec` to run git commands (existing pattern in `internal/webui/server.go:detectRepoSlug`)
  - File: `internal/webui/diff.go`

- [X] T003 [P1] [P] Add `computeDiffSummary` to `internal/webui/diff.go`
  - Implement `computeDiffSummary(baseBranch, headBranch string) (*DiffSummary, error)`
  - Run `git diff --numstat <base>...<head>` to get per-file additions/deletions
  - Run `git diff --name-status <base>...<head>` to get per-file change status (A/M/D/R)
  - Parse output into `[]FileSummary` sorted alphabetically by path (FR-007)
  - Detect binary files: `--numstat` shows `-\t-\t` for binary files
  - Compute totals: `TotalFiles`, `TotalAdditions`, `TotalDeletions`
  - Set `Available: true`, populate `BaseBranch` and `HeadBranch`
  - Handle git errors: branch not found → return `DiffSummary{Available: false, Message: "Branch deleted — diff unavailable"}`
  - Depends on: T001, T002
  - File: `internal/webui/diff.go`

- [X] T004 [P1] [P] Add `computeFileDiff` to `internal/webui/diff.go`
  - Implement `computeFileDiff(baseBranch, headBranch, filePath string) (*FileDiff, error)`
  - Validate `filePath`: reject paths containing `..` or starting with `/` (FR-013). Use `filepath.Clean` + `strings.Contains(cleanPath, "..")` pattern from `handlers_artifacts.go:46`
  - Run `git diff <base>...<head> -- <path>` to get unified diff content
  - Parse additions/deletions count from the diff output (count `+`/`-` prefixed lines)
  - Determine status from the diff: new file → "added", deleted file → "deleted", rename → "renamed", else "modified"
  - Implement truncation: if diff content exceeds 100KB (FR-005), truncate and set `Truncated: true`, `Size` to original byte count
  - Binary detection: if `git diff --numstat` shows `-\t-\t` for the file, set `Binary: true`, `Content: ""`
  - Depends on: T001, T002
  - File: `internal/webui/diff.go`

- [X] T005 [P1] [P] Create `internal/webui/diff_test.go` with unit tests
  - Test `resolveBaseBranch` with mock git repo (use `t.TempDir()` + `git init` + create branches)
  - Test `computeDiffSummary` with a temp repo containing added, modified, deleted, renamed, and binary files
  - Test `computeFileDiff` with: normal diff, large diff (truncation), binary file, path traversal rejection
  - Test edge cases: nonexistent branch returns `Available: false`, empty repo
  - Follow existing test patterns from `internal/webui/handlers_test.go`
  - Depends on: T002, T003, T004
  - File: `internal/webui/diff_test.go`

## Phase 3: User Story 1 + User Story 2 — API Handlers (P1)

- [X] T006 [P1] Create `internal/webui/handlers_diff.go` with diff API handlers
  - Implement `handleAPIDiffSummary(w, r)` for `GET /api/runs/{id}/diff`:
    - Extract `runID` via `r.PathValue("id")`, validate non-empty
    - Get `RunRecord` via `s.store.GetRun(runID)`, return 404 if not found
    - Validate `BranchName` is non-empty (FR-006), return `DiffSummary{Available: false, Message: "No branch associated with this run"}` if empty
    - Call `resolveBaseBranch()`, then `computeDiffSummary(base, run.BranchName)`
    - Return JSON via `writeJSON(w, http.StatusOK, summary)`
  - Implement `handleAPIDiffFile(w, r)` for `GET /api/runs/{id}/diff/{path...}`:
    - Extract `runID` via `r.PathValue("id")`, extract file path via `r.PathValue("path")`
    - Get `RunRecord`, validate `BranchName`
    - Sanitize `path` parameter: reject `..`, absolute paths (FR-013)
    - Call `resolveBaseBranch()`, then `computeFileDiff(base, run.BranchName, path)`
    - Return JSON via `writeJSON(w, http.StatusOK, fileDiff)`
  - Follow pattern from `handlers_artifacts.go` and `handlers_runs.go`
  - Depends on: T003, T004
  - File: `internal/webui/handlers_diff.go`

- [X] T007 [P1] Register diff API routes in `internal/webui/routes.go`
  - Add `mux.HandleFunc("GET /api/runs/{id}/diff", s.handleAPIDiffSummary)` in the API endpoints section
  - Add `mux.HandleFunc("GET /api/runs/{id}/diff/{path...}", s.handleAPIDiffFile)` using Go 1.22+ wildcard path matching
  - Place routes after existing `/api/runs/{id}/artifacts/{step}/{name}` route
  - Depends on: T006
  - File: `internal/webui/routes.go`

- [X] T008 [P1] [P] Create `internal/webui/handlers_diff_test.go` with handler tests
  - Use `testServer(t)` pattern from `handlers_test.go` to create test server with temp DB
  - Create a `RunRecord` with `BranchName` set, insert via state store
  - Test `GET /api/runs/{id}/diff`: success (200), run not found (404), empty BranchName (200 with `available: false`)
  - Test `GET /api/runs/{id}/diff/{path...}`: success (200), run not found (404), path traversal blocked (400/403)
  - Test error responses match `{"error": "..."}` format
  - Note: Full integration tests require a real git repo; handler tests can mock the git operations or use a temp git repo
  - Depends on: T006, T007
  - File: `internal/webui/handlers_diff_test.go`

## Phase 4: User Story 1 + User Story 2 — Frontend File List & Unified Diff (P1)

- [X] T009 [P1] Create `internal/webui/static/diff-viewer.js` with DiffViewer class and file list
  - Create `DiffViewer` class with constructor taking `runID` and container element
  - Implement `loadFileList(runID)`: fetch `GET /api/runs/{runID}/diff`, parse JSON response
  - Implement `renderFileList(summary)`: render flat file list sorted alphabetically, each entry shows relative path and status icon (A green, M yellow, D red) with add/delete line counts
  - Implement file click handler: on click, call `loadFileDiff(runID, path)` to lazy-load single file diff
  - Implement `loadFileDiff(runID, path)`: fetch `GET /api/runs/{runID}/diff/{path}`, parse JSON
  - Handle unavailable diff: if `summary.available === false`, show `summary.message` text
  - Handle binary files: show "Binary file changed" indicator instead of diff content
  - Use vanilla JavaScript ES5+ (no ES6 modules) matching existing `app.js`, `log-viewer.js` patterns
  - Depends on: T007
  - File: `internal/webui/static/diff-viewer.js`

- [X] T010 [P1] Add unified diff parser and renderer to `internal/webui/static/diff-viewer.js`
  - Implement `parseDiff(content)`: parse unified diff text into structured hunks
    - Split on `@@` markers to identify hunks with line number ranges
    - Classify each line as addition (`+`), deletion (`-`), or context (` `)
    - Track old and new line numbers through each hunk
  - Implement `renderUnified(parsedDiff)`: render unified diff view
    - Show old and new line numbers in gutter columns
    - Green background for addition lines, red for deletion lines
    - Hunk headers (`@@`) displayed as separator rows
    - Use `<pre>` with monospace font for code content
  - Depends on: T009
  - File: `internal/webui/static/diff-viewer.js`

- [X] T011 [P1] Add diff panel HTML section to `internal/webui/templates/run_detail.html`
  - Add a new `<div class="card" id="diff-panel">` section inside `.run-detail-main`, positioned after the Steps card and before the Events card
  - Panel contains: `<h2>Changed Files</h2>`, a `<div id="diff-file-list">` for the file list, and a `<div id="diff-viewer-content">` for the diff content
  - Only render the panel when `{{.Run.BranchName}}` is non-empty
  - Add `<script src="/static/diff-viewer.js"></script>` to the `{{define "scripts"}}` block
  - Initialize DiffViewer in the script block: `var diffViewer = new DiffViewer('{{.Run.RunID}}', document.getElementById('diff-panel'));`
  - Call `diffViewer.loadFileList()` on page load
  - Depends on: T009, T010
  - File: `internal/webui/templates/run_detail.html`

- [X] T012 [P1] Add diff viewer CSS to `internal/webui/static/style.css`
  - Add styles for `#diff-panel` layout: split into file list (left, ~250px) and diff content (right, flex)
  - File list styles: `.diff-file-item` with hover highlight, `.diff-status-A` (green), `.diff-status-M` (yellow), `.diff-status-D` (red)
  - Diff content styles: `.diff-line` with `.diff-add` (green bg), `.diff-del` (red bg), `.diff-context` (no bg)
  - Line number gutter: `.diff-line-num` in muted color, fixed width
  - Hunk header: `.diff-hunk-header` with distinct background
  - Code content: monospace `font-family`, appropriate `font-size`
  - Responsive: on narrow screens (<768px), stack file list above diff content
  - Depends on: T011
  - File: `internal/webui/static/style.css`

## Phase 5: User Story 3 — View Mode Switching (P2)

- [X] T013 [P2] [P] Add side-by-side diff renderer to `internal/webui/static/diff-viewer.js`
  - Implement `renderSideBySide(parsedDiff)`: render two-column layout
  - Left column shows old file content with line numbers, right column shows new content
  - Align corresponding lines: context lines appear on both sides, deletions on left only, additions on right only
  - Use CSS `display: grid` or `table` for column alignment
  - Color coding: red for left-only (deleted), green for right-only (added)
  - Depends on: T010
  - File: `internal/webui/static/diff-viewer.js`

- [X] T014 [P2] [P] Add raw before/after view renderer to `internal/webui/static/diff-viewer.js`
  - Implement `renderRaw(parsedDiff, mode)` where mode is "before" or "after"
  - "Before" mode: reconstruct the old file content by taking context lines and deletion lines
  - "After" mode: reconstruct the new file content by taking context lines and addition lines
  - Display complete file content with line numbers, no diff markers
  - Add a sub-toggle (Before | After) that defaults to "After" — sub-toggle state is NOT persisted in localStorage
  - Depends on: T010
  - File: `internal/webui/static/diff-viewer.js`

- [X] T015 [P2] Add view mode toggle UI and localStorage persistence to `internal/webui/static/diff-viewer.js`
  - Add toggle buttons above the diff content area: "Unified" | "Side-by-side" | "Raw"
  - Style active toggle with distinct background color
  - On toggle click: re-render the current file diff in the selected mode using the already-fetched content
  - Persist selected mode to `localStorage` key `"wave-diff-view-mode"` (FR-012)
  - On page load: read `localStorage` for saved mode, default to `"unified"` if absent
  - CSS for toggle bar: `.diff-view-toggle` with `.diff-toggle-btn` and `.diff-toggle-active`
  - Depends on: T013, T014
  - File: `internal/webui/static/diff-viewer.js`, `internal/webui/static/style.css`

## Phase 6: User Story 2 — Syntax Highlighting (P1)

- [X] T016 [P1] Add CSS-based regex syntax highlighting to `internal/webui/static/diff-viewer.js`
  - Implement `highlightSyntax(code, language)` function
  - Detect language from file extension: `.go`, `.js`, `.ts`, `.yaml`/`.yml`, `.json`, `.md`, `.html`, `.css`, `.sql`, `.sh`/`.bash`
  - Token classes and regex patterns per language category:
    - **Comments**: `//`, `/* */`, `#` (line/block) → `.syntax-comment` (gray/italic)
    - **Strings**: single-quoted, double-quoted, backtick templates → `.syntax-string` (green)
    - **Keywords**: language-specific keyword lists → `.syntax-keyword` (blue/bold)
    - **Numbers**: integer and float literals → `.syntax-number` (orange)
  - Apply highlighting to each line's text content before inserting into DOM
  - HTML-escape code content BEFORE applying highlight spans to prevent XSS
  - Add CSS classes to `style.css`: `.syntax-comment`, `.syntax-string`, `.syntax-keyword`, `.syntax-number`
  - Depends on: T010
  - File: `internal/webui/static/diff-viewer.js`, `internal/webui/static/style.css`

## Phase 7: User Story 4 — Virtualization (P3)

- [X] T017 [P3] Implement viewport-based virtualization for large diffs in `internal/webui/static/diff-viewer.js`
  - Only activate virtualization when diff exceeds 500 lines (FR-010)
  - Calculate total container height: `lineCount * LINE_HEIGHT` (fixed monospace line height)
  - Render only visible lines plus a buffer of ±50 lines above/below viewport
  - Use a spacer element (invisible `div` with calculated height) for correct scrollbar behavior
  - On scroll: recalculate visible range via `requestAnimationFrame`, replace rendered lines
  - Container: `overflow-y: auto` with absolute-positioned content
  - Ensure line numbers remain correct when viewport shifts
  - Must work with all three view modes (unified, side-by-side, raw)
  - Depends on: T010, T013, T014
  - File: `internal/webui/static/diff-viewer.js`

## Phase 8: Polish & Cross-Cutting Concerns

- [X] T018 [P1] Add summary bar showing file count and line changes to diff panel
  - Display total changed files, total additions (green `+N`), total deletions (red `-N`) from `DiffSummary` response
  - Position above the file list panel, styled similar to `.run-summary-bar`
  - Update summary when file list loads or refreshes
  - FR-011 compliance
  - Depends on: T009
  - File: `internal/webui/static/diff-viewer.js`, `internal/webui/static/style.css`

- [X] T019 [P1] Handle all edge cases in both backend and frontend
  - Backend (`handlers_diff.go`/`diff.go`):
    - Run not found → 404 with `{"error": "run not found"}`
    - Empty `BranchName` (FR-006) → 200 with `{available: false, message: "No branch associated with this run"}`
    - Branch deleted → 200 with `{available: false, message: "Branch deleted — diff unavailable"}`
    - No base branch resolvable → 200 with `{available: false, message: "..."}`
    - Detached HEAD with no branch ref → structured error
    - Binary file → `{binary: true, content: ""}`
    - Run still in progress → return current partial diff with note
  - Frontend (`diff-viewer.js`):
    - `available: false` → show message text, hide file list and diff content
    - Binary file clicked → show "Binary file changed" indicator
    - Truncated file → show "Truncated — file too large (N bytes)" indicator
    - Empty file list → show "No files changed" message
    - Fetch error → show error message with retry option
  - Depends on: T006, T009
  - File: `internal/webui/handlers_diff.go`, `internal/webui/diff.go`, `internal/webui/static/diff-viewer.js`

- [X] T020 [P2] Wire SSE events to refresh diff file list during running pipelines
  - In `run_detail.html` script block: when SSE receives a `step_completed` event, call `diffViewer.loadFileList()` to refresh
  - Only refresh when run status is "running"
  - Debounce rapid successive refreshes (wait 2s between refreshes)
  - Depends on: T009, T011
  - File: `internal/webui/templates/run_detail.html`

## Dependency Graph

```
T001 (types)
├── T002 (resolveBaseBranch)
│   ├── T003 [P] (computeDiffSummary)
│   │   └── T006 (API handlers)
│   │       ├── T007 (register routes)
│   │       │   ├── T008 [P] (handler tests)
│   │       │   └── T009 (DiffViewer + file list)
│   │       │       ├── T010 (unified parser/renderer)
│   │       │       │   ├── T011 (HTML integration)
│   │       │       │   │   └── T012 (CSS)
│   │       │       │   ├── T013 [P] (side-by-side)
│   │       │       │   ├── T014 [P] (raw view)
│   │       │       │   ├── T015 (view mode toggle) ← T013, T014
│   │       │       │   ├── T016 (syntax highlighting)
│   │       │       │   └── T017 (virtualization) ← T013, T014
│   │       │       ├── T018 (summary bar)
│   │       │       └── T020 (SSE refresh) ← T011
│   │       └── T019 (edge cases)
│   └── T004 [P] (computeFileDiff)
│       └── T006
└── T005 [P] (diff unit tests) ← T003, T004
```

## Summary

| Phase | Tasks | Priority | Description |
|-------|-------|----------|-------------|
| 1: Setup | T001 | P1 | Go type definitions |
| 2: Backend Engine | T002–T005 | P1 | Git diff computation, sanitization, tests |
| 3: API Handlers | T006–T008 | P1 | HTTP handlers, routes, handler tests |
| 4: Frontend Core | T009–T012 | P1 | File list, unified diff, HTML, CSS |
| 5: View Modes | T013–T015 | P2 | Side-by-side, raw, toggle |
| 6: Highlighting | T016 | P1 | Regex syntax highlighting |
| 7: Virtualization | T017 | P3 | Large diff viewport rendering |
| 8: Polish | T018–T020 | P1/P2 | Summary bar, edge cases, SSE |

**Total tasks**: 20
**Parallel opportunities**: 6 (T003+T004, T005 with T003+T004, T008 with T009, T013+T014, T016+T017)
