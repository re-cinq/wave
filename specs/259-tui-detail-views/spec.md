# Feature Specification: TUI Alternative Master-Detail Views

**Feature Branch**: `259-tui-detail-views`  
**Created**: 2026-03-06  
**Status**: Draft  
**Issue**: [#259](https://github.com/re-cinq/wave/issues/259) (part 8 of 10, parent: [#251](https://github.com/re-cinq/wave/issues/251))  
**Input**: Implement four alternative views accessible via `Tab` cycling: **Personas**, **Contracts**, **Skills**, and **Health**. Each view follows the same two-pane master-detail pattern as the Pipelines view. Tab cycles through: Pipelines → Personas → Contracts → Skills → Health → Pipelines. View state (selection, scroll position) is preserved when switching tabs.

## Clarifications

The following ambiguities were identified and resolved during specification refinement:

### C1: Tab key handling — conflict with existing form navigation

**Ambiguity**: The `Tab` key is used for view cycling (Pipelines → Personas → ...) but is also used by `huh` forms for field navigation (`Tab`/`Shift+Tab` to move between form fields) in the pipeline launch flow (#256). If the user has a launch form open and presses `Tab`, should it switch views or move to the next form field?

**Resolution**: Tab for view cycling is only active when the left pane is focused or the right pane is focused in a non-form state. When `formActive` is true (i.e., a huh form is displayed), `Tab` is forwarded to the form for field navigation. Similarly, when `liveOutputActive` is true, `Tab` is consumed for view cycling since live output does not use `Tab`. The content model checks the current detail pane state before deciding whether to handle `Tab` as a view switch or forward it. The status bar already shows context-appropriate hints — when a form is active, hints show "Tab: next" (form navigation); when no form is active, the hints will include "Tab: view" for view cycling.

### C2: View switching scope — AppModel vs ContentModel

**Ambiguity**: The issue says `Tab` cycles between views. The current `ContentModel` is tightly coupled to pipelines (it owns a `PipelineListModel` and `PipelineDetailModel`). Should view switching happen at the `AppModel` level (replacing the entire content area) or within `ContentModel` (swapping child models)?

**Resolution**: View switching happens at the `ContentModel` level. `ContentModel` gains a `currentView` field (enum: `ViewPipelines`, `ViewPersonas`, `ViewContracts`, `ViewSkills`, `ViewHealth`) and holds lazy-initialized view models for each alternative view. When `Tab` is pressed and the view-switch precondition is met (C1), `ContentModel` increments `currentView` and renders the appropriate child models. This approach keeps `AppModel` simple (header + content + statusbar) and allows `ContentModel` to own focus management across all views consistently. Each alternative view consists of a list model (left pane) and a detail model (right pane), following the same `SetSize`/`Update`/`View` pattern as the pipeline models. View models are created on first access (lazy init) so views the user never visits incur no data-fetching overhead.

### C3: View state preservation — model retention vs recreation

**Ambiguity**: The issue says "view state preserved when switching between tabs (selection, scroll position)." Should view models be retained in memory across tab switches, or recreated with cached data?

**Resolution**: View models are retained in memory for the lifetime of the TUI session. When the user tabs away from a view, the model is not destroyed — it simply stops receiving key messages and its `View()` is not called. When the user tabs back, the model's state (cursor position, scroll offset, selected item, loaded data) is exactly as they left it. This is the simplest approach and consistent with how Bubble Tea models work — they are plain structs held by value or pointer. Memory overhead is minimal since each view holds a small list and detail projection. Data refresh tickers (if any) continue running in the background so data stays current even for non-active views.

### C4: Persona run stats — data source and aggregation

**Ambiguity**: The issue says the Personas view should show "run stats (total runs, avg duration, success rate)" but `StepPerformanceStats` is indexed by `(pipeline_name, step_id)`, not by persona name directly. A persona may be used across multiple pipelines and steps. How should stats be aggregated?

**Resolution**: Aggregate stats across all `performance_metric` rows where the `persona` column matches the persona name, regardless of pipeline or step. The query groups by `persona` and computes: `COUNT(*) as total_runs`, `SUM(CASE WHEN success=1 THEN 1 ELSE 0 END) as successful_runs`, `AVG(duration_ms)`, and `MAX(started_at) as last_run`. This requires a new state store method (e.g., `GetPersonaPerformanceStats(personaName string) (*PersonaStats, error)`) or reuse of `GetRecentPerformanceHistory` with persona filtering. The `PersonaStats` struct contains: `TotalRuns`, `SuccessfulRuns`, `AvgDurationMs`, `LastRunAt`. Success rate is computed at render time: `(SuccessfulRuns / TotalRuns) * 100`. If no performance data exists (persona has never been run), stats show "No runs recorded."

### C5: Contract data source — manifest vs filesystem

**Ambiguity**: The issue says the Contracts view lists "contract files" with type, schema preview, and pipeline usage. Contracts are defined inline in pipeline step YAML (`handover.contract`) and may reference external schema files (`schema_path`). There is no top-level `contracts:` section in `wave.yaml`. Should the view list distinct contract configurations from all pipeline steps, or list `.json` files found in `.wave/contracts/`?

**Resolution**: Enumerate contracts by scanning all pipeline step definitions in the manifest. For each step that has a `Handover.Contract` config, extract the contract type, schema path (if any), and associate it with the pipeline name and step ID. Deduplicate by schema path (two steps referencing the same schema file appear as one entry with multiple "used by" references). Inline contracts (no `SchemaPath`) appear as entries named after their step ID. The schema preview is loaded from `SchemaPath` if it exists, or shows the inline `Source` content. This approach captures all contracts actually in use and their pipeline context, without requiring a separate contracts directory convention.

### C6: Skills data source — manifest pipeline requires vs skill package

**Ambiguity**: Skills are declared in pipeline YAML under `requires.skills` with `install`, `check`, and `commands_glob` fields. The `internal/skill/` package handles provisioning. The issue says the Skills view should show "source path, available commands, pipeline usage." What "source path" means for skills is unclear — skills don't have a single source file.

**Resolution**: The "source path" for a skill is its `commands_glob` pattern (e.g., `.claude/commands/speckit.*.md`), which identifies where the skill's command files live. "Available commands" are discovered by resolving the glob pattern against the filesystem. Each skill entry shows: name, commands glob path, resolved command file names, and which pipelines require it. Data comes from two sources: the manifest's `Pipeline.Requires.Skills` map for pipeline associations, and the filesystem (glob resolution) for discovered commands. Skills that appear in multiple pipelines are listed once with all pipeline references.

### C7: Health checks — what constitutes a "check"

**Ambiguity**: The issue shows health checks like "Git Repository," "GitHub Connection," "Claude Code Adapter," "SQLite Database," "Wave Configuration," and "Workspace Permissions" with OK/WARN/FAIL status. The existing preflight system (`internal/preflight/`) only checks tool availability and skill installation. The header already aggregates pipeline health into a single `HealthStatus` (OK/WARN/ERR). What specific checks should the Health view perform?

**Resolution**: Define a fixed set of health checks that cover the major system dependencies. Each check has a name, description, check function, and result (status + details). The checks are:

1. **Git Repository** — `git rev-parse --is-inside-work-tree`: status, remote URL, branch, clean/dirty, hooks
2. **Adapter Binary** — check each adapter binary exists on PATH (from `manifest.Adapters`): binary name, version (if available)
3. **SQLite Database** — attempt to open the state store and run a test query: path, table count, size
4. **Wave Configuration** — validate `wave.yaml` loads without error: persona count, pipeline count, adapter count
5. **Required Tools** — run preflight tool checks for all pipelines: tool name, availability
6. **Required Skills** — run preflight skill checks for all pipelines: skill name, installed status

Each check returns `HealthOK`, `HealthWarn`, or `HealthErr` with a detail message. The left pane shows all checks with status icons (● OK, ▲ WARN, ✗ FAIL). The right pane shows the selected check's details and diagnostic info. Checks are performed once on first view access and can be re-run with `r` key.

### C8: Health check timing — blocking vs async

**Ambiguity**: Some health checks may be slow (network calls to GitHub, database operations). Should the Health view block on checks or load them asynchronously?

**Resolution**: Health checks run asynchronously. On first access to the Health view, all checks are launched in parallel via `tea.Cmd` closures. The left pane initially shows all checks in a "checking..." state. As each check completes, a `HealthCheckResultMsg` message updates the corresponding entry. This follows the same async pattern used by the header's metadata provider (`GitStateMsg`, `ManifestInfoMsg`, `GitHubInfoMsg`). A "last checked" timestamp is stored per-check and displayed in the right pane. The `r` key re-runs all checks.

### C9: Status bar context label — updating with current view

**Ambiguity**: The status bar's `contextLabel` currently shows "Dashboard". The issue says the status bar should update to show the current view name. Should the label change to "Personas", "Contracts", etc.?

**Resolution**: Yes. When the view changes, a `ViewChangedMsg{View: ViewPersonas}` (or similar) is emitted and forwarded to the status bar. The status bar updates its `contextLabel` to the view name: "Pipelines", "Personas", "Contracts", "Skills", or "Health". The status bar also updates its hints based on the current view: all views share the basic "↑↓: navigate Enter: view Esc: back Tab: view q: quit" pattern, with view-specific hints added as needed (e.g., "r: recheck" for Health view).

### C10: Navigation keys in alternative views — consistency with pipeline view

**Ambiguity**: The issue says "all views support arrow key navigation and scrollable content." Should the alternative views support the same full navigation model as pipelines (Enter to focus right pane, Esc to return, filtering with `/`)?

**Resolution**: Yes, for consistency. All alternative views support: ↑/↓ to navigate the left pane list, Enter to focus the right pane detail, Esc to return to the left pane, and scrollable right pane content via viewport. The `/` filter is supported on views where it's useful (Personas, Contracts, Skills lists). The Health view does not support filtering (fixed set of checks) but adds `r` to re-run checks. The right pane in alternative views is read-only — no forms, no actions, just scrollable content. This means `FormActiveMsg`, `LiveOutputActiveMsg`, and `FinishedDetailActiveMsg` are never emitted by alternative views.

### C11: Pipeline usage cross-reference — computation

**Ambiguity**: The Personas, Contracts, and Skills views all show "pipeline usage" (which pipelines use a given persona/contract/skill). How is this computed?

**Resolution**: Pipeline usage is computed by scanning the manifest's pipeline definitions at data-load time. For personas: iterate all pipeline steps, collect `step.Persona` values, map each persona name to the list of `(pipeline_name, step_id)` pairs that reference it. For contracts: iterate all pipeline steps with `Handover.Contract`, map schema paths to `(pipeline_name, step_id)` pairs. For skills: iterate all pipeline `Requires.Skills` maps, collect pipeline names per skill. This is a pure manifest scan — no database queries needed. The computation happens once during the initial data fetch and is cached in the view model.

### C12: Alternative view detail pane — rendering approach

**Ambiguity**: The pipeline detail pane uses `stateAvailableDetail`, `stateFinishedDetail`, etc. as rendering states with a `viewport` for scrolling. Should alternative view detail panes use the same state machine or a simpler approach?

**Resolution**: Alternative view detail panes use a simpler approach — each has only two states: empty (no selection) and showing detail. The detail content is rendered as styled text and set on a viewport for scrolling. No forms, no live output, no state machines with multiple modes. This matches the read-only nature of these views. Each alternative detail model embeds a `viewport.Model` for content scrolling and receives size updates via `SetSize()`. The rendered content updates when the list selection changes (via a selection message similar to `PipelineSelectedMsg`).

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Tab Cycling Between Views (Priority: P1)

A developer launches the TUI and sees the default Pipelines view. They press `Tab` to cycle to the Personas view, which shows a list of all configured personas in the left pane and the selected persona's details in the right pane. They press `Tab` again to see Contracts, then Skills, then Health, and one more `Tab` returns them to Pipelines. The status bar updates to show the current view name at each step.

**Why this priority**: Tab cycling is the foundational navigation mechanism that enables all other alternative views. Without it, none of the other views are accessible.

**Independent Test**: Can be tested by pressing `Tab` repeatedly and verifying the view transitions, status bar label updates, and that each view renders its appropriate content.

**Acceptance Scenarios**:

1. **Given** the TUI is showing the Pipelines view, **When** the user presses `Tab`, **Then** the content area switches to the Personas view and the status bar label changes to "Personas".
2. **Given** the TUI is showing the Health view, **When** the user presses `Tab`, **Then** the content area wraps back to the Pipelines view and the status bar label changes to "Pipelines".
3. **Given** the user is in the Personas view with a persona selected, **When** they press `Tab` to switch to Contracts and then `Tab` back to Personas, **Then** the Personas view shows the same persona still selected.
4. **Given** the right pane is focused (Enter was pressed on a left pane item), **When** the user presses `Tab`, **Then** the view cycles and the focus resets to the left pane of the new view.
5. **Given** a launch form is active in the pipeline detail (formActive=true), **When** the user presses `Tab`, **Then** Tab navigates form fields (not view cycling).

---

### User Story 2 - Personas View with Run Stats (Priority: P1)

A developer tabs to the Personas view to understand the agent roles configured in their project. The left pane lists all personas from `wave.yaml` (e.g., navigator, craftsman, implementer, reviewer). They navigate to "craftsman" and the right pane shows: role description, permissions (allow/deny lists), which pipelines/steps use this persona, and run stats (total runs, average duration, success rate) aggregated from the performance metrics database.

**Why this priority**: Personas are the core concept in Wave — understanding their roles, permissions, and performance is essential for project introspection.

**Independent Test**: Can be tested by loading a manifest with multiple personas, navigating to the Personas view, selecting a persona, and verifying the right pane displays correct role, permissions, pipeline usage, and run stats from the database.

**Acceptance Scenarios**:

1. **Given** the manifest defines 4 personas, **When** the user views the Personas view, **Then** the left pane lists all 4 persona names alphabetically.
2. **Given** "navigator" is selected, **When** the right pane renders, **Then** it shows the persona's description, adapter, model, allowed tools, denied tools, and pipeline usage.
3. **Given** the performance database has runs for "craftsman", **When** "craftsman" is selected, **Then** the right pane shows total runs, success rate as percentage, and average duration formatted as human-readable time (e.g., "2m 34s").
4. **Given** a persona has never been run, **When** it is selected, **Then** the right pane shows "No runs recorded" in the stats section.

---

### User Story 3 - Contracts View with Schema Preview (Priority: P2)

A developer tabs to the Contracts view to inspect the output validation schemas used in their pipelines. The left pane lists all distinct contracts (identified by schema path or step ID for inline contracts). They select a contract and the right pane shows: contract type (json_schema, test_suite, etc.), file path, a preview of the schema content (truncated if long), and which pipeline steps use this contract.

**Why this priority**: Contracts enforce pipeline output quality. Inspecting them helps developers understand what validation is applied at each step, but this is less frequently needed than persona introspection.

**Independent Test**: Can be tested by loading a manifest with pipelines that have contract configurations, navigating to the Contracts view, and verifying the right pane displays contract type, schema content, and pipeline usage.

**Acceptance Scenarios**:

1. **Given** the manifest has 3 distinct contracts across pipelines, **When** the user views the Contracts view, **Then** the left pane lists 3 contract entries.
2. **Given** a contract with `schema_path` pointing to a JSON file, **When** it is selected, **Then** the right pane shows the contract type, file path, and a preview of the JSON schema content.
3. **Given** a contract with inline `source` and no `schema_path`, **When** it is selected, **Then** the right pane shows the inline content as the preview.
4. **Given** two pipeline steps reference the same schema file, **When** the contract is selected, **Then** the "Used by" section lists both pipeline/step pairs.

---

### User Story 4 - Skills View with Command Discovery (Priority: P2)

A developer tabs to the Skills view to see which skills are available and what commands they provide. The left pane lists all skills declared in pipeline `requires.skills` sections. They select a skill and the right pane shows: the commands glob path, resolved command files discovered on the filesystem, and which pipelines require this skill.

**Why this priority**: Skills extend Wave's capabilities. Understanding available commands and their pipeline associations helps developers configure and troubleshoot workflows.

**Independent Test**: Can be tested by loading a manifest with skills declarations, navigating to the Skills view, and verifying the right pane displays command files and pipeline associations.

**Acceptance Scenarios**:

1. **Given** pipelines declare 2 skills, **When** the user views the Skills view, **Then** the left pane lists 2 skill names.
2. **Given** the "speckit" skill has commands_glob `.claude/commands/speckit.*.md`, **When** it is selected, **Then** the right pane shows the glob path and lists all matching command files found on disk.
3. **Given** a skill's command files don't exist on disk, **When** it is selected, **Then** the right pane shows "No commands found" with the glob path for troubleshooting.
4. **Given** two pipelines require "speckit", **When** it is selected, **Then** the "Used by" section lists both pipeline names.

---

### User Story 5 - Health View with Diagnostic Details (Priority: P2)

A developer tabs to the Health view to assess system readiness. The left pane shows health checks with status icons: ● OK (green), ▲ WARN (yellow), ✗ FAIL (red). They select "Wave Configuration" which shows ▲ WARN. The right pane shows details: the configuration loaded successfully but with warnings (e.g., unused persona, missing optional skill). They press `r` to re-run all checks.

**Why this priority**: Health checks provide at-a-glance system readiness and help diagnose issues before running pipelines.

**Independent Test**: Can be tested by running health checks against a known system state and verifying the status icons, detail content, and re-check behavior.

**Acceptance Scenarios**:

1. **Given** the TUI is launched, **When** the user tabs to the Health view, **Then** health checks run asynchronously and the left pane updates as results arrive.
2. **Given** all checks pass, **When** the left pane renders, **Then** all entries show "● OK" with green styling.
3. **Given** the git repository is dirty, **When** "Git Repository" is selected, **Then** the right pane shows branch, remote, and a warning about uncommitted changes.
4. **Given** the user presses `r`, **When** all checks re-run, **Then** the left pane updates with fresh results and timestamps.
5. **Given** an adapter binary is not found, **When** "Adapter Binary" is selected, **Then** the entry shows "✗ FAIL" and the right pane suggests how to install it.

---

### User Story 6 - View-Specific Status Bar Hints (Priority: P3)

When the user is in any alternative view, the status bar shows view-appropriate key binding hints. In the Personas/Contracts/Skills views: "↑↓: navigate Enter: view /: filter Tab: view q: quit". In the Health view: "↑↓: navigate Enter: view r: recheck Tab: view q: quit". When the right pane is focused: "↑↓: scroll Esc: back Tab: view q: quit".

**Why this priority**: Hints improve discoverability but the views are fully functional without them.

**Independent Test**: Can be tested by switching to each view and verifying the status bar content matches the expected hints.

**Acceptance Scenarios**:

1. **Given** the user is in the Personas view with the left pane focused, **When** the status bar renders, **Then** it shows navigation and filter hints including "Tab: view".
2. **Given** the user is in the Health view, **When** the status bar renders, **Then** it includes "r: recheck" hint.
3. **Given** the right pane is focused in the Contracts view, **When** the status bar renders, **Then** it shows scroll and back hints.

---

### Edge Cases

- What happens when the manifest has no personas defined? The Personas left pane shows an empty list with a "No personas configured" placeholder message.
- What happens when no pipelines have contracts? The Contracts left pane shows "No contracts configured."
- What happens when no pipelines declare skills? The Skills left pane shows "No skills configured."
- What happens when health checks are running and the user tabs away? The checks continue running in the background. When the user tabs back, results that arrived while away are already displayed.
- What happens when the user presses `Tab` rapidly? Each `Tab` press increments the view index by one. Rapid pressing cycles through views normally — no debouncing needed since view switching is instantaneous (no async data loading blocks the switch).
- What happens when the terminal is very narrow? Alternative views follow the same min-width (80 columns) constraint as the pipeline view. Below minimum, the "Terminal too small" message is shown.
- What happens when the state database is unavailable? Persona run stats show "Stats unavailable" instead of crashing. Other data (from manifest/filesystem) renders normally.
- What happens when an alternative view's right pane is focused and the user presses `Tab`? Focus returns to the left pane of the new view (C1 resolution).

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: `ContentModel` MUST support a `currentView` field that tracks the active view (Pipelines, Personas, Contracts, Skills, Health). The default view MUST be Pipelines.
- **FR-002**: When `Tab` is pressed and no form is active (`formActive` is false), the system MUST cycle `currentView` to the next view in order: Pipelines → Personas → Contracts → Skills → Health → Pipelines. If the right pane is focused, focus MUST return to the left pane before switching.
- **FR-003**: A `ViewChangedMsg{View ViewType}` MUST be emitted on every view switch. The status bar MUST update its `contextLabel` to reflect the current view name.
- **FR-004**: Each view model MUST be retained in memory across tab switches. View state (cursor position, scroll offset, selected item, loaded data) MUST be preserved when switching away and back.
- **FR-005**: Alternative view models MUST be lazy-initialized — created and data-fetched on first access, not at TUI startup.
- **FR-006**: The **Personas view** left pane MUST list all personas from `manifest.Personas`, sorted alphabetically by name.
- **FR-007**: The **Personas view** right pane MUST display: persona name, description, adapter name, model (if set), allowed tools list, denied tools list, pipeline usage (pipeline/step pairs), and run stats (total runs, success rate, average duration).
- **FR-008**: Persona run stats MUST be aggregated from the `performance_metric` table by grouping on the `persona` column. If the state store is unavailable or no metrics exist, stats MUST show "No runs recorded" rather than erroring.
- **FR-009**: The **Contracts view** left pane MUST list all distinct contracts found across pipeline step definitions, identified by schema path (for file-backed contracts) or by `pipeline:step` label (for inline contracts).
- **FR-010**: The **Contracts view** right pane MUST display: contract name/label, type (`json_schema`, `test_suite`, etc.), schema file path (if applicable), schema content preview (first ~30 lines, scrollable), and pipeline usage (pipeline/step pairs).
- **FR-011**: Contracts sharing the same `SchemaPath` MUST be deduplicated into a single entry with combined pipeline usage references.
- **FR-012**: The **Skills view** left pane MUST list all skills declared in any pipeline's `requires.skills` map, deduplicated by skill name.
- **FR-013**: The **Skills view** right pane MUST display: skill name, commands glob pattern, discovered command files (resolved from glob), install command, check command, and pipeline usage.
- **FR-014**: The **Health view** left pane MUST list health checks with status icons: `●` for OK (green), `▲` for WARN (yellow), `✗` for FAIL (red). Checks run asynchronously on first view access.
- **FR-015**: The **Health view** MUST include checks for: git repository status, adapter binary availability, state database connectivity, wave configuration validity, required tools availability, and required skills installation.
- **FR-016**: The **Health view** right pane MUST display: check name, status, diagnostic details, and "last checked" timestamp.
- **FR-017**: The `r` key in the Health view MUST re-run all health checks and update results.
- **FR-018**: All alternative views MUST support arrow key navigation (↑/↓) in the left pane and Enter/Esc for right pane focus management, consistent with the pipeline view.
- **FR-019**: The Personas, Contracts, and Skills views MUST support `/` filter in the left pane, filtering items by name substring.
- **FR-020**: All alternative view right panes MUST be scrollable via viewport when content exceeds the visible area.
- **FR-021**: Each alternative view MUST provide a data provider interface for testability, following the established `PipelineDataProvider`/`DetailDataProvider` pattern.
- **FR-022**: The `q` key MUST quit the TUI from any view, not just Pipelines.
- **FR-023**: `Ctrl-C` MUST trigger graceful shutdown from any view.

### Key Entities

- **ViewType**: Enum identifying the active view — `ViewPipelines`, `ViewPersonas`, `ViewContracts`, `ViewSkills`, `ViewHealth`.
- **ViewChangedMsg**: Message emitted on view switch, carrying the new `ViewType`. Consumed by the status bar and header (for context awareness).
- **PersonaListModel**: Left pane model for the Personas view. Holds a sorted list of persona names with cursor navigation and filtering.
- **PersonaDetailModel**: Right pane model. Displays persona metadata, permissions, pipeline usage, and run stats in a scrollable viewport.
- **PersonaInfo**: Data projection for a persona — name, description, adapter, model, permissions, pipeline usage (computed from manifest scan).
- **PersonaStats**: Run statistics — total runs, successful runs, average duration (ms), last run timestamp. Aggregated from `performance_metric` table.
- **ContractListModel**: Left pane model for the Contracts view. Lists distinct contracts with cursor navigation and filtering.
- **ContractDetailModel**: Right pane model. Displays contract type, path, schema preview, and pipeline usage.
- **ContractInfo**: Data projection — label, type, schema path, source content, pipeline usage pairs.
- **SkillListModel**: Left pane model for the Skills view. Lists skill names with cursor navigation and filtering.
- **SkillDetailModel**: Right pane model. Displays glob path, discovered commands, install/check commands, pipeline usage.
- **SkillInfo**: Data projection — name, commands glob, resolved command files, install cmd, check cmd, pipeline usage.
- **HealthListModel**: Left pane model for the Health view. Lists checks with status icons. Supports `r` to re-run.
- **HealthDetailModel**: Right pane model. Displays check details and diagnostics.
- **HealthCheck**: Data structure — name, status (OK/WARN/FAIL), message, details map, last-checked timestamp.
- **HealthCheckResultMsg**: Async message carrying the result of a single health check. Identified by check name.
- **PersonaDataProvider**: Interface for persona data fetching — `FetchPersonas() []PersonaInfo` and `FetchPersonaStats(name string) (*PersonaStats, error)`. Enables mock testing of the Personas view.
- **ContractDataProvider**: Interface for contract data fetching — `FetchContracts() []ContractInfo`. Enables mock testing of the Contracts view.
- **SkillDataProvider**: Interface for skill data fetching — `FetchSkills() []SkillInfo`. Enables mock testing of the Skills view.
- **HealthDataProvider**: Interface for health check execution — `RunHealthCheck(name string) HealthCheckResult`. Each check runs independently. Enables mock testing of the Health view.

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: `Tab` cycles through all 5 views in order and wraps back to Pipelines — verified by unit tests sending `Tab` key events and asserting `currentView` transitions.
- **SC-002**: View state (cursor, scroll, selection) is preserved across tab switches — verified by tests that select an item, switch away, switch back, and assert the selection remains.
- **SC-003**: The Personas view lists all manifest personas and displays correct metadata (description, permissions, pipeline usage) for the selected persona — verified by tests with a mock manifest containing known personas.
- **SC-004**: Persona run stats display correct aggregated values from the performance database — verified by tests with mock state store returning known performance records.
- **SC-005**: The Contracts view correctly deduplicates contracts by schema path and shows combined pipeline usage — verified by tests with a manifest containing shared and unique contracts.
- **SC-006**: The Skills view discovers command files by resolving glob patterns and associates skills with their declaring pipelines — verified by tests with mock filesystem and manifest.
- **SC-007**: Health checks run asynchronously and display status icons (OK/WARN/FAIL) with diagnostic details — verified by tests with mock check functions returning known results.
- **SC-008**: The `r` key in the Health view re-runs all checks and updates the display — verified by tests asserting re-execution and result update.
- **SC-009**: The status bar context label updates to the correct view name on every tab switch — verified by unit tests asserting `contextLabel` after `ViewChangedMsg`.
- **SC-010**: All existing TUI tests continue to pass — the alternative views do not break pipeline list, detail, header, status bar, launch flow, live output, or finished pipeline action components.
- **SC-011**: `Tab` is forwarded to forms when `formActive` is true, not consumed for view switching — verified by tests with formActive=true asserting no view change.
- **SC-012**: Alternative views support ↑/↓ navigation, Enter/Esc focus management, and `/` filtering (where applicable) — verified by integration tests exercising the navigation model.

### C13: Persona run stats — query method

**Ambiguity**: The spec mentions "a new state store method (e.g., `GetPersonaPerformanceStats(personaName string)`) or reuse of `GetRecentPerformanceHistory` with persona filtering." Which approach should be used?

**Resolution**: Use the existing `GetRecentPerformanceHistory(PerformanceQueryOptions{Persona: personaName})` method — it already supports filtering by persona and returns `[]PerformanceMetricRecord`. The TUI data provider aggregates the results in-process: iterate returned records, count total/successful, compute average duration, find max `StartedAt`. No new `StateStore` interface method is needed. This keeps the state store interface stable and follows the pattern already used by `FetchFinishedDetail` in `pipeline_detail_provider.go`, which fetches raw records and projects them into TUI-specific structs. The aggregation is trivially O(n) where n is small (persona run count).

### C14: `q` quit across alternative views — filtering guard

**Ambiguity**: The current quit logic in `app.go:58` checks `!m.content.list.filtering && m.content.focus == FocusPaneLeft`. The `m.content.list` field is the `PipelineListModel`. When alternative views are active with their own list models, the pipeline list's `filtering` field is irrelevant. Pressing `q` while filtering in the Personas view should not quit, but the current guard only checks the pipeline list.

**Resolution**: `ContentModel` exposes an `IsFiltering() bool` method that delegates to the *active* view's list model. When `currentView == ViewPipelines`, it returns `m.list.filtering`. For alternative views, it returns the active list model's filtering state. `AppModel`'s quit guard changes from `!m.content.list.filtering` to `!m.content.IsFiltering()`. Similarly, `ContentModel` exposes `CurrentFocus() FocusPane` so AppModel doesn't reach into content internals. This maintains encapsulation as ContentModel gains complexity.

### C15: ViewDataProvider — single interface vs per-view interfaces

**Ambiguity**: The Key Entities section defines a single `ViewDataProvider` interface with `FetchPersonas()`, `FetchContracts()`, `FetchSkills()`, `RunHealthChecks()`. But FR-021 says "each alternative view MUST provide a data provider interface for testability, following the established PipelineDataProvider/DetailDataProvider pattern." Is it one combined interface or separate per-view interfaces?

**Resolution**: Separate per-view interfaces, following the established pattern. `PipelineDataProvider` and `DetailDataProvider` are separate focused interfaces — alternative views follow suit. Define: `PersonaDataProvider` (with `FetchPersonas()` and `FetchPersonaStats(name string)`), `ContractDataProvider` (with `FetchContracts()`), `SkillDataProvider` (with `FetchSkills()`), and `HealthDataProvider` (with `RunHealthCheck(name string) HealthCheckResult`). Each view model receives its own provider, enabling independent mocking in tests. The `ViewDataProvider` entity in Key Entities is removed in favor of these four specific interfaces.

### C16: Tab routing when right pane is focused — interception before child routing

**Ambiguity**: Acceptance scenario US1.4 states "Given the right pane is focused, When the user presses Tab, Then the view cycles and the focus resets to the left pane of the new view." However, the current `ContentModel.Update` routes ALL key messages to the focused child when `m.focus == FocusPaneRight` (content.go:138-142). This means Tab would be consumed by the detail model (for forms) or ignored (for viewports) rather than triggering a view switch.

**Resolution**: `ContentModel.Update` intercepts `Tab` key presses *before* the focus-based routing. The logic is: (1) if `Tab` is pressed and the current right-pane child is in form mode (`stateConfiguring` for pipelines, never for alternative views), forward Tab to the child for form field navigation; (2) otherwise, consume Tab for view cycling, resetting focus to left pane of the new view. This requires checking whether the active detail pane has an active form. For alternative views (which never have forms per C12), Tab always cycles. For the pipeline view, Tab cycles unless `paneState == stateConfiguring`. This check replaces the `formActive` boolean concept with a direct state check on the active detail model, keeping the logic co-located in `ContentModel`.

### C17: Shift+Tab reverse view cycling

**Ambiguity**: The spec defines `Tab` for forward cycling through views (Pipelines → Personas → ... → Pipelines) but does not mention `Shift+Tab`. Standard tabbed interfaces in terminals typically support reverse cycling with `Shift+Tab`. Should this be supported?

**Resolution**: No. Forward-only cycling with `Tab` is sufficient for 5 views — worst case, the user presses `Tab` 4 times to reach the "previous" view. Adding `Shift+Tab` complicates the implementation because `Shift+Tab` is the key used by `huh` forms for backward field navigation (see `statusbar.go:65`). Intercepting `Shift+Tab` before it reaches the form would require duplicating the form-active guard logic for a second key. The 5-view cycle is short enough that forward-only cycling is ergonomically acceptable. If reverse cycling is requested later, it can be added as a follow-up without breaking changes.
