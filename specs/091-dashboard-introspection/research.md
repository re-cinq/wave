# Research: Dashboard Inspection, Rendering, Statistics & Run Introspection

**Branch**: `091-dashboard-introspection` | **Date**: 2026-02-14

## Phase 0 — Research Findings

### R-001: Client-Side Markdown Parsing Library

**Decision**: Custom minimal markdown parser (~3 KB gzipped)

**Rationale**: The spec resolves this in C-001 — client-side parser targeting only the subset needed (headings, lists, code blocks, emphasis, links, tables). The existing JS bundle is ~15 KB raw (app.js 4.6 KB + dag.js 1 KB + sse.js 9.2 KB), well under the 50 KB gzipped budget. A custom parser avoids pulling in `marked.js` (~25 KB minified) or `markdown-it` (~45 KB minified) which would consume too much of the budget.

**Alternatives Rejected**:
- `marked.js` (~25 KB min, ~8 KB gzipped) — exceeds the safe margin given the budget constraint and multiple other new JS modules needed
- `markdown-it` (~45 KB min) — way over budget
- Server-side Go rendering via `goldmark` — violates FR-009 (no additional network requests for toggle)

**Implementation Approach**: Write a ~200-line JS parser supporting:
- `# ## ### ####` headings → `<h1>` through `<h4>`
- `- *` unordered lists → `<ul><li>`
- `1.` ordered lists → `<ol><li>`
- `` ``` `` code blocks → `<pre><code>`
- `` ` `` inline code → `<code>`
- `**bold**` `*italic*` emphasis
- `[text](url)` links
- `| col | col |` tables → `<table>`

All output is sanitized — no raw HTML passthrough, text content is escaped.

---

### R-002: Client-Side Syntax Highlighting

**Decision**: Custom regex-based tokenizer (~2 KB gzipped) using CSS classes

**Rationale**: Per C-002, a custom tokenizer for the known finite set of languages (Go, YAML, JSON, Markdown, JS, CSS, HTML, SQL, Shell) avoids heavy libraries. The tokenizer assigns CSS classes (`tok-key`, `tok-str`, `tok-num`, `tok-comment`, `tok-bool`, `tok-kw`) and the existing `style.css` handles colors for both themes.

**Alternatives Rejected**:
- Prism.js (~16 KB min core + language packs) — overkill for 9 languages
- highlight.js (~50 KB+) — exceeds budget entirely
- Server-side Go highlighting — violates toggle requirements (needs round-trip)

**Implementation Approach**: A single `highlight(code, language)` function that:
1. Dispatches to language-specific tokenizers (YAML, JSON, Go, SQL, Shell, JS, CSS, HTML, Markdown)
2. Each tokenizer is a list of regex → class-name rules applied in order
3. Returns HTML with `<span class="tok-*">` wrappers
4. All content is escaped before tokenization to prevent XSS

---

### R-003: Statistics Query Architecture

**Decision**: Server-side SQL aggregation via new `StateStore` methods

**Rationale**: Per C-005, server-side aggregation is the correct approach. SQLite handles `GROUP BY`/`COUNT`/`AVG` efficiently for up to 10K runs. The existing `GetStepPerformanceStats` demonstrates this pattern already. Client-side aggregation would require fetching all `RunRecord` objects which violates NFR-002.

**New StateStore Methods Required**:
1. `GetRunStatistics(since time.Time) (*RunStatistics, error)` — aggregate run counts by status
2. `GetRunTrends(since time.Time, groupBy string) ([]RunTrendPoint, error)` — per-day run counts and success rate
3. `GetPipelineStatistics(since time.Time) ([]PipelineStatistics, error)` — per-pipeline aggregates
4. `GetPipelineStepStats(pipelineName string, since time.Time) ([]StepPerformanceStats, error)` — per-step stats for a pipeline

**SQL Queries**: All use existing indexed columns (`started_at`, `pipeline_name`, `status`) with `GROUP BY` and `strftime` for date grouping.

---

### R-004: Recovery Hints in Run Introspection

**Decision**: Extract from event message field + re-generate via `recovery.ClassifyError` at display time

**Rationale**: Per C-006, recovery hints are not persisted in `event_log`. The minimum viable approach extracts context from the `message` field of failed events and uses `recovery.BuildRecoveryBlock` to generate hints at display time from the error message, pipeline name, step ID, and run ID. No schema migration needed for MVP.

**Implementation Approach**:
1. When displaying a failed step, call a display-time function that parses the error message
2. Use pattern matching to classify the error (contract validation, security, runtime, unknown)
3. Generate recovery hints using the same logic as `recovery.BuildRecoveryBlock`
4. Display hints in the step detail view

---

### R-005: Workspace File Browsing Path Resolution

**Decision**: Resolve from `step_state.workspace_path` with convention-based fallback

**Rationale**: Per C-007, the `step_state` table stores `workspace_path`. Despite the cross-run collision caveat, the path is valid for the most recent execution. The Server already has `wsManager` (workspace.WorkspaceManager). Path traversal prevention uses existing `security.PathValidator` patterns.

**New API Endpoints Required**:
1. `GET /api/runs/{id}/workspace/{step}/tree?path=` — returns directory listing
2. `GET /api/runs/{id}/workspace/{step}/file?path=` — returns file content with syntax highlighting

**Security Constraints**:
- Path must be validated against workspace root (no traversal)
- Read-only access only
- File size limit (1 MB, consistent with `maxArtifactSize`)
- Symlink following disabled
- Content sanitized (HTML escaped) before serving

---

### R-006: Pipeline Detail View Data Sources

**Decision**: Load pipeline YAML via existing `loadPipelineYAML` + manifest personas

**Rationale**: The existing `handlers_pipelines.go` already loads pipeline YAML to build summaries. The pipeline detail view extends this with full step configuration, persona cross-references (via `s.manifest.Personas`), and contract definitions. No new data source needed — all configuration is available from YAML files and the manifest.

**Data Assembly**:
- Pipeline metadata, steps, dependencies, input config → from YAML
- Persona details (adapter, model, permissions, prompt) → from `s.manifest.Personas`
- Contract schemas → from step `Handover.Contract` config, resolved via `SchemaPath` or inline `Schema`
- Last run status → query `pipeline_run` table for most recent run per pipeline name

---

### R-007: JS Budget Analysis

**Decision**: Total new JS estimated at ~7 KB gzipped, within budget

| Asset | Raw Size | Gzipped (est.) |
|-------|----------|----------------|
| Existing app.js | 4,604 B | ~1.8 KB |
| Existing dag.js | 1,047 B | ~0.5 KB |
| Existing sse.js | 9,152 B | ~3.2 KB |
| **Existing total** | **14,803 B** | **~5.5 KB** |
| New markdown.js | ~5,000 B | ~2.0 KB |
| New highlight.js | ~4,000 B | ~1.5 KB |
| New stats.js (charts via CSS/canvas) | ~3,000 B | ~1.2 KB |
| New workspace.js (tree browser) | ~3,000 B | ~1.2 KB |
| New introspect.js (toggles, drill-down) | ~2,000 B | ~0.8 KB |
| **New total** | **~17,000 B** | **~6.7 KB** |
| **Grand total** | **~31,803 B** | **~12.2 KB** |

This is well within the 50 KB gzipped budget (NFR-001).

---

### R-008: Template Architecture

**Decision**: Extend existing clone-per-page template system with new page templates

**Rationale**: The existing `embed.go` uses a clone-per-page strategy where layout + partials form a shared base. New pages (pipeline detail, persona detail, statistics, run introspection) follow this same pattern. New partials for reusable components (markdown renderer, syntax highlighter, step inspector).

**New Templates**:
- `templates/pipeline_detail.html` — pipeline inspection view
- `templates/persona_detail.html` — persona inspection view
- `templates/statistics.html` — statistics dashboard
- `templates/partials/markdown_viewer.html` — markdown with raw/rendered toggle
- `templates/partials/code_viewer.html` — syntax highlighted code with raw/formatted toggle
- `templates/partials/step_inspector.html` — step introspection with contract/artifact/event detail
- `templates/partials/workspace_tree.html` — file tree browser
- `templates/partials/stats_chart.html` — statistics visualizations (CSS-based bar charts)

---

### R-009: Statistics Visualization Without External Libraries

**Decision**: CSS-based bar charts + HTML tables for trend data

**Rationale**: Canvas/SVG charting libraries would blow the JS budget. CSS-based horizontal bar charts using `width: N%` with gradient backgrounds are sufficient for showing run counts, success rates, and per-pipeline breakdowns. Trend data uses a simple table with bar sparklines.

**Implementation**:
- Aggregate counts → big number cards (total, succeeded, failed, cancelled, success rate %)
- Per-pipeline breakdown → table with inline CSS bar charts
- Trend data → table with date column + bar sparkline column using `<div>` bars
- Time range filter → `<select>` that reloads the page with a `?range=` query param
