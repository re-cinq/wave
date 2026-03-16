# Feature Specification: Embed Web UI as Default Built-in with CLI/TUI Feature Parity

**Feature Branch**: `299-webui-builtin-parity`
**Created**: 2026-03-16
**Status**: Draft
**Input**: https://github.com/re-cinq/wave/issues/299

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Default WebUI Access Without Build Tags (Priority: P1)

A developer installs Wave via `go install` or downloads the binary. They run `wave serve` and immediately get the web operations dashboard without needing special build flags, `-tags webui`, or any additional configuration.

**Why this priority**: This is the foundational change — without removing the build tag gating, no other webui improvements matter. Users currently must know about the `webui` build tag to get the dashboard, which is a hidden capability.

**Independent Test**: Build Wave with `go build ./cmd/wave` (no tags) and verify `wave serve` starts the HTTP server and serves the dashboard at `http://localhost:8080/runs`.

**Acceptance Scenarios**:

1. **Given** a standard `go build ./cmd/wave` (no build tags), **When** the user runs `wave serve`, **Then** the HTTP server starts and serves the web dashboard on the configured port.
2. **Given** a pre-built Wave binary, **When** the user runs `wave serve --port 9090`, **Then** the dashboard is accessible at `http://localhost:9090/runs`.
3. **Given** the webui was previously gated behind `//go:build webui`, **When** the build tag constraint is removed, **Then** all webui source files compile unconditionally and the `wave serve` command is always registered.

---

### User Story 2 - Pipeline Execution and Control via WebUI (Priority: P1)

A user opens the web dashboard to start a pipeline, monitor its execution in real time, and control it (cancel, retry) — matching the same capabilities available through `wave run` CLI and the TUI.

**Why this priority**: Pipeline execution is the core function of Wave. Without full execution control in the webui, it cannot replace CLI/TUI for daily use.

**Independent Test**: Start a pipeline via the webui form, observe real-time SSE step progress updates, cancel it mid-execution, then retry it — all from the browser.

**Acceptance Scenarios**:

1. **Given** the webui is running, **When** the user selects a pipeline and provides input via the start form, **Then** a new pipeline run begins and the user is redirected to the run detail page showing live progress.
2. **Given** a running pipeline in the webui, **When** the user clicks "Cancel", **Then** the pipeline execution stops and the run status updates to "cancelled".
3. **Given** a failed pipeline run, **When** the user clicks "Retry", **Then** a new run is created from the failed run's configuration and execution begins.
4. **Given** a completed run, **When** the user views the run detail page, **Then** they see the DAG visualization, step cards with status, token usage, and timing for each step.

---

### User Story 3 - Step-Level Log Streaming and Artifact Inspection (Priority: P1)

A user debugging a failed pipeline step uses the webui to stream step-level logs in real time and inspect output artifacts — equivalent to `wave logs <run-id>` and `wave artifacts <run-id>`.

**Why this priority**: Observability is critical for debugging pipeline failures. Without log streaming and artifact viewing, users must fall back to the CLI.

**Independent Test**: Run a multi-step pipeline, click into a specific step, observe log output streaming in real time, then browse and view the step's output artifacts.

**Acceptance Scenarios**:

1. **Given** a running pipeline step, **When** the user views the step detail in the webui, **Then** log events stream in real time via SSE without page refresh.
2. **Given** a completed step with output artifacts, **When** the user clicks on an artifact, **Then** the artifact content is displayed with appropriate formatting for its type (JSON, Markdown, plain text).
3. **Given** an artifact containing credentials, **When** the artifact is displayed, **Then** sensitive values (API keys, tokens, passwords) are redacted before rendering.
4. **Given** a large artifact (>100KB), **When** the user views it, **Then** the content is truncated with a clear indicator showing the total size.

---

### User Story 4 - Resume from Failed Step via WebUI (Priority: P2)

A user whose pipeline failed at step 4 of 7 wants to resume execution from that step — equivalent to `wave run --from-step <step>` — directly from the webui.

**Why this priority**: Resume/retry from a specific step saves significant time on long pipelines. This is an important power-user feature but not required for basic operation.

**Independent Test**: Trigger a pipeline that fails at a known step, then use the webui to resume from that step and verify only remaining steps execute.

**Acceptance Scenarios**:

1. **Given** a failed run with step "plan" as the failure point, **When** the user selects "Resume from step" and chooses "plan", **Then** a new run starts executing from the "plan" step, skipping previously completed steps.
2. **Given** the resume dialog, **When** the user views step options, **Then** each step shows its previous status (completed, failed, pending) to inform the resume decision.

---

### User Story 5 - Persona and Manifest Configuration Viewing (Priority: P2)

A user wants to understand their Wave configuration — which personas are available, what permissions they have, and how pipelines are structured — through the webui.

**Why this priority**: Configuration visibility helps users understand and debug their Wave setup. It's important for onboarding but not blocking for core pipeline operation.

**Independent Test**: Navigate to the Personas and Pipelines pages, verify all configured personas and pipelines are listed with their metadata.

**Acceptance Scenarios**:

1. **Given** a manifest with 5 personas, **When** the user navigates to `/personas`, **Then** all 5 personas are displayed with their adapter, model, description, and permission summary.
2. **Given** a pipeline with 6 steps, **When** the user navigates to `/pipelines` and selects it, **Then** the pipeline detail shows all steps, their dependencies, and a DAG visualization.
3. **Given** the manifest configuration, **When** the user views it in the webui, **Then** the display matches what `wave config show` would output in the CLI.

---

### User Story 6 - DAG Visualization and Introspection (Priority: P2)

A user working with complex multi-step pipelines uses the webui's DAG visualization to understand step dependencies, see artifact flow between steps, and inspect contract validation results.

**Why this priority**: DAG visualization is a key differentiator of the webui over CLI/TUI — it provides spatial understanding of pipeline structure that text-based interfaces cannot match.

**Independent Test**: View a pipeline with parallel branches in the DAG view, hover over nodes for details, and click edges to see artifact flow.

**Acceptance Scenarios**:

1. **Given** a pipeline with parallel steps, **When** the DAG is rendered, **Then** parallel branches are laid out side-by-side with dependency edges clearly drawn.
2. **Given** a completed run, **When** the user hovers over a DAG node, **Then** a tooltip shows step status, duration, token usage, and artifact summary.
3. **Given** a step with contract validation, **When** the user inspects the step detail, **Then** contract validation results (pass/fail, schema details, error messages) are displayed.

---

### User Story 7 - Responsive Layout and Accessibility (Priority: P3)

A user accesses the webui from a tablet or uses keyboard navigation and screen reader basics to interact with the dashboard.

**Why this priority**: Responsive design and accessibility expand the user base but are not blocking for core functionality on desktop browsers.

**Independent Test**: Access the webui from a tablet-width viewport, navigate using only the keyboard (Tab, Enter, Escape), and verify all interactive elements are reachable.

**Acceptance Scenarios**:

1. **Given** a viewport width of 768px (tablet), **When** the user views the runs page, **Then** the layout adapts without horizontal scrolling or overlapping elements.
2. **Given** keyboard-only navigation, **When** the user presses Tab repeatedly, **Then** focus moves through all interactive elements in a logical order with visible focus indicators.
3. **Given** a screen reader, **When** the user navigates the page, **Then** all interactive elements have appropriate ARIA labels and page structure uses semantic HTML.

---

### Edge Cases

- What happens when the SSE connection drops mid-stream? The webui reconnects automatically and backfills missed events from the state database.
- What happens when multiple users start the same pipeline concurrently? Each run is independent with its own state and workspace.
- What happens when the webui is served behind a reverse proxy with a path prefix (e.g., `/wave/`)? Path-relative asset URLs work correctly.
- What happens when the SQLite state database is locked by another process? The webui displays a clear error message, not a hang or blank page.
- What happens when a pipeline has 50+ steps? The DAG visualization remains usable with scrolling and/or zooming.
- What happens when the binary is built on different architectures (linux/arm64, darwin/amd64)? Embedded assets are architecture-independent (HTML/CSS/JS).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST compile the webui into the binary unconditionally — no build tags required.
- **FR-002**: System MUST serve the web dashboard when `wave serve` is invoked, binding to the configured host and port.
- **FR-003**: System MUST provide API endpoints for listing, starting, cancelling, and retrying pipeline runs.
- **FR-004**: System MUST stream real-time progress events to the browser via Server-Sent Events (SSE).
- **FR-005**: System MUST provide a fallback polling mechanism when SSE connections fail or are unavailable.
- **FR-006**: System MUST display step-level logs, artifacts, and contract validation results for each run.
- **FR-007**: System MUST support resume-from-step functionality equivalent to `wave run --from-step`.
- **FR-008**: System MUST render pipeline DAG visualizations showing step dependencies and parallel branches.
- **FR-009**: System MUST redact credentials (API keys, tokens, passwords) in all artifact and log displays.
- **FR-010**: System MUST enforce authentication via bearer token for non-localhost bindings.
- **FR-011**: System MUST set security headers (CSP, X-Frame-Options, X-Content-Type-Options) on all responses.
- **FR-012**: System MUST provide a responsive layout that works on viewports from 768px to 1920px+.
- **FR-013**: System MUST provide keyboard navigation for all interactive elements.
- **FR-014**: System MUST display persona metadata (adapter, model, permissions, description) from the manifest.
- **FR-015**: System MUST display pipeline metadata (steps, dependencies, step count) from pipeline YAML files.
- **FR-016**: System MUST display structured error messages with recovery hints when pipeline steps fail.
- **FR-017**: System MUST document the binary size delta (with vs. without embedded webui assets) in the PR description.
- **FR-018**: System MUST use the same event system (`internal/event/`) as the TUI for real-time updates.

### Key Entities

- **Run**: A single execution of a pipeline, with status (pending, running, completed, failed, cancelled), timing, and token usage.
- **Step**: A unit of work within a run, executed by a persona, producing artifacts and validated by contracts.
- **Artifact**: An output file produced by a step, consumable by downstream steps. Has name, path, type, and size.
- **Pipeline**: A DAG of steps with dependency relationships, loaded from YAML configuration.
- **Persona**: An AI agent configuration with adapter, model, permissions, and system prompt.
- **Event**: A progress update emitted during pipeline execution, consumed by SSE clients for real-time display.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `go build ./cmd/wave` (no build tags) produces a binary that serves the webui via `wave serve`.
- **SC-002**: Binary size increase from embedding webui assets is documented and does not exceed 2MB over the base binary.
- **SC-003**: Every pipeline control action available in `wave run` CLI (start, cancel, retry, resume-from-step) is available through the webui API and UI.
- **SC-004**: Every read operation available in `wave list runs`, `wave logs`, `wave artifacts`, and `wave status` has a webui equivalent.
- **SC-005**: Real-time progress events are delivered to the browser within 1 second of emission via SSE.
- **SC-006**: `go test ./internal/webui/...` passes with no skipped tests and covers core routes, SSE streaming, and authentication.
- **SC-007**: The webui layout renders without horizontal scrolling on viewports 768px and wider.
- **SC-008**: All interactive elements are reachable via keyboard navigation (Tab/Enter/Escape).
- **SC-009**: Credential redaction prevents display of AWS keys, API tokens, GitHub PATs, and bearer tokens in artifact views.
- **SC-010**: The webui DAG visualization correctly renders pipelines with up to 20 parallel branches.

## Clarifications

### C1: Build Tag Removal Strategy

**Ambiguity**: FR-001 states "no build tags required" but doesn't specify what happens to the `serve_stub.go` fallback (`//go:build !webui`) or whether a headless build option should be preserved.

**Resolution**: Remove ALL `//go:build webui` constraints from every file in `internal/webui/` and delete `cmd/wave/commands/serve_stub.go` entirely. The webui is always compiled in — there is no headless build variant. This aligns with Wave's "single static binary" constraint (CLAUDE.md) and eliminates the hidden capability problem described in User Story 1. The `serve_stub.go` exists solely as the build-tag fallback and has no independent value once the tag is removed.

### C2: Resume-from-Step API Contract

**Ambiguity**: FR-007 requires resume-from-step parity with `wave run --from-step` but the spec defines no API endpoint, request/response schema, or UI interaction for this capability. The existing routes have retry (POST `/api/runs/{id}/retry`) but no resume.

**Resolution**: Add a new API endpoint `POST /api/runs/{id}/resume` with request body `{"from_step": "<step-id>", "force": false}`. This delegates to `pipeline.ResumeManager.ResumeFromStep()` which already implements the full resume logic including phase validation, stale artifact detection, and sub-pipeline creation. The response mirrors `StartPipelineResponse` with a new `run_id`. The UI surfaces this as a "Resume from…" dropdown on the run detail page for failed/cancelled runs, listing each step with its previous status (completed/failed/pending) per User Story 4, Acceptance Scenario 2.

### C3: SSE Reconnection Protocol

**Ambiguity**: The edge case section states "reconnects automatically and backfills missed events" but doesn't specify the protocol mechanism (Last-Event-ID header, cursor-based pagination, or full replay).

**Resolution**: Use the standard SSE `Last-Event-ID` header protocol. Each SSE event includes an `id:` field set to the event's database row ID. On reconnection, the browser's `EventSource` automatically sends `Last-Event-ID` and the server queries `SELECT * FROM events WHERE id > ? AND run_id = ?` to backfill. This is the industry-standard approach for SSE reconnection and requires no custom client logic beyond the browser's built-in `EventSource` retry. The existing `state.StateStore` already stores events with auto-increment IDs, making this straightforward.

### C4: DAG Visualization Rendering Approach

**Ambiguity**: FR-008 requires DAG visualization but doesn't specify whether rendering is server-side SVG (which `internal/webui/dag.go` already computes via `ComputeDAGLayout`) or client-side JavaScript.

**Resolution**: Use server-side SVG layout computation (existing `ComputeDAGLayout` in `dag.go`) with server-rendered SVG in Go templates. The layout algorithm already implements Kahn's topological sort with layer assignment, node positioning, and bezier curve edge computation. This approach requires no JavaScript dependencies, works without a JS build step, and keeps the "single static binary" constraint intact. Client-side interaction (hover tooltips, click-to-inspect) uses lightweight vanilla JavaScript event handlers on SVG elements, not a full graph library. For the 50+ step edge case, the SVG container gets CSS `overflow: auto` for scrolling.

### C5: Retry Handler Execution Gap

**Ambiguity**: The spec's User Story 2 (Acceptance Scenario 3) states that clicking "Retry" creates a new run and "execution begins", but the existing `handleRetryRun` implementation only creates a database record without launching pipeline execution — unlike `handleStartPipeline` which spawns a goroutine.

**Resolution**: The retry handler MUST launch actual pipeline execution, not just create a DB record. The implementation should extract the shared execution logic from `handleStartPipeline` into a reusable `launchPipelineExecution(runID, pipelineName, input)` method on `Server`, then call it from both `handleStartPipeline` and `handleRetryRun`. This ensures retry semantics match the spec: "a new run is created from the failed run's configuration and execution begins."
