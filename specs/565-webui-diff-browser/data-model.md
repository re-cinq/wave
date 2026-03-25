# Data Model: WebUI Changed-Files Browser with Diff Views

**Feature Branch**: `565-webui-diff-browser`
**Date**: 2026-03-25

## Entities

### DiffSummary

Represents the aggregate diff metadata for a pipeline run. Returned by `GET /api/runs/{id}/diff`.

```go
// DiffSummary represents the aggregate changed-file list for a pipeline run.
type DiffSummary struct {
    Files          []FileSummary `json:"files"`
    TotalFiles     int           `json:"total_files"`
    TotalAdditions int           `json:"total_additions"`
    TotalDeletions int           `json:"total_deletions"`
    BaseBranch     string        `json:"base_branch"`
    HeadBranch     string        `json:"head_branch"`
    Available      bool          `json:"available"`
    Message        string        `json:"message,omitempty"`
}
```

### FileSummary

Represents metadata for a single changed file (no diff content — that's lazy-loaded).

```go
// FileSummary represents a single changed file in the diff summary.
type FileSummary struct {
    Path      string `json:"path"`
    Status    string `json:"status"`    // "added", "modified", "deleted", "renamed"
    Additions int    `json:"additions"`
    Deletions int    `json:"deletions"`
    Binary    bool   `json:"binary"`
}
```

**Status values**:
- `added` — file is new (maps to git `A`)
- `modified` — file content changed (maps to git `M`)
- `deleted` — file removed (maps to git `D`)
- `renamed` — file moved (maps to git `R`)

### FileDiff

Represents the full diff content for a single file. Returned by `GET /api/runs/{id}/diff/{path...}`.

```go
// FileDiff represents the diff content for a single file.
type FileDiff struct {
    Path      string `json:"path"`
    Status    string `json:"status"`
    Additions int    `json:"additions"`
    Deletions int    `json:"deletions"`
    Content   string `json:"content"`   // unified diff text
    Truncated bool   `json:"truncated"`
    Size      int    `json:"size"`      // size of diff content in bytes
    Binary    bool   `json:"binary"`
    OldPath   string `json:"old_path,omitempty"` // for renames
}
```

### DiffViewMode (Frontend Only)

Not a Go type — managed entirely in JavaScript and `localStorage`.

```
Values: "unified" | "side-by-side" | "raw"
localStorage key: "wave-diff-view-mode"
Default: "unified"
```

## Relationships

```
RunRecord (state.types.go)
  └── BranchName string  ── used to compute ──► DiffSummary
                                                   └── Files []FileSummary
                                                         └── (on click) ──► FileDiff
```

## Data Flow

1. **Run detail page loads** → existing `handleRunDetailPage` populates run data including `BranchName`
2. **Client JS fetches** `GET /api/runs/{id}/diff` → backend reads `RunRecord.BranchName`, runs `git diff --stat --numstat`, returns `DiffSummary`
3. **User clicks file** → client JS fetches `GET /api/runs/{id}/diff/{path}` → backend runs `git diff <base>...<branch> -- <path>`, returns `FileDiff`
4. **View mode toggle** → client-side only, re-renders the already-fetched `FileDiff.Content` in the selected mode

## Backend Internals (Not Exposed)

### baseBranchResolver

Internal helper to determine the base branch for diff comparison.

```
Resolution order:
1. git symbolic-ref refs/remotes/origin/HEAD → strip "refs/remotes/origin/" prefix
2. Check if "main" branch exists locally
3. Check if "master" branch exists locally
4. Return error: "no base branch could be determined"
```

### Diff size limits

| Limit | Value | Source |
|-------|-------|--------|
| Max single-file diff | 100KB | FR-005, configurable |
| Virtualization threshold | 500 lines | FR-010, client-side |
| Memory budget | 200MB tab | SC-004, enforced by virtualization |

## File Placement

All new Go types live in `internal/webui/types.go` alongside existing `RunSummary`, `StepDetail`, etc. No new packages needed — this is a webui feature that uses `os/exec` for git commands.

All new JS lives in `internal/webui/static/diff-viewer.js` — a new static asset loaded only on the run detail page.

All new CSS lives in the existing `internal/webui/static/style.css` — appended to the bottom.
