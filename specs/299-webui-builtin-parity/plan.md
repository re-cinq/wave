# Implementation Plan: Embed Web UI as Default Built-in with CLI/TUI Feature Parity

**Branch**: `299-webui-builtin-parity` | **Date**: 2026-03-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/299-webui-builtin-parity/spec.md`

## Summary

Remove the `//go:build webui` build tag from all 22 webui source files and the serve command, making the web dashboard compile unconditionally into the Wave binary. Then achieve CLI/TUI feature parity by: fixing the retry handler to actually launch execution, adding a resume-from-step API endpoint, implementing SSE Last-Event-ID backfill for reconnection, and enhancing templates with responsive layout, keyboard navigation, and DAG interaction.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `net/http` (stdlib), `html/template` (stdlib), `embed` (stdlib), `modernc.org/sqlite`
**Storage**: SQLite via `internal/state/` — no schema changes needed
**Testing**: `go test ./...` with `-race` flag
**Target Platform**: Linux (single static binary)
**Project Type**: Single binary with embedded web assets
**Performance Goals**: SSE events delivered within 1 second of emission (SC-005), binary size delta <2MB (SC-002)
**Constraints**: No JavaScript build step, no external runtime dependencies, server-rendered templates with vanilla JS
**Scale/Scope**: 7 user stories, 18 functional requirements, ~25 files modified

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | Removing build tag makes webui always-compiled. Embedded assets are architecture-independent. No new runtime deps. |
| P2: Manifest as SSOT | PASS | Server config still reads from `wave.yaml`. No new config files. |
| P3: Persona-Scoped Execution | PASS | Pipeline execution through webui uses same executor/persona system. |
| P4: Fresh Memory at Step Boundary | PASS | No change to execution model. |
| P5: Navigator-First | PASS | Pipeline execution unchanged. |
| P6: Contracts at Every Handover | PASS | Contract validation unchanged. |
| P7: Relay via Summarizer | PASS | No change to relay system. |
| P8: Ephemeral Workspaces | PASS | Webui-launched pipelines use same workspace isolation. |
| P9: Credentials Never Touch Disk | PASS | Credential redaction already in `redact.go`. Auth tokens via env/flags only. |
| P10: Observable Progress | PASS | SSE streaming uses same event system as TUI. Adding Last-Event-ID improves reliability. |
| P11: Bounded Recursion | PASS | No change to recursion limits. |
| P12: Minimal Step State Machine | PASS | No new states. |
| P13: Test Ownership | PASS | All existing tests must pass with build tag removed. New tests for resume endpoint. |

**Post-Phase-1 Re-check**: All principles still satisfied. The resume API delegates to existing `ResumeManager` which already enforces P4, P5, P6, P8.

## Project Structure

### Documentation (this feature)

```
specs/299-webui-builtin-parity/
├── plan.md              # This file
├── research.md          # Phase 0: research decisions
├── data-model.md        # Phase 1: entity/API model
├── spec.md              # Feature specification
├── checklists/          # Validation checklists
└── tasks.md             # Phase 2 output (not yet created)
```

### Source Code (repository root)

```
cmd/wave/commands/
├── serve.go             # MODIFY: remove //go:build webui tag
└── serve_stub.go        # DELETE: no longer needed

internal/webui/
├── auth.go              # MODIFY: remove build tag
├── auth_test.go         # MODIFY: remove build tag
├── dag.go               # MODIFY: remove build tag
├── dag_test.go          # MODIFY: remove build tag
├── embed.go             # MODIFY: remove build tag
├── handlers_artifacts.go # MODIFY: remove build tag
├── handlers_control.go  # MODIFY: remove build tag, add resume handler, fix retry handler
├── handlers_personas.go # MODIFY: remove build tag
├── handlers_pipelines.go # MODIFY: remove build tag
├── handlers_runs.go     # MODIFY: remove build tag
├── handlers_sse.go      # MODIFY: remove build tag, add Last-Event-ID backfill
├── handlers_test.go     # MODIFY: remove build tag, add resume tests
├── middleware.go         # MODIFY: remove build tag
├── pagination.go        # MODIFY: remove build tag
├── pagination_test.go   # MODIFY: remove build tag
├── redact.go            # MODIFY: remove build tag
├── redact_test.go       # MODIFY: remove build tag
├── routes.go            # MODIFY: remove build tag, add resume route
├── server.go            # MODIFY: remove build tag
├── sse.go               # MODIFY: remove build tag
├── sse_broker.go        # MODIFY: remove build tag
├── types.go             # NO CHANGE: already unconditional
├── static/
│   ├── style.css        # MODIFY: responsive breakpoints, focus indicators
│   ├── app.js           # MODIFY: keyboard navigation, polling fallback
│   ├── sse.js           # MODIFY: Last-Event-ID support, reconnection backfill
│   └── dag.js           # MODIFY: hover tooltips, click-to-inspect
└── templates/
    ├── layout.html      # MODIFY: viewport meta, ARIA landmarks
    ├── run_detail.html   # MODIFY: resume dropdown, enhanced step cards
    ├── runs.html         # MINOR: responsive adjustments
    ├── personas.html     # MINOR: permission summary display
    ├── pipelines.html    # MINOR: DAG preview
    └── partials/
        ├── dag_svg.html      # MODIFY: ARIA labels, tooltip containers
        ├── step_card.html    # MODIFY: log toggle, artifact link
        ├── artifact_viewer.html # MODIFY: truncation notice
        └── resume_dialog.html # NEW: step picker for resume

internal/state/
└── store.go             # MODIFY: add AfterID to EventQueryOptions, update GetEvents query
```

**Structure Decision**: This is a modification-heavy feature within the existing `internal/webui/` package. No new packages or structural changes. The primary change is removing build tags (mechanical) and adding resume/SSE/responsive capabilities (behavioral).

## Implementation Phases

### Phase A: Build Tag Removal (FR-001, SC-001)
1. Remove `//go:build webui` from all 22 files in `internal/webui/`
2. Remove `//go:build webui` from `cmd/wave/commands/serve.go`
3. Delete `cmd/wave/commands/serve_stub.go`
4. Run `go build ./cmd/wave` (no tags) — verify compilation
5. Run `go test ./internal/webui/... ./cmd/wave/...` — verify all tests pass
6. Document binary size delta (SC-002)

### Phase B: Retry Handler Fix (C5, User Story 2)
1. Extract execution logic from `handleStartPipeline` into `launchPipelineExecution(runID, pipelineName, input string) error` method on Server
2. Refactor `handleStartPipeline` to call `launchPipelineExecution`
3. Update `handleRetryRun` to call `launchPipelineExecution` after creating the DB record
4. Add test: retry creates a new run AND launches execution

### Phase C: Resume-from-Step API (FR-007, User Story 4, C2)
1. Add `ResumeRunRequest` and `ResumeRunResponse` types to `types.go`
2. Add `handleResumeRun` handler in `handlers_control.go`:
   - Validate run exists and is failed/cancelled
   - Load pipeline YAML
   - Create new run record
   - Create executor with `ResumeManager`
   - Call `ResumeManager.ResumeFromStep()`
3. Register route `POST /api/runs/{id}/resume` in `routes.go`
4. Add resume dialog partial template
5. Add resume dropdown UI in `run_detail.html` for failed/cancelled runs
6. Add tests for resume endpoint (valid step, invalid step, wrong state)

### Phase D: SSE Reconnection Backfill (FR-004, FR-005, C3)
1. Add `AfterID int64` to `EventQueryOptions` in `internal/state/store.go`
2. Update `GetEvents` SQL query to support `WHERE id > ?` filtering
3. Update `SSEBroker.EmitProgress` to include `ID` in SSE event output
4. Update `handleSSE` to:
   - Parse `Last-Event-ID` header
   - Backfill missed events from DB before subscribing to live stream
   - Include `id:` field in SSE output format
5. Add polling fallback endpoint or document that `GET /api/runs/{id}` serves as fallback
6. Update `static/sse.js` with reconnection logic

### Phase E: Template & Frontend Enhancements (FR-008, FR-012, FR-013, FR-014, FR-015)
1. **Responsive layout** (FR-012, User Story 7):
   - Add viewport meta tag to `layout.html`
   - Add CSS media queries for 768px, 1024px, 1920px+ breakpoints
   - Ensure no horizontal scrolling at 768px (SC-007)
2. **Keyboard navigation** (FR-013, User Story 7):
   - Add tabindex and ARIA labels to all interactive elements
   - Add keyboard event handlers for Tab/Enter/Escape
   - Add visible focus indicators in CSS (SC-008)
3. **DAG interaction** (FR-008, User Story 6):
   - Add hover tooltips showing step status, duration, tokens
   - Add click-to-inspect linking to step detail section
   - Add CSS `overflow: auto` for large DAGs (50+ steps edge case)
   - Add ARIA labels on SVG nodes/edges
4. **Persona display** (FR-014, User Story 5):
   - Enhance `personas.html` with adapter, model, permission summary
5. **Pipeline display** (FR-015, User Story 5):
   - Enhance `pipelines.html` with step dependencies and mini DAG

### Phase F: Security & Error Handling (FR-009, FR-010, FR-011, FR-016)
1. Verify credential redaction covers all patterns (FR-009, SC-009)
2. Verify auth enforcement for non-localhost (FR-010) — already implemented
3. Verify security headers (FR-011) — CSP, X-Frame-Options already set
4. Add structured error messages with recovery hints (FR-016)
5. Run security-focused test pass

### Phase G: Testing & Validation
1. Ensure `go test ./internal/webui/...` passes with no skipped tests (SC-006)
2. Add integration tests for: SSE streaming, resume endpoint, retry execution
3. Run `go test -race ./...` for race detector
4. Verify binary size delta (SC-002): measure with and without webui assets
5. Document binary size in PR description (FR-017)

## Complexity Tracking

_No constitution violations identified. All changes align with existing principles._

| Area | Complexity | Rationale |
|------|-----------|-----------|
| Build tag removal | Low | Mechanical edit to 23 files + 1 deletion |
| Retry handler fix | Low | Extract + call pattern, ~30 lines moved |
| Resume API | Medium | New handler delegating to existing ResumeManager |
| SSE backfill | Medium | DB query extension + handler modification |
| Responsive/a11y | Medium | CSS/HTML changes across templates |
| Frontend interaction | Medium | Vanilla JS for DAG tooltips, keyboard nav |
