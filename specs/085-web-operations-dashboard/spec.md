# Feature Specification: Web-Based Pipeline Operations Dashboard

**Feature Branch**: `085-web-operations-dashboard`
**Created**: 2026-02-13
**Status**: Draft
**Input**: [GitHub Issue #81](https://github.com/re-cinq/wave/issues/81) — Add web-based pipeline operations dashboard (`wave serve`)

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Monitor Pipeline Runs (Priority: P1)

As a Wave operator, I want to open a browser and see all pipeline runs with their current status so that I can monitor pipeline health without keeping a terminal open.

**Why this priority**: This is the core value proposition of a web dashboard — persistent visibility into pipeline state. Without it, the dashboard serves no purpose.

**Independent Test**: Can be fully tested by starting `wave serve`, opening the browser, and verifying that all pipeline runs from the state database are displayed with correct statuses. Delivers standalone monitoring value even without real-time updates.

**Acceptance Scenarios**:

1. **Given** Wave has completed and active pipeline runs in its state database, **When** a user navigates to the dashboard URL, **Then** all pipeline runs are listed with their name, status, start time, and duration.
2. **Given** the dashboard is open, **When** a user clicks on a specific pipeline run, **Then** step-level details are shown including persona, duration, contract validation status, and any error messages.
3. **Given** the state database is empty, **When** a user opens the dashboard, **Then** an empty state is displayed with a clear message indicating no pipeline runs exist.
4. **Given** multiple pipeline runs exist, **When** a user applies a status filter (e.g., "failed"), **Then** only runs matching that status are displayed.

---

### User Story 2 - Observe Real-Time Pipeline Progress (Priority: P1)

As a Wave operator, I want to see active pipeline executions update in real-time so that I can follow progress without refreshing the page.

**Why this priority**: Real-time feedback is what differentiates a dashboard from a static log viewer. Operators need live updates to know when steps complete, fail, or stall.

**Independent Test**: Can be tested by starting a pipeline via CLI while the dashboard is open, and verifying that progress events appear in the browser without manual page refresh. Delivers real-time monitoring value.

**Acceptance Scenarios**:

1. **Given** the dashboard is open, **When** a pipeline starts executing via CLI, **Then** the new run appears in the dashboard within 2 seconds and progress updates stream in real-time.
2. **Given** a pipeline is actively running, **When** a step completes or fails, **Then** the step status updates in the dashboard immediately (within 1 second of the event).
3. **Given** the browser loses its connection to the server (e.g., network blip), **When** the connection is re-established, **Then** the client automatically reconnects and resumes receiving updates without user intervention.
4. **Given** multiple pipelines are running concurrently, **When** viewing the run list, **Then** all active runs show independent real-time progress indicators.

---

### User Story 3 - Start the Dashboard Server (Priority: P1)

As a Wave operator, I want to run `wave serve` to start the dashboard server so that I have a simple, single-command entry point.

**Why this priority**: Without the serve command, there is no dashboard. This is the foundational infrastructure that enables all other stories.

**Independent Test**: Can be tested by running `wave serve` and verifying the HTTP server starts, binds to the expected address, and serves responses on the root path.

**Acceptance Scenarios**:

1. **Given** Wave is installed, **When** a user runs `wave serve`, **Then** an HTTP server starts on the default port and a message displays the URL to access the dashboard.
2. **Given** the default port is in use, **When** a user runs `wave serve --port 9090`, **Then** the server starts on port 9090.
3. **Given** the server is running, **When** the user presses Ctrl+C, **Then** the server shuts down gracefully, completing any in-flight responses.
4. **Given** no state database exists at the expected path, **When** a user runs `wave serve`, **Then** the server starts successfully and displays an appropriate message indicating no pipeline history is available.

---

### User Story 4 - Control Pipeline Execution (Priority: P2)

As a Wave operator, I want to start, stop, and retry pipelines from the dashboard so that I can manage pipeline execution without switching to a terminal.

**Why this priority**: Execution control elevates the dashboard from read-only monitoring to operational management. This is high value but depends on the monitoring foundation from P1 stories.

**Independent Test**: Can be tested by triggering a pipeline start from the dashboard, observing it in the run list, cancelling it, and then retrying — all through the browser interface.

**Acceptance Scenarios**:

1. **Given** the dashboard is open, **When** a user selects a pipeline and provides input, **Then** a new pipeline run starts and appears in the run list with real-time progress.
2. **Given** a pipeline is actively running, **When** a user clicks "Stop", **Then** a cancellation request is issued and the pipeline transitions to a cancelled state.
3. **Given** a pipeline has failed, **When** a user clicks "Retry", **Then** a new run of the same pipeline starts with the same input parameters.
4. **Given** the user attempts to start a pipeline that requires input, **When** no input is provided, **Then** the dashboard displays a validation message requesting the required input.

---

### User Story 5 - Visualize Pipeline DAG (Priority: P2)

As a Wave operator, I want to see pipeline step dependencies rendered as a visual graph so that I can understand the execution flow and identify bottlenecks.

**Why this priority**: DAG visualization provides unique value that CLI output cannot match. It helps operators understand complex multi-step pipelines and diagnose where failures occur in the dependency chain.

**Independent Test**: Can be tested by opening a pipeline run detail view and verifying that steps are rendered as a directed acyclic graph with visible dependency edges and status colors.

**Acceptance Scenarios**:

1. **Given** a pipeline has multiple steps with dependencies, **When** viewing the run detail page, **Then** steps are rendered as a directed graph with edges showing dependency relationships.
2. **Given** a pipeline is actively running, **When** viewing the DAG, **Then** completed steps are visually distinct from running, pending, and failed steps.
3. **Given** a step has failed, **When** viewing the DAG, **Then** the failed step and all downstream dependents are visually marked as blocked/affected.

---

### User Story 6 - Browse Workspace Artifacts (Priority: P3)

As a Wave operator, I want to browse step outputs and workspace artifacts from the dashboard so that I can inspect results without navigating the filesystem.

**Why this priority**: Artifact browsing is a convenience feature that adds value but is not essential for core dashboard functionality. Operators can always inspect artifacts via the filesystem.

**Independent Test**: Can be tested by completing a pipeline run, navigating to the step detail view, and verifying that output artifacts are listed and viewable in the browser.

**Acceptance Scenarios**:

1. **Given** a completed pipeline run with output artifacts, **When** viewing a step's detail panel, **Then** all artifacts are listed with their name, type, and size.
2. **Given** an artifact is a text-based file (JSON, Markdown, YAML), **When** a user clicks on it, **Then** the file contents are displayed with syntax highlighting.
3. **Given** an artifact may contain sensitive data, **When** displaying artifact contents, **Then** known credential patterns are redacted before display.

---

### User Story 7 - View Configured Personas (Priority: P3)

As a Wave operator, I want to browse all configured personas from the dashboard so that I can understand which agents are available and their configurations.

**Why this priority**: Persona visibility is informational and supports team understanding of the system. It is not required for pipeline monitoring or control.

**Independent Test**: Can be tested by navigating to the persona list page and verifying all personas from the manifest and defaults are displayed with their descriptions and permissions.

**Acceptance Scenarios**:

1. **Given** Wave has personas configured in the manifest and defaults, **When** navigating to the personas section, **Then** all personas are listed with their name, description, and role.
2. **Given** a persona has permission rules, **When** viewing the persona detail, **Then** allowed and denied tool patterns are clearly displayed.

---

### Edge Cases

- What happens when the state database is locked by an active pipeline execution? The dashboard server MUST open a separate read-only SQLite connection (`?mode=ro`) for serving API queries, distinct from the pipeline executor's read-write connection. Both connections use WAL mode, which supports concurrent readers alongside a single writer. The existing `StateStore` interface should be reused via a new constructor (e.g., `NewReadOnlyStateStore`) that opens the database with `PRAGMA query_only=ON` and read-optimized settings (higher `MaxOpenConns` to support concurrent HTTP handlers).
- What happens when the SSE connection is interrupted? The client MUST automatically reconnect and recover without losing visibility into active runs.
- What happens when a pipeline run references a workspace that has been cleaned up? The artifact browser MUST display a clear message indicating artifacts are no longer available.
- What happens when the server is started and no manifest file exists? The server MUST start successfully using only the state database, with persona/pipeline details degraded gracefully.
- What happens when hundreds of pipeline runs exist in the database? The run list MUST support pagination and MUST NOT load all records at once.
- What happens when artifact files are very large (>1 MB)? The artifact viewer MUST truncate display and indicate the file has been truncated, with an option to download the full file.
- What happens when the server is accessed from a non-localhost address without authentication? The server MUST reject the connection unless `--bind 0.0.0.0` was explicitly specified.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: System MUST provide a `wave serve` command that starts an HTTP server serving the dashboard.
- **FR-002**: The `wave serve` command MUST accept `--port` (default: 8080) and `--bind` (default: 127.0.0.1) flags for network configuration.
- **FR-003**: System MUST display all pipeline runs from the state database with their status (pending, running, completed, failed, cancelled).
- **FR-004**: System MUST provide real-time progress updates for active pipeline executions via Server-Sent Events (SSE) without requiring page refresh.
- **FR-005**: System MUST allow users to view step-level details for any pipeline run, including persona, duration, and contract validation results.
- **FR-006**: System MUST support filtering and searching pipeline run history by status, pipeline name, and time range.
- **FR-007**: System MUST allow users to start pipeline executions from the dashboard by selecting a pipeline and providing input.
- **FR-008**: System MUST allow users to stop (cancel) active pipeline executions from the dashboard.
- **FR-009**: System MUST allow users to retry failed pipeline executions from the dashboard.
- **FR-010**: System MUST render pipeline step dependencies as a visual directed acyclic graph (DAG).
- **FR-011**: System MUST allow users to browse workspace artifacts and step outputs for completed runs.
- **FR-012**: System MUST display all configured personas with their descriptions and permission rules.
- **FR-013**: System MUST embed all frontend assets in the Go binary via `go:embed`, requiring no external CDN or runtime file dependencies. The frontend MUST use Go `html/template` for server-side rendering combined with vanilla JavaScript and minimal CSS for interactivity. This approach aligns with the project's zero-external-dependency philosophy, keeps the bundle well under NFR-001's 50 KB limit, and follows the same `go:embed` pattern established in `internal/defaults/embed.go`. No JavaScript build toolchain is required.
- **FR-014**: System MUST support a build tag `webui` to opt in to embedded UI assets. When built without `//go:build webui`, the `wave serve` command MUST print an error message indicating the binary was built without dashboard support. The tag name `webui` follows Go convention of short, lowercase build tags.
- **FR-015**: System MUST shut down gracefully on interrupt signals, completing in-flight responses before stopping.
- **FR-016**: System MUST sanitize artifact contents before display, redacting patterns that match known credential formats.
- **FR-017**: System MUST paginate pipeline run results using cursor-based pagination (keyed on `started_at` timestamp + `run_id` for uniqueness). Default page size MUST be 25, maximum page size MUST be 100. Cursor-based pagination is preferred over offset-based because it provides stable results when new runs are added during browsing and aligns with the existing `ListRuns` query pattern that orders by `started_at DESC`.
- **FR-018**: System MUST NOT affect existing CLI functionality — all current commands MUST work identically with or without the dashboard feature.

### Non-Functional Requirements

- **NFR-001**: Total JavaScript bundle size MUST be under 50 KB gzipped.
- **NFR-002**: Binary size increase from embedded assets MUST NOT exceed 200 KB.
- **NFR-003**: Dashboard MUST be responsive and usable on desktop and tablet screen sizes.
- **NFR-004**: SSE reconnection MUST happen automatically within 5 seconds of connection loss.
- **NFR-005**: API response time for listing pipeline runs MUST be under 200ms for up to 1000 records.

### Security Requirements

- **SR-001**: Server MUST bind to localhost (127.0.0.1) by default.
- **SR-002**: When `--bind 0.0.0.0` is specified, the server MUST require authentication for all API endpoints. Authentication MUST use a static bearer token passed via `--token` flag or `WAVE_SERVE_TOKEN` environment variable. The server MUST generate and display a random token at startup if none is provided. Clients authenticate via `Authorization: Bearer <token>` header. This approach avoids introducing user management infrastructure while providing adequate protection for a single-operator tool.
- **SR-003**: System MUST prevent path traversal when serving workspace artifacts, validating all paths against the workspace root.
- **SR-004**: CORS MUST be restricted to same-origin for localhost-bound servers.
- **SR-005**: Pipeline outputs displayed in the browser MUST be sanitized to prevent XSS attacks.

### Key Entities

- **Server**: The HTTP server instance managing the dashboard lifecycle, configured with bind address, port, and database path.
- **Pipeline Run**: A record of a pipeline execution with status, timing, step progress, and associated artifacts. Central entity for all dashboard views.
- **Step Progress**: Per-step execution state within a pipeline run, including persona assignment, duration, validation results, and output artifacts.
- **SSE Stream**: A persistent connection from the browser to the server carrying real-time progress events for active pipeline executions.
- **Artifact**: A file produced by a pipeline step, stored in the workspace filesystem, referenced by path and metadata in the state database.
- **Persona**: A configured AI agent with a specific role, system prompt, and permission rules (allowed/denied tools).
- **DAG Node**: A visual representation of a pipeline step in the dependency graph, with edges to its dependencies and a status indicator.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `wave serve` starts an HTTP server and the dashboard is accessible in a browser within 2 seconds of command execution.
- **SC-002**: All pipeline runs from the state database are visible on the dashboard with correct status, name, and timing information.
- **SC-003**: Active pipeline progress updates appear in the browser within 2 seconds of the event occurring on the server, without page refresh.
- **SC-004**: Users can start, stop, and retry pipeline executions entirely from the browser without touching the CLI.
- **SC-005**: Pipeline step dependencies are rendered as a visual DAG with clear status indicators for each node.
- **SC-006**: Total JavaScript bundle is under 50 KB gzipped, measured by the build output.
- **SC-007**: All frontend assets are embedded in the binary — the dashboard functions with zero external network requests for assets.
- **SC-008**: Build tag exclusion produces a binary with no UI asset overhead when the dashboard feature is not needed.
- **SC-009**: Existing CLI commands (`wave run`, `wave status`, `wave list`) produce identical behavior with and without the dashboard build tag.
- **SC-010**: The server handles concurrent read access from multiple browser clients while a pipeline is executing without errors or data corruption.

## Clarifications

The following ambiguities were identified during spec review and resolved based on codebase patterns and industry standards.

### C-001: Authentication mechanism for non-localhost binding (SR-002)

**Ambiguity**: SR-002 requires authentication when `--bind 0.0.0.0` is specified but did not define the authentication mechanism.

**Resolution**: Static bearer token via `--token` flag or `WAVE_SERVE_TOKEN` environment variable. Auto-generated random token displayed at startup if none provided.

**Rationale**: Wave is a single-operator development tool, not a multi-tenant service. The codebase has no existing user management or session infrastructure (`internal/security/` contains no authentication code). A static bearer token provides adequate protection against unauthorized access on a LAN without introducing infrastructure complexity. This follows the pattern used by tools like Jupyter Notebook and similar development servers.

### C-002: SQLite concurrent access for dashboard reads (Edge Cases)

**Ambiguity**: The edge case for database locking mentioned WAL mode but didn't specify how the dashboard server should coexist with the pipeline executor, which uses `SetMaxOpenConns(1)`.

**Resolution**: Dashboard opens a separate read-only connection via `NewReadOnlyStateStore` with `PRAGMA query_only=ON` and higher connection limits suitable for concurrent HTTP handlers.

**Rationale**: The existing `StateStore` in `internal/state/store.go` uses `SetMaxOpenConns(1)` which is correct for the single-writer pipeline executor. SQLite WAL mode (already enabled at line 129) supports unlimited concurrent readers alongside a single writer. A read-only connection pool allows multiple browser clients to query simultaneously without contending with the executor's write lock.

### C-003: Frontend technology stack (FR-013)

**Ambiguity**: The spec required `go:embed` and <50 KB JS but did not specify what frontend technology to use.

**Resolution**: Go `html/template` for server-side rendering + vanilla JavaScript + minimal CSS. No JavaScript build toolchain.

**Rationale**: The project already uses `go:embed` extensively (see `internal/defaults/embed.go` lines 18-28). Server-side templates with vanilla JS trivially meet the 50 KB JS limit, add zero build complexity, and keep the binary size increase well under NFR-002's 200 KB limit. This avoids introducing npm/Node.js as a build dependency, which would conflict with Wave's single-binary philosophy. DAG visualization (FR-010) can be achieved with SVG generation from Go templates or a tiny canvas library.

### C-004: Build tag naming (FR-014)

**Ambiguity**: FR-014 mentioned a "build tag mechanism" without specifying the tag name.

**Resolution**: Build tag name is `webui`. Files guarded by `//go:build webui`.

**Rationale**: Go convention favors short, lowercase build tags. `webui` is descriptive, unambiguous, and follows the pattern of other Go projects with optional web interfaces. Usage: `go build -tags webui ./cmd/wave`.

### C-005: Pagination strategy (FR-017)

**Ambiguity**: FR-017 required pagination without specifying the pagination model, default page size, or maximum page size.

**Resolution**: Cursor-based pagination keyed on `(started_at, run_id)`. Default page size: 25. Maximum page size: 100.

**Rationale**: The existing `ListRuns` method in `internal/state/store.go` (line 553) already queries with `ORDER BY started_at DESC` and supports a `Limit` parameter. Cursor-based pagination provides stable results when new runs are created during browsing (offset-based would cause items to shift). The composite cursor `(started_at, run_id)` ensures uniqueness since `started_at` alone could have ties. Default of 25 balances initial load time with utility; maximum of 100 prevents expensive queries.
