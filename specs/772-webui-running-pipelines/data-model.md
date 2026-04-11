# Data Model: Expandable Running Pipelines Section

**Feature**: `772-webui-running-pipelines`  
**Phase**: 1 — Design & Contracts  
**Date**: 2026-04-11

## Overview

This feature adds a read-only UI section to the runs overview. No new persistent data types are
introduced. The implementation extends the existing `handleRunsPage` template data struct with
a pre-filtered slice of running runs and a count.

---

## Existing Entities (unchanged)

### RunSummary (`internal/webui/types.go:17`)

The central entity for all run list views. The running-pipelines section is a **filtered view**
of this type — no new struct required.

| Field | Type | Used by running section |
|-------|------|------------------------|
| `RunID` | `string` | Link href: `/runs/{RunID}` |
| `PipelineName` | `string` | Card title, filter match |
| `Status` | `string` | Always `"running"` in this section |
| `Progress` | `int` | Step progress bar (0–100) |
| `StepsCompleted` | `int` | `N/M` display in row 2 |
| `StepsTotal` | `int` | `N/M` display in row 2 |
| `Duration` | `string` | Duration in row 1 right |
| `FormattedStartedAt` | `string` | Timestamp in row 2 right |
| `InputPreview` | `string` | Optional input preview in row 1 |
| `LinkedURL` | `string` | Optional GitHub link in row 1 |
| `Models` | `[]string` | Model tier badge in row 2 |
| `TotalTokens` | `int` | Token count in row 2 |
| `ChildRuns` | `[]RunSummary` | Not rendered in running section (child runs excluded) |

### state.ListRunsOptions (`internal/state/types.go`)

Used to query the database for runs. The running-section query uses:

| Field | Value |
|-------|-------|
| `Status` | `"running"` (always, regardless of page status filter) |
| `PipelineName` | `pipelineFilter` from query param (respects FR-008) |
| `Limit` | `0` — no pagination for running section (unbounded, CL-002) |

---

## Template Data Extension

The anonymous `data` struct in `handleRunsPage` gains two fields:

```go
data := struct {
    ActivePage     string
    Runs           []RunSummary  // main list (existing)
    HasMore        bool
    NextCursor     string
    Pipelines      []string
    FilterStatus   string
    FilterPipeline string
    // NEW:
    RunningRuns  []RunSummary  // filtered to status=running, respects pipeline filter
    RunningCount int           // len(RunningRuns), pre-computed for header badge
}{...}
```

**Why `RunningRuns` is separate from `Runs`**:
- `Runs` reflects the user's status filter tab (may be "completed", "failed", etc.)
- `RunningRuns` always shows currently running pipelines regardless of tab
- Avoids template-side filtering (Go templates have no `filter` function)

**`RunningCount` is pre-computed** because Go templates cannot call `len()` on slices without
a custom template function. Pre-computing avoids adding a new template func.

---

## UI State Model

The running-pipelines section has **transient** (not persisted) UI state:

| State | Type | Initial Value | Storage |
|-------|------|---------------|---------|
| `expanded` | bool | `true` | DOM attribute `aria-expanded` on header |

The collapsed/expanded state is toggled by `toggleRunningSection()` in inline JS and reset to
`true` on every page load (FR-002). No `localStorage` involvement (contrast with sidebar nav
groups which persist collapse state).

---

## Empty-State Entity

When `RunningCount == 0`, the section renders an empty-state placeholder. No struct needed —
rendered directly in the template as a conditional block:

```html
{{if eq .RunningCount 0}}
<div class="rp-empty">
    <p>No pipelines running</p>
    <a href="/pipelines" class="rp-cta">Start a pipeline →</a>
</div>
{{else}}
... run cards ...
{{end}}
```

---

## CSS Classes (new, to be added to style.css)

| Class | Element | Purpose |
|-------|---------|---------|
| `.rp-section` | `<div>` wrapper | Container for the entire running section |
| `.rp-header` | `<div>` header | Clickable header with label, badge, chevron |
| `.rp-label` | `<span>` | "Running" label text |
| `.rp-badge` | `<span>` | Count badge (styled like existing `.badge`) |
| `.rp-chevron` | `<span>` | Collapse/expand chevron indicator (▾/▸) |
| `.rp-body` | `<div>` | Collapsible content area |
| `.rp-empty` | `<div>` | Empty-state placeholder |
| `.rp-cta` | `<a>` | CTA link in empty state |

**Design note**: Run cards inside `.rp-body` reuse `.wr-run`, `.wr-accent`, `.wr-body`, etc.
unchanged — the running section is visually consistent with the main list.

---

## Data Flow

```
GET /runs?pipeline=<name>&status=<tab>
         ↓
handleRunsPage()
    ├─ pipelineFilter = r.URL.Query().Get("pipeline")
    ├─ status = r.URL.Query().Get("status") || "all"
    │
    ├─ [NEW] runningOpts = ListRunsOptions{Status:"running", PipelineName: pipelineFilter}
    ├─ [NEW] runningRecs = s.store.ListRuns(runningOpts)
    ├─ [NEW] runningRuns = enrichRunSummaries(runningRecs)  // step progress, models, tokens
    │
    ├─ mainOpts = ListRunsOptions{Status: queryStatus, PipelineName: pipelineFilter, ...}
    ├─ runs = s.store.ListRuns(mainOpts)
    ├─ summaries = nestChildRuns(enrichRunSummaries(runs))
    │
    └─ render runs.html with data{RunningRuns: runningRuns, RunningCount: len, Runs: summaries, ...}
         ↓
    runs.html
    ├─ [NEW] rp-section (above wr-list)
    │   ├─ rp-header: "Running" label + count badge + chevron
    │   └─ rp-body: run cards or empty-state
    └─ wr-list (existing main list, unchanged)
```
