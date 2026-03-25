# Feature Specification: WebUI Changed-Files Browser with Diff Views

**Feature Branch**: `565-webui-diff-browser`
**Created**: 2026-03-25
**Status**: Draft
**Input**: User description: "https://github.com/re-cinq/wave/issues/565"

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Changed Files for a Pipeline Run (Priority: P1)

As a developer reviewing a pipeline run, I want to see a list of all files changed during the run so that I can quickly understand the scope of changes without leaving the Wave dashboard.

**Why this priority**: This is the foundational capability — without a file list, neither the diff viewer nor any downstream feature can function. It closes the largest gap compared to GitHub Actions.

**Independent Test**: Can be fully tested by navigating to any completed run with a valid branch, and verifying the changed-file list appears with correct add/modify/delete status indicators.

**Acceptance Scenarios**:

1. **Given** a completed pipeline run with an existing branch, **When** I navigate to the run detail page, **Then** I see a file tree sidebar listing all changed files with status icons (A for added, M for modified, D for deleted) and a summary showing total file count and lines changed.
2. **Given** a completed pipeline run, **When** I view the changed files list, **Then** each file entry shows its relative path and change status, and the list is sorted alphabetically by path.
3. **Given** a pipeline run where the branch has been deleted, **When** I navigate to the run detail page, **Then** I see a clear message: "Branch deleted — diff unavailable" instead of the file list, with no error or broken UI.

---

### User Story 2 - View Unified (Inline) Diff for a File (Priority: P1)

As a developer, I want to see the unified diff for any changed file so that I can review exactly what a pipeline persona wrote or modified.

**Why this priority**: The unified diff view is the most common and essential diff format — it provides the core review capability that makes the file browser useful.

**Independent Test**: Can be tested by clicking any file in the changed-files list and verifying the unified diff renders with correct additions (green) and deletions (red), with line numbers.

**Acceptance Scenarios**:

1. **Given** a changed file list is visible, **When** I click on a file entry, **Then** a diff viewer panel displays the unified diff for that file with additions highlighted in green and deletions highlighted in red.
2. **Given** a unified diff is displayed, **When** the diff contains more than 500 lines, **Then** the content is virtualized (rendered incrementally) so the browser does not freeze.
3. **Given** I am viewing a diff for a Go, JavaScript, YAML, JSON, or Markdown file, **When** the diff renders, **Then** the code has syntax highlighting appropriate to the file type.

---

### User Story 3 - Switch Between Diff View Modes (Priority: P2)

As a developer, I want to toggle between side-by-side diff, unified diff, and raw before/after views so that I can review changes in whichever format is most readable for the file at hand.

**Why this priority**: Different change types are easier to review in different formats — renaming is clearest side-by-side, large additions are clearest in raw "after" view.

**Independent Test**: Can be tested by opening any file diff, toggling between the three modes, and verifying each renders correctly with the same file content.

**Acceptance Scenarios**:

1. **Given** a diff viewer panel is open, **When** I click the "Side-by-side" toggle, **Then** the view switches to show the old version on the left and the new version on the right, with corresponding lines aligned.
2. **Given** a diff viewer panel is open, **When** I click the "Unified" toggle, **Then** the view switches to a standard unified diff format with additions and deletions interleaved.
3. **Given** a diff viewer panel is open, **When** I click "Before" or "After" in the raw view toggle, **Then** the viewer shows the complete file content for the selected version without diff markers.

---

### User Story 4 - Fetch Single-File Diff for Large Repos (Priority: P3)

As a developer working in a large repository, I want to fetch diffs one file at a time so that I do not have to wait for the entire diff to load before I can start reviewing.

**Why this priority**: For pipeline runs that touch many files or produce large diffs, fetching everything upfront degrades performance. Per-file fetching keeps the UI responsive.

**Independent Test**: Can be tested by navigating to a run with many changed files, verifying the file list loads quickly, and then clicking individual files to see their diffs load on demand.

**Acceptance Scenarios**:

1. **Given** a pipeline run that changed 50+ files, **When** I open the run detail page, **Then** the file list loads within 2 seconds and individual file diffs are fetched only when I click on them.
2. **Given** a single file diff exceeds the configurable size limit, **When** I view the file, **Then** the diff is truncated with a "Truncated — file too large" indicator and a total size display.

---

### Edge Cases

- What happens when `RunRecord.BranchName` is empty or was never set? The API returns a clear "No branch associated with this run" message and the UI shows this gracefully.
- What happens when the git repository is in a detached HEAD state with no branch ref? The diff endpoint returns an appropriate error rather than crashing.
- What happens when a file was renamed (appears as both deleted and added)? Both entries appear in the file list with their respective statuses; git's rename detection is not required.
- What happens when a binary file was changed? The file list shows the file with a "Binary file changed" indicator instead of a text diff.
- What happens when the diff endpoint is called for a run that is still in progress? The endpoint returns the current diff state (partial changes) with a note that the run is still active.
- What happens when `main` branch does not exist (non-standard base branch)? The diff computation falls back to the repository's default branch or the merge-base of the run branch.
- What happens when the run ID does not exist? The API returns 404 with the standard `{"error": "..."}` response format.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST expose an API endpoint that returns a list of changed files for a given pipeline run, including each file's path, change status (added, modified, deleted), and unified diff content.
- **FR-002**: System MUST expose an API endpoint that returns the diff for a single file within a pipeline run, to support lazy-loading of individual file diffs.
- **FR-003**: System MUST compute diffs by comparing the run's branch against the base branch using git's three-dot diff syntax (`git diff main...branch`).
- **FR-004**: System MUST return a structured error response when the run's branch has been deleted or the workspace is no longer available, rather than failing silently or returning an HTTP 500.
- **FR-005**: System MUST truncate individual file diffs that exceed a configurable size limit (default: 100KB of diff text) and include a `truncated` flag in the response.
- **FR-006**: System MUST validate that the `BranchName` field is populated in `RunRecord` before attempting diff computation — if empty, return an informative error.
- **FR-007**: The WebUI run detail page MUST display a file tree sidebar showing changed files with visual status indicators (A/M/D icons or color coding).
- **FR-008**: The WebUI MUST provide a diff viewer panel that supports three view modes: unified (inline), side-by-side, and raw (before/after).
- **FR-009**: The WebUI MUST apply syntax highlighting to diff content for common file types: Go, JavaScript, TypeScript, YAML, JSON, Markdown, HTML, CSS, SQL, Shell.
- **FR-010**: The WebUI MUST virtualize rendering of diffs exceeding 500 lines to prevent browser performance degradation.
- **FR-011**: The WebUI MUST display a summary bar showing the total number of changed files and net lines added/deleted.
- **FR-012**: The WebUI MUST persist the user's preferred diff view mode in `localStorage` so it is retained across page loads.
- **FR-013**: System MUST sanitize file paths in diff output to prevent path traversal — only files within the repository root are included.

### Key Entities

- **DiffSummary**: Represents the aggregate diff for a run — contains a list of changed files with metadata (file count, total additions, total deletions) and availability status.
- **FileDiff**: Represents the diff for a single file — contains file path, change status (added/modified/deleted), unified diff content, line counts (additions/deletions), truncation flag, and file size.
- **DiffViewMode**: The user's selected viewing mode — one of `unified`, `side-by-side`, `before`, or `after`. Persisted in browser local storage.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: The changed-files API endpoint responds within 3 seconds for runs with up to 100 changed files.
- **SC-002**: The single-file diff API endpoint responds within 1 second for files up to 100KB of diff content.
- **SC-003**: The file tree sidebar renders within 500ms of the API response arriving, for up to 200 files.
- **SC-004**: The diff viewer does not cause the browser tab to exceed 200MB of memory when displaying a 5,000-line diff.
- **SC-005**: All three diff view modes (unified, side-by-side, raw) render correctly for files in each supported language.
- **SC-006**: Graceful degradation messages appear correctly in 100% of cases where the branch or workspace is unavailable.
- **SC-007**: All new API endpoints have corresponding handler tests covering success, error, and edge-case paths.
