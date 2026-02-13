# Implementation Plan: Dashboard Inspection, Rendering, Statistics & Run Introspection

**Branch**: `091-dashboard-introspection` | **Date**: 2026-02-14 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/091-dashboard-introspection/spec.md`

## Summary

Enhance the Wave web dashboard with 7 feature areas: pipeline/persona/contract inspection views, markdown rendering with raw/rendered toggle, YAML/JSON syntax highlighting, run statistics dashboard, meta information display, run introspection with drill-down, and workspace/source browsing. Implementation uses server-side SQL aggregation for statistics, client-side custom parsers for markdown and syntax highlighting (within 50 KB gzipped JS budget), and extends the existing `webui` package architecture with new handlers, templates, and API endpoints. All features gated behind the `webui` build tag.

## Technical Context

**Language/Version**: Go 1.25+ (existing project)
**Primary Dependencies**: `net/http` (stdlib, Go 1.22+ enhanced ServeMux), `html/template`, `go:embed`, `modernc.org/sqlite` (existing), `gopkg.in/yaml.v3` (existing)
**Storage**: SQLite via existing `internal/state` package — new aggregate query methods, no schema migration required
**Testing**: `go test ./...` with `-race` flag
**Target Platform**: Linux server (single binary)
**Project Type**: Single Go binary with embedded web assets
**Performance Goals**: Statistics queries <500ms for 10K runs (NFR-002), workspace tree listing <200ms for 500 entries (NFR-003)
**Constraints**: Total JS bundle <50 KB gzipped (NFR-001), all assets embedded via `go:embed` (FR-030), `webui` build tag gating (FR-029)
**Scale/Scope**: Dashboard serving up to 10K pipeline runs, 9 supported syntax highlighting languages, ~17 new files

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | All assets embedded via `go:embed`, no new runtime dependencies |
| P2: Manifest as SSOT | PASS | Reads from manifest (personas, adapters) and pipeline YAML — no config duplication |
| P3: Persona-Scoped Boundaries | N/A | Dashboard is read-only viewer, not an execution boundary |
| P4: Fresh Memory | N/A | No pipeline execution or chat history involved |
| P5: Navigator-First | N/A | Dashboard feature, not a pipeline step |
| P6: Contracts at Handover | N/A | No pipeline handover in this feature |
| P7: Relay via Summarizer | N/A | No context window concerns for HTTP handlers |
| P8: Ephemeral Workspaces | PASS | Workspace browser is strictly read-only (FR-027), path traversal prevented |
| P9: Credentials Never Touch Disk | PASS | System prompt content sanitized, credential redaction via existing `RedactCredentials` |
| P10: Observable Progress | PASS | Reads existing event_log and progress tables — adds visibility, not mutation |
| P11: Bounded Recursion | N/A | No recursive execution |
| P12: Minimal State Machine | N/A | No state transitions — read-only views |
| P13: Test Ownership | PASS | All new code includes unit tests; existing tests must continue passing |

**Post-Phase 1 Re-check**: All principles verified. No violations found.

## Project Structure

### Documentation (this feature)

```
specs/091-dashboard-introspection/
├── plan.md              # This file
├── research.md          # Phase 0 research findings
├── data-model.md        # Phase 1 data model
├── contracts/           # Phase 1 API contracts
│   ├── api-pipeline-detail.json
│   ├── api-persona-detail.json
│   ├── api-statistics.json
│   ├── api-workspace.json
│   └── api-enhanced-run-detail.json
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```
internal/
├── webui/
│   ├── types.go                          # Extended with new response types
│   ├── routes.go                         # Extended with new routes
│   ├── server.go                         # No changes needed
│   ├── embed.go                          # Extended with new page templates
│   ├── handlers_pipelines.go             # Extended: pipeline detail handler
│   ├── handlers_personas.go              # Extended: persona detail handler
│   ├── handlers_runs.go                  # Extended: enhanced run detail with introspection
│   ├── handlers_statistics.go            # NEW: statistics API and page handlers
│   ├── handlers_workspace.go             # NEW: workspace browsing handlers
│   ├── static/
│   │   ├── app.js                        # Existing (no changes)
│   │   ├── dag.js                        # Existing (no changes)
│   │   ├── sse.js                        # Existing (no changes)
│   │   ├── style.css                     # Extended: new component styles
│   │   ├── markdown.js                   # NEW: minimal markdown parser
│   │   ├── highlight.js                  # NEW: syntax highlighter
│   │   ├── stats.js                      # NEW: statistics page interactions
│   │   ├── workspace.js                  # NEW: file tree browser
│   │   └── introspect.js                 # NEW: introspection interactions
│   └── templates/
│       ├── layout.html                   # Extended: nav links for new pages
│       ├── pipeline_detail.html          # NEW: pipeline inspection view
│       ├── persona_detail.html           # NEW: persona inspection view
│       ├── statistics.html               # NEW: statistics dashboard
│       ├── run_detail.html               # Extended: introspection panels
│       ├── pipelines.html                # Extended: links to detail views
│       ├── personas.html                 # Extended: links to detail views
│       └── partials/
│           ├── markdown_viewer.html      # NEW: markdown with toggle
│           ├── code_viewer.html          # NEW: syntax highlight with toggle
│           ├── step_inspector.html       # NEW: step drill-down panel
│           ├── workspace_tree.html       # NEW: file tree browser
│           └── stats_chart.html          # NEW: CSS-based charts
├── state/
│   ├── store.go                          # Extended: new StateStore methods
│   ├── types.go                          # Extended: new record types
│   └── store_test.go                     # Extended: tests for new methods
└── recovery/
    └── recovery.go                       # Referenced (no changes)
```

**Structure Decision**: Extends the existing Go single-project structure. All web assets are within `internal/webui/` using the established patterns (build-tagged Go files, embedded templates and static assets). New state query methods follow the existing `StateStore` interface pattern.

## Complexity Tracking

_No constitution violations found. No complexity justifications needed._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none) | — | — |
