# Implementation Plan: Expandable Running Pipelines Section

**Branch**: `772-webui-running-pipelines` | **Date**: 2026-04-11 | **Spec**: `specs/772-webui-running-pipelines/spec.md`  
**Input**: Feature specification from `specs/772-webui-running-pipelines/spec.md`

## Summary

Add an expandable "Running Pipelines" section to the runs overview (`/runs`) that sits between
the filter toolbar and the main run list. The section is always visible, expanded by default,
shows an empty-state CTA when no runs are active, and respects the pipeline-name filter.
Implementation is pure SSR: one additional `store.ListRuns(status=running)` query in the page
handler, two new template fields, a new `rp-section` block in `runs.html`, and new CSS classes
in `style.css`. No new routes, no JavaScript frameworks, no new data types.

## Technical Context

**Language/Version**: Go 1.23+  
**Primary Dependencies**: `html/template` (stdlib), `net/http` (stdlib)  
**Storage**: SQLite via `internal/state` — read-only `ListRuns` query  
**Testing**: `go test ./...` — existing `handlers_runs_test.go`  
**Target Platform**: Web browser (served by Wave's embedded HTTP server)  
**Project Type**: Single Go binary with embedded web UI  
**Performance Goals**: No new N+1 queries; running section query is O(running-runs count), typically low  
**Constraints**: SSR only (no new fetch/SSE); vanilla JS inline in template; no new dependencies  
**Scale/Scope**: Single operator, low concurrency ceiling; unbounded display (CL-002)

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | No new runtime deps; vanilla JS inline in template |
| P2: Manifest as SSOT | N/A | No manifest changes; pure UI feature |
| P3: Persona-Scoped Execution | N/A | UI-only; no persona/pipeline changes |
| P4: Fresh Memory | N/A | UI-only |
| P5: Navigator-First | N/A | UI-only |
| P6: Contracts at Handover | PASS | Contracts defined in `contracts/` directory |
| P7: Relay | N/A | UI-only |
| P8: Ephemeral Workspaces | N/A | UI-only |
| P9: Credentials | N/A | UI-only; no credentials touched |
| P10: Observable Progress | N/A | UI-only; no pipeline state changes |
| P11: Bounded Recursion | N/A | UI-only |
| P12: Step State Machine | N/A | UI-only |
| P13: Test Ownership | PASS | `handlers_runs_test.go` must be extended; `go test ./...` required |

**Verdict**: No constitution violations. Feature is a pure UI change with no impact on
pipeline execution, personas, contracts, or manifest.

## Project Structure

### Documentation (this feature)

```
specs/772-webui-running-pipelines/
├── plan.md                                    # This file
├── spec.md                                    # Feature specification
├── research.md                                # Phase 0 research findings
├── data-model.md                              # Phase 1 data model
├── contracts/
│   ├── handler-data-contract.md              # handleRunsPage data struct invariants
│   └── template-rendering-contract.md        # runs.html output structure invariants
├── checklists/
│   └── requirements.md                       # Requirements checklist (from specify step)
└── tasks.md                                   # Phase 2 output (not yet created)
```

### Source Code (files to modify)

```
internal/webui/
├── handlers_runs.go         # Add running runs query + extend template data struct
├── templates/
│   └── runs.html            # Insert rp-section block between toolbar and wr-list
├── static/
│   └── style.css            # Add .rp-section, .rp-header, .rp-body, .rp-empty CSS
└── handlers_runs_test.go    # Add TestHandleRunsPage_RunningSection test cases
```

**Structure Decision**: Single Go project, standard wave webui layout. All changes are
confined to `internal/webui/` — no new files, only modifications to existing source.

## Complexity Tracking

_No constitution violations — table not applicable._

## Implementation Phases

### Phase 0 — Research (Complete)

See `research.md`. Key findings:
- Collapse/expand pattern: reuse `step_card.html` inline-JS approach with `aria-expanded`
- Data access: second `ListRuns(status=running)` in handler (SSR, no client fetch)
- Filter integration: apply `pipelineFilter` to running query; ignore status tab
- Accessibility: `role="button"`, `tabindex="0"`, Enter/Space keyboard handler
- CSS: new `.rp-*` classes; run cards reuse existing `.wr-run` classes unchanged
- Empty-state CTA: link to `/pipelines`

### Phase 1 — Design & Contracts (Complete)

See `data-model.md` and `contracts/`.

**Template data extension**:
```go
// Added to anonymous struct in handleRunsPage
RunningRuns  []RunSummary  // always non-nil; filtered to status=running + pipeline filter
RunningCount int           // == len(RunningRuns)
```

**Handler change** (`handlers_runs.go`):
1. After parsing `pipelineFilter`, execute a second `s.store.ListRuns` with
   `Status: "running"`, `PipelineName: pipelineFilter`, `Limit: 0` (unbounded).
2. Convert results to `[]RunSummary`, enrich with `s.enrichRunSummaries`.
3. Filter to top-level only (`ParentRunID == ""`).
4. Assign to `RunningRuns`; compute `RunningCount`.

**Template change** (`runs.html`):
- Insert `<div class="rp-section">` block after `</div>` closing `.wr-toolbar` and
  before `<div class="wr-list">`.
- Header: `role="button"`, `aria-expanded="true"`, `aria-controls="rp-section-body"`,
  toggle JS inline.
- Body: conditional on `RunningCount` — empty-state CTA or run cards using existing
  `.wr-run` card markup.
- Toggle function `toggleRunningSection()` added to `{{define "scripts"}}` block.

**CSS change** (`style.css`):
- `.rp-section`: margin-bottom spacing matching `.wr-toolbar`.
- `.rp-header`: flex row, cursor pointer, hover state matching existing interactive elements.
- `.rp-badge`: inherits `.badge` base styles, uses `--color-running` background.
- `.rp-chevron`: rotates 90° when collapsed (CSS transform via `.rp-section.collapsed`).
- `.rp-body`: no special styles beyond default block display.
- `.rp-empty`: centered flex column, muted text color, spacing.
- `.rp-cta`: styled as a secondary button/link.

### Phase 2 — Tasks (Pending)

`tasks.md` to be generated by `/speckit.tasks` command.
