# Feature Specification: Dashboard Inspection, Rendering, Statistics & Run Introspection

**Feature Branch**: `091-dashboard-introspection`
**Created**: 2026-02-13
**Status**: Draft
**Input**: [GitHub Issue #91](https://github.com/re-cinq/wave/issues/91) — Enhance dashboard with inspection views, rendering, statistics, and run introspection

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Pipeline, Persona & Contract Inspection (Priority: P1)

As a Wave operator, I want to view detailed configuration for pipelines, personas, and contracts from the dashboard so that I can understand how my system is configured without reading YAML files manually.

**Why this priority**: Configuration inspection is foundational for all other introspection features. Operators need to understand what exists before they can interpret run results, statistics, or logs.

**Independent Test**: Can be fully tested by starting `wave serve`, navigating to the pipelines page, clicking on a specific pipeline, and verifying that step definitions, persona assignments, contract schemas, dependency graph, and input configuration are displayed. Delivers standalone configuration visibility.

**Acceptance Scenarios**:

1. **Given** Wave has pipelines configured in the manifest, **When** a user navigates to the pipeline detail view, **Then** all steps are displayed with their persona, dependencies, workspace configuration, contract definitions, and input schema.
2. **Given** Wave has personas configured, **When** a user navigates to the persona detail view, **Then** the persona's system prompt, adapter, model, temperature, allowed/denied tools, hooks, and sandbox configuration are displayed.
3. **Given** a pipeline step has a contract defined, **When** viewing the step in the pipeline detail, **Then** the contract type (JSON schema, TypeScript, test suite, markdown spec, template, format), schema content, and validation rules are displayed.
4. **Given** a pipeline step references a persona, **When** viewing the pipeline detail, **Then** the persona name is a link to the persona detail view for cross-reference navigation.
5. **Given** a contract references an external schema file, **When** viewing the contract detail, **Then** the schema file content is displayed inline with syntax highlighting.

---

### User Story 2 - Run Statistics Dashboard (Priority: P1)

As a Wave operator, I want to view aggregate statistics about pipeline runs — total, succeeded, failed — along with trends over time, so that I can assess system health and identify recurring issues.

**Why this priority**: Statistics provide operational insight that raw run lists cannot. Understanding failure rates and trends is essential for a production operations tool. This delivers unique value beyond what the existing run list provides.

**Independent Test**: Can be fully tested by navigating to a statistics page and verifying that aggregate counts (total, succeeded, failed, cancelled) and per-pipeline breakdowns are displayed based on historical run data in the state database.

**Acceptance Scenarios**:

1. **Given** the state database contains completed pipeline runs, **When** a user navigates to the statistics page, **Then** aggregate counts are displayed: total runs, successful runs, failed runs, cancelled runs, and overall success rate as a percentage.
2. **Given** pipeline runs span multiple days, **When** viewing the statistics page, **Then** a trend view shows run counts and success rate grouped by day for the selected time period.
3. **Given** multiple pipelines have been executed, **When** viewing the statistics page, **Then** a per-pipeline breakdown shows run count, success rate, average duration, and average token consumption for each pipeline.
4. **Given** the statistics page is open, **When** a user selects a time range filter (last 24 hours, last 7 days, last 30 days, all time), **Then** all statistics update to reflect only runs within that time range.
5. **Given** per-step performance data exists, **When** viewing pipeline-specific statistics, **Then** per-step statistics show average duration, token usage, and success rate for each step in the pipeline.

---

### User Story 3 - Run Introspection (Priority: P1)

As a Wave operator, I want to deeply inspect a specific pipeline run — including every step's contracts, persona configuration, artifacts, event timeline, and error details — so that I can debug failures and understand execution behavior.

**Why this priority**: Run introspection is the core debugging tool. The existing run detail view shows basic status; this enhancement provides the depth needed to diagnose failures, understand token consumption, and trace execution flow.

**Independent Test**: Can be fully tested by opening a completed (or failed) pipeline run, drilling into a specific step, and verifying that the full event timeline, contract validation results, persona assignment, artifacts, error messages, recovery hints, and performance metrics are displayed.

**Acceptance Scenarios**:

1. **Given** a pipeline run has completed, **When** a user views the run detail page, **Then** a step-by-step timeline shows each step's start time, end time, duration, persona, status, token usage, and any error or recovery hints.
2. **Given** a step has contract validation results, **When** viewing the step detail, **Then** the contract type, schema, validation outcome (pass/fail), and any validation error messages are displayed.
3. **Given** a step has produced artifacts, **When** viewing the step detail, **Then** artifacts are listed with name, type, size, and a preview of text-based content.
4. **Given** a step has failed, **When** viewing the step detail, **Then** the failure reason (timeout, context exhaustion, general error), error message, and any recovery hints are prominently displayed.
5. **Given** a run has multiple steps, **When** viewing the run detail page, **Then** the user can drill down from the run overview to individual step details and back without losing context.
6. **Given** event log entries exist for a run, **When** viewing the run detail, **Then** a chronological event timeline shows all events with timestamps, state transitions, messages, and token deltas.

---

### User Story 4 - Markdown Rendering (Priority: P2)

As a Wave operator, I want markdown content (system prompts, artifact content, documentation) rendered with proper formatting in the dashboard, with a toggle to switch between rendered and raw views, so that I can read content naturally while retaining access to the source.

**Why this priority**: Markdown rendering improves readability for system prompts, artifact content, and documentation. It's valuable but not essential for core operational tasks.

**Independent Test**: Can be tested by navigating to a persona detail view and verifying that the system prompt file renders with proper markdown formatting, and that a toggle switches between rendered and raw views.

**Acceptance Scenarios**:

1. **Given** a persona has a system prompt file containing markdown, **When** viewing the persona detail, **Then** the system prompt is rendered with proper heading, list, code block, and emphasis formatting.
2. **Given** markdown content is displayed, **When** a user clicks the "Raw" toggle, **Then** the raw markdown source is shown in a monospace font. Clicking "Rendered" restores the formatted view.
3. **Given** an artifact file has a `.md` extension, **When** viewing the artifact in the artifact browser, **Then** the content is rendered as formatted markdown by default.

---

### User Story 5 - YAML & Schema Rendering (Priority: P2)

As a Wave operator, I want YAML files (manifests, pipeline definitions) and JSON schemas (contracts) rendered with syntax highlighting in the dashboard, with a toggle between formatted and raw views, so that I can quickly read configuration without a text editor.

**Why this priority**: Syntax-highlighted YAML and JSON improves comprehension of configuration. Combined with inspection views, this makes the dashboard a self-contained configuration browser.

**Independent Test**: Can be tested by navigating to a pipeline detail view and verifying that the pipeline YAML is displayed with syntax highlighting, and that JSON schema contract definitions are similarly highlighted.

**Acceptance Scenarios**:

1. **Given** a pipeline configuration is displayed, **When** viewing the pipeline detail, **Then** YAML content is rendered with syntax highlighting distinguishing keys, values, strings, and comments.
2. **Given** a contract has a JSON schema definition, **When** viewing the contract detail, **Then** the JSON schema is rendered with syntax highlighting.
3. **Given** highlighted content is displayed, **When** a user clicks the "Raw" toggle, **Then** the content is shown as plain text without highlighting. Clicking "Formatted" restores the highlighted view.
4. **Given** a YAML file contains deeply nested structures, **When** viewing the file, **Then** indentation levels are visually distinct and the structure is readable.

---

### User Story 6 - Meta Information Display (Priority: P2)

As a Wave operator, I want to see metadata for all entities — pipelines, personas, contracts — including descriptive information, relationships, and usage context, so that I can understand the full picture of my Wave configuration.

**Why this priority**: Metadata display provides contextual information that helps operators make informed decisions. It builds on the inspection views from P1 to add a richer understanding of the system.

**Independent Test**: Can be tested by navigating to pipeline, persona, and contract views and verifying that descriptive metadata (description, relationships, operational status) is displayed alongside the configuration details.

**Acceptance Scenarios**:

1. **Given** a pipeline has metadata (name, description), **When** viewing the pipeline detail, **Then** the metadata is displayed prominently at the top of the page.
2. **Given** a persona has a description, **When** viewing the persona list, **Then** each persona shows its name, description, adapter, and model.
3. **Given** a pipeline step uses a persona, **When** viewing the pipeline detail, **Then** the relationship between step and persona is clearly indicated with navigation links.
4. **Given** a pipeline has been executed at least once, **When** viewing the pipeline detail, **Then** the most recent run status and time are displayed as part of the pipeline metadata.
5. **Given** a pipeline defines input configuration with examples, **When** viewing the pipeline detail, **Then** the input examples are displayed to help operators understand expected input format.

**Note on metadata fields**: The GitHub issue requests "last changed, created date, version, author" metadata. The current manifest and pipeline data models (`Manifest`, `Pipeline`, `Persona` types) do not store these fields — these are out of scope (see C-004). This spec addresses the metadata that IS available: name, description, adapter, model, relationships, and operational status derived from run history.

---

### User Story 7 - Workspace & Source Browsing (Priority: P3)

As a Wave operator, I want to browse the workspace files and source code associated with a pipeline run from the dashboard, with syntax highlighting for code files, so that I can inspect execution context without navigating the filesystem.

**Why this priority**: Source browsing is a convenience feature for deep debugging. Operators can always use the filesystem directly, but having it in the dashboard eliminates context switching.

**Independent Test**: Can be tested by opening a completed pipeline run, selecting the workspace browsing tab, and verifying that the directory tree and file contents are displayed with syntax highlighting for recognized file types.

**Acceptance Scenarios**:

1. **Given** a pipeline run has a workspace that still exists on disk, **When** viewing the run detail page, **Then** a "Workspace" tab shows a file tree of the workspace directory.
2. **Given** the workspace file tree is displayed, **When** a user clicks on a file, **Then** the file contents are displayed with syntax highlighting appropriate to the file type (Go, YAML, JSON, Markdown, etc.).
3. **Given** a workspace has been cleaned up, **When** viewing the run detail page, **Then** the workspace tab displays a message indicating the workspace is no longer available.
4. **Given** the workspace browser is open, **When** viewing any file, **Then** the view is strictly read-only with no editing capability.
5. **Given** a file in the workspace is very large (>1 MB), **When** viewing the file, **Then** the content is truncated with a clear indicator and a download link for the full file.

---

### Edge Cases

- What happens when a pipeline definition has changed since a run was executed? The run detail MUST show the configuration as it was at execution time (from stored event data), not the current definition. If historical configuration is unavailable, a notice MUST indicate that the displayed configuration is current and may differ from what was used during execution.
- What happens when the statistics page is loaded with zero pipeline runs? The statistics page MUST display zero counts and an empty state message rather than errors or broken visualizations.
- What happens when a persona's system prompt file no longer exists on disk? The persona detail MUST display a notice that the file is unavailable rather than returning an error.
- What happens when syntax highlighting is requested for an unrecognized file type? The file MUST be displayed as plain text without errors.
- What happens when artifact content contains HTML or script tags? All content MUST be escaped/sanitized before rendering to prevent XSS, consistent with SR-005 from spec 085.
- What happens when a workspace directory contains thousands of files? The file tree browser MUST use lazy loading (expand-on-click) rather than loading the full tree at once, and MUST limit directory listings to a reasonable maximum (e.g., 500 entries per directory).
- What happens when a contract schema references external files that no longer exist? The contract display MUST show a notice that referenced files are unavailable, alongside any content that is available.
- What happens when the database contains performance metrics for pipelines that have been removed from the manifest? Statistics MUST still display historical data with a note that the pipeline is no longer configured.

## Requirements _(mandatory)_

### Functional Requirements

#### Inspection Views

- **FR-001**: System MUST provide a pipeline detail view showing all steps with their persona assignments, dependencies, workspace configuration, contract definitions, and input schema.
- **FR-002**: System MUST provide a persona detail view showing the persona's system prompt content, adapter, model, temperature, allowed/denied tools, hooks configuration, and sandbox settings.
- **FR-003**: System MUST provide contract detail display within pipeline and run views, showing contract type, schema content, and validation rules.
- **FR-004**: System MUST support navigation between related entities — clicking a persona name in a pipeline view MUST navigate to the persona detail, and vice versa.

#### Rendering

- **FR-005**: System MUST render markdown content with proper formatting including headings, lists, code blocks, emphasis, links, and tables.
- **FR-006**: System MUST provide a toggle between rendered and raw views for markdown content. Default view MUST be rendered.
- **FR-007**: System MUST render YAML and JSON content with syntax highlighting that visually distinguishes keys, values, strings, numbers, booleans, and comments.
- **FR-008**: System MUST provide a toggle between syntax-highlighted and plain text views for YAML and JSON content.
- **FR-009**: Markdown rendering MUST use a lightweight client-side JavaScript parser (~5-8 KB gzipped) to support the raw/rendered toggle without additional network requests. The parser MUST support the subset needed for system prompts and artifacts: headings, lists, code blocks, emphasis, links, and tables. The rendering approach MUST fit within the existing 50 KB gzipped JS budget (NFR-001 from spec 085). See C-001 for rationale.

#### Statistics

- **FR-010**: System MUST display aggregate run statistics: total runs, successful runs, failed runs, cancelled runs, and overall success rate.
- **FR-011**: System MUST display run trend data grouped by day for a configurable time range (last 24 hours, 7 days, 30 days, all time).
- **FR-012**: System MUST display per-pipeline statistics including run count, success rate, average duration, and average token consumption.
- **FR-013**: System MUST display per-step performance statistics including average duration, average token usage, and success rate, using data from the existing `performance_metric` table.
- **FR-014**: System MUST support time range filtering for all statistics views.

#### Run Introspection

- **FR-015**: System MUST display a chronological event timeline for each run showing all events with timestamps, state transitions, step assignments, messages, and token deltas.
- **FR-016**: System MUST display step-level introspection including contract validation results (pass/fail with error details), persona configuration, performance metrics, and recovery hints.
- **FR-017**: System MUST display failure details prominently: failure reason classification (timeout, context exhaustion, general error), error message, and recovery hints/remediation suggestions.
- **FR-018**: System MUST display artifacts produced by each step, including artifact name, type, file size, and a text preview for text-based artifacts (consistent with the existing artifact browsing from spec 085).
- **FR-019**: System MUST support drill-down navigation from run overview to step detail and back.

#### Meta Information

- **FR-020**: System MUST display pipeline metadata (name, description) in the pipeline detail and list views.
- **FR-021**: System MUST display persona metadata (name, description, adapter, model) in the persona detail and list views.
- **FR-022**: System MUST display the most recent run status and timestamp in the pipeline detail view, providing at-a-glance operational status.
- **FR-023**: System MUST display pipeline input configuration details, including input examples when defined, in the pipeline detail view.

#### Workspace Browsing

- **FR-024**: System MUST provide a workspace file tree browser in the run detail view, showing the directory structure of the step workspace.
- **FR-025**: System MUST display file contents with syntax highlighting for recognized file types (Go, YAML, JSON, Markdown, JavaScript, CSS, HTML, SQL, Shell script).
- **FR-026**: System MUST use lazy-loading for directory tree expansion (load subdirectory contents on click, not upfront).
- **FR-027**: All workspace browsing MUST be strictly read-only. No file modification operations are exposed.
- **FR-028**: System MUST gracefully handle missing workspaces (cleaned up after run), displaying a clear unavailability message.

#### Cross-Cutting

- **FR-029**: All new views MUST be gated by the existing `webui` build tag. Building without the tag MUST NOT include any new code or assets.
- **FR-030**: All new frontend assets MUST be embedded via `go:embed`, maintaining the zero-external-dependency deployment model.
- **FR-031**: All content displayed from user-generated sources (artifacts, prompts, YAML files) MUST be sanitized to prevent XSS attacks.
- **FR-032**: New API endpoints MUST follow the existing authentication pattern — requiring bearer token for non-localhost bindings.

### Non-Functional Requirements

- **NFR-001**: Total JavaScript bundle size (including new rendering libraries) MUST remain under 50 KB gzipped, per the constraint from spec 085.
- **NFR-002**: Statistics queries MUST return within 500ms for databases containing up to 10,000 pipeline runs.
- **NFR-003**: Workspace file tree listing MUST return within 200ms for directories containing up to 500 entries.
- **NFR-004**: All new views MUST be responsive and usable on desktop and tablet screen sizes, consistent with spec 085 NFR-003.
- **NFR-005**: Syntax highlighting MUST NOT require external network requests or CDN resources.

### Key Entities

- **Pipeline Configuration**: A pipeline definition loaded from YAML including steps, dependencies, input schema, and contract references. Displayed in the inspection view.
- **Persona Configuration**: A persona definition from the manifest including system prompt, permissions, adapter settings, and hooks. Displayed in the persona detail view.
- **Contract Definition**: A validation contract attached to a pipeline step, specifying output schema and validation rules. Displayed in both pipeline inspection and run introspection.
- **Run Statistics**: Aggregate metrics computed from `pipeline_run` and `performance_metric` tables — counts, rates, averages, and trends over time.
- **Step Performance**: Per-step historical metrics from the `performance_metric` table including duration, tokens, files modified, and success rate.
- **Event Timeline**: Ordered list of `event_log` records for a specific run, providing a chronological narrative of execution.
- **Workspace Tree**: A directory/file hierarchy of a pipeline step's execution workspace, available for browsing when the workspace still exists on disk.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Users can navigate from the pipeline list to a pipeline detail view and see all step configurations, persona assignments, and contract definitions within 2 clicks.
- **SC-002**: Statistics page displays aggregate counts (total, succeeded, failed, cancelled) and per-pipeline breakdowns that match the actual data in the state database with 100% accuracy.
- **SC-003**: Run introspection provides full event timeline, step-level contract validation results, failure reasons, and recovery hints for any completed or failed pipeline run.
- **SC-004**: Markdown content (system prompts, artifacts) renders with correct formatting and the raw/rendered toggle works in both directions without page reload.
- **SC-005**: YAML and JSON content displays with syntax highlighting and the formatted/raw toggle works without page reload.
- **SC-006**: Workspace file tree browser loads directory listings in under 200ms and displays file contents with syntax highlighting for supported file types.
- **SC-007**: All new views work within the existing 50 KB gzipped JavaScript budget — total bundle size verified at build time.
- **SC-008**: All new features are gated behind the `webui` build tag — building without the tag produces no increase in binary size.
- **SC-009**: No new external runtime dependencies are introduced — all assets remain embedded in the binary.
- **SC-010**: All displayed content is XSS-safe — no user-generated content is rendered as executable HTML or JavaScript.

## Clarifications

### C-001: Markdown rendering approach (FR-009) — RESOLVED

**Ambiguity**: FR-009 requires markdown rendering with a raw/rendered toggle. Two approaches are possible: client-side JavaScript parser or server-side Go template rendering. The 50 KB gzipped JS budget from spec 085 constrains the choice.

**Resolution**: Use a lightweight client-side markdown parser (~5-8 KB gzipped). The parser targets only the markdown subset needed for system prompts and artifacts: headings, lists, code blocks, emphasis, links, and tables. A minimal `marked.js` variant or purpose-built parser is appropriate.

**Rationale**: Client-side rendering allows the raw/rendered toggle to work without additional network requests, matching the toggle behavior specified in FR-006. The existing JS assets (`app.js`, `sse.js`, `dag.js`) are well under 50 KB combined, leaving ample budget for a markdown parser. Server-side rendering was rejected because it would require a round-trip per toggle action, violating the "no additional network requests" requirement in FR-009.

### C-002: Syntax highlighting approach (FR-007, FR-023)

**Ambiguity**: Syntax highlighting is required for YAML, JSON, and source code, but no library is specified.

**Resolution**: Recommend a lightweight client-side syntax highlighter using CSS classes and regex-based tokenization. This avoids heavy libraries like Prism.js or highlight.js. A custom tokenizer for the supported languages (Go, YAML, JSON, Markdown, JS, CSS, HTML, SQL, Shell) can be implemented in under 5 KB gzipped.

**Rationale**: The supported language set is known and finite. A custom solution keeps the JS budget well within limits and avoids pulling in a general-purpose library that supports hundreds of languages. Server-side highlighting via Go templates is an alternative but reduces toggle interactivity.

### C-003: Historical vs current configuration in run introspection (Edge Case)

**Ambiguity**: The edge case asks whether run detail shows configuration "as it was" or "as it is now."

**Resolution**: The run detail view shows current configuration annotated with a notice when historical configuration is not available. The existing `event_log` table stores persona and step data per event, which provides partial historical context. Full pipeline/persona YAML snapshots are not stored.

**Rationale**: Storing full configuration snapshots per run would require schema changes and significant storage overhead. The current event log captures persona name, step ID, and key metadata per event, which provides sufficient historical context for debugging. A future enhancement could add configuration snapshotting if needed.

### C-004: Entity metadata fields — last changed, created date, version, author (FR-020 through FR-023) — RESOLVED

**Ambiguity**: The GitHub issue requests "last changed, created date, version, author" metadata for pipelines, personas, and contracts, plus "usage examples where applicable."

**Resolution**: The fields `last changed`, `created date`, `version`, and `author` are explicitly **out of scope** for this feature. The current manifest and pipeline data models (`Manifest`, `Persona`, `Pipeline`, `ContractConfig` types in `internal/manifest/types.go` and `internal/pipeline/types.go`) do not store these fields. This spec addresses metadata that IS available: name, description, adapter, model, permissions, relationships between entities, and operational status derived from run history (most recent run time, status). Input examples are addressed via `InputConfig.Example` when defined in pipeline YAML. Adding these metadata fields requires a separate schema change tracked as a follow-up issue.

**Rationale**: Adding metadata fields that don't exist in the data model would require changes to the manifest schema (`internal/manifest/types.go`), pipeline schema (`internal/pipeline/types.go`), YAML parsing logic, and all existing pipeline/manifest YAML files. This is a separate concern from the dashboard UI feature and should be addressed independently if desired. The dashboard can only display data that exists.

### C-005: Statistics query architecture (FR-010 through FR-014)

**Ambiguity**: The spec requires aggregate statistics (total runs, success rate, per-day trends, per-pipeline breakdowns) but the current `StateStore` interface (`internal/state/store.go`) has no aggregate query methods. It provides `ListRuns` (individual records) and `GetStepPerformanceStats` (per-step only), but no pipeline-level aggregate queries. The spec doesn't specify whether aggregation happens server-side via new SQL queries or client-side from raw run data.

**Resolution**: Server-side aggregation via new `StateStore` methods. The implementation MUST add new query methods to the `StateStore` interface for:
- Aggregate run counts by status (total, succeeded, failed, cancelled) with time range filtering
- Per-day run count and success rate grouping for trend data
- Per-pipeline aggregate statistics (run count, success rate, average duration, average tokens)

These queries will use SQL `GROUP BY`, `COUNT`, `SUM`, and `AVG` aggregations directly in SQLite, which is efficient for datasets up to 10,000 runs (per NFR-002).

**Rationale**: Server-side SQL aggregation is the correct approach because: (1) SQLite efficiently handles aggregate queries, (2) transferring thousands of raw `RunRecord` objects to the client for aggregation would violate NFR-002's 500ms response time requirement for large datasets, (3) the existing `GetStepPerformanceStats` method already demonstrates the pattern of SQL-level aggregation in the codebase, and (4) the `ListRuns` method's time-range filtering pattern can be extended for aggregate queries.

### C-006: Recovery hints availability in run introspection (FR-017)

**Ambiguity**: FR-017 requires displaying recovery hints for failed steps. The `Event` struct in `internal/event/emitter.go` includes `RecoveryHints []RecoveryHintJSON`, but the `event_log` table schema and `LogRecord` type in `internal/state/types.go` do not store recovery hints. The `LogEvent` method only persists `message`, `state`, `persona`, `tokens_used`, and `duration_ms`. Recovery hints emitted during execution are not currently persisted to the database.

**Resolution**: Recovery hints will be sourced from the `message` field of failed events in the `event_log` table, where the pipeline executor already encodes failure context (including hint text) into the message string. For richer recovery hint display, the implementation MAY add a `recovery_hints_json` column to the `event_log` table via a database migration, but this is an optional enhancement — not a blocking requirement. The minimum viable implementation extracts hints from the existing `message` field and from the `recovery` package's `ClassifyAndFormat` function applied to error messages at display time.

**Rationale**: The `buildStepDetails` method in `internal/webui/handlers_runs.go` already captures `ev.Message` for failed steps as `si.errMsg`. The recovery package (`internal/recovery/`) provides `ClassifyAndFormat` to regenerate hints from error text. Adding a dedicated column would provide higher fidelity but requires a schema migration, which is better tracked as a follow-up enhancement rather than a blocking dependency for the dashboard UI.

### C-007: Workspace path resolution for file browsing (FR-024)

**Ambiguity**: FR-024 requires a workspace file tree browser in the run detail view, but the spec doesn't specify how workspace paths are resolved. The `StepStateRecord` type has a `WorkspacePath` field, but the current `buildStepDetails` method in `internal/webui/handlers_runs.go` derives step state from the `event_log` table (not `step_state`), and the `StepDetail` API response type does not include a workspace path field. Additionally, the `step_state` table has a unique constraint on `step_id` alone (not per-run), causing cross-run collisions as noted in the codebase comments.

**Resolution**: Workspace paths for file browsing will be resolved through a combination of:
1. Adding a `WorkspacePath` field to the `StepDetail` response type in `internal/webui/types.go`
2. Querying workspace paths from the `step_state` table (which does store `workspace_path`) as a supplementary data source alongside event-derived state
3. Falling back to convention-based path construction (`{workspace_root}/{run_id}/{step_id}`) when `step_state` data is unavailable

The workspace browser API endpoint will accept a `run_id` and `step_id`, resolve the workspace path, validate it exists on disk, and serve directory listings and file contents. Path traversal prevention MUST be enforced per the existing security model in `internal/security/`.

**Rationale**: The `step_state` table's `workspace_path` column is populated by the pipeline executor when a step starts. Despite the cross-run collision issue noted in the `buildStepDetails` comments, the `workspace_path` for the most recent run is still valid for browsing. The `Server` already has access to a `workspace.WorkspaceManager` (initialized in `server.go:73`), providing an existing integration point for workspace operations.
