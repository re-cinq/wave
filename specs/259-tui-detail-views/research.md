# Research: TUI Alternative Master-Detail Views

**Date**: 2026-03-06  
**Feature**: #259 — TUI Alternative Master-Detail Views  
**Branch**: `259-tui-detail-views`

## R1: View Switching Architecture within ContentModel

**Decision**: View switching happens within `ContentModel` via a `currentView ViewType` enum field, not by replacing `ContentModel` at the `AppModel` level.

**Rationale**: The existing `AppModel` is a simple 3-row layout (header + content + statusbar). View switching is a content-area concern. `ContentModel` already owns focus management between list/detail panes — extending it with view awareness is the natural extension. Each view consists of its own list+detail model pair, following the same structural pattern as `PipelineListModel` + `PipelineDetailModel`.

**Alternatives Rejected**:
- **AppModel-level switching**: Would require AppModel to understand view-specific focus states, key routing, and data providers. Breaks separation of concerns.
- **Single multiplexed model**: One list/detail model that switches data based on view. Violates view state preservation requirement (cursor, scroll, selection must survive tab switches).

## R2: Tab Key Interception Strategy

**Decision**: `ContentModel.Update()` intercepts `Tab` key presses *before* focus-based child routing. Tab cycles views unless the active right pane has an active form (`paneState == stateConfiguring` for pipeline view; never for alternative views).

**Rationale**: The current key routing sends all key messages to the focused child when `focus == FocusPaneRight` (content.go:138-142). Tab must be intercepted before this point to enable view cycling from both panes. The form-active guard is implemented as a direct state check on the pipeline detail model (checking `paneState == stateConfiguring`), not as a boolean flag, to keep the logic co-located.

**Alternatives Rejected**:
- **Boolean `formActive` tracking**: Adds another state variable that must be kept in sync with the detail model's actual state. Direct state check is simpler and always correct.
- **Tab only from left pane**: Spec explicitly requires Tab to work from the right pane too (US1.4), resetting focus to the left pane of the new view.

## R3: View Model Lifecycle — Lazy Init with Retention

**Decision**: View models are created on first access (lazy initialization) and retained in memory for the entire TUI session. Data is fetched once on init and cached within the model.

**Rationale**: Most users won't visit all 5 views in a session. Lazy init avoids unnecessary data loading at startup. Retention preserves view state (cursor, scroll, selection) across tab switches without serialization/deserialization overhead. Memory footprint is minimal — each view holds a small list and a viewport with rendered text.

**Alternatives Rejected**:
- **Eager init at startup**: Wastes time fetching data for views the user may never visit.
- **Recreate on each visit**: Loses view state; requires implementing state save/restore protocol for each view.

## R4: Persona Run Stats — Aggregation via GetRecentPerformanceHistory

**Decision**: Use existing `GetRecentPerformanceHistory(PerformanceQueryOptions{Persona: personaName})` method from `StateStore`. Aggregate results in-process: iterate records, count total/successful, compute average duration, find max `StartedAt`.

**Rationale**: The `StateStore` interface already supports filtering by persona via `PerformanceQueryOptions`. No new interface methods needed. The aggregation is O(n) where n is small (persona run history). This follows the same pattern as `FetchFinishedDetail` in `pipeline_detail_provider.go`, which fetches raw records and projects them into TUI-specific structs.

**Alternatives Rejected**:
- **New `GetPersonaPerformanceStats()` method**: Requires modifying the `StateStore` interface and SQLite implementation — unnecessary coupling for a simple aggregation.
- **SQL-level aggregation**: Would need a new query in store.go. The data volume is too small to justify the complexity.

## R5: Contract Enumeration — Manifest Pipeline Scan

**Decision**: Enumerate contracts by scanning all pipeline step definitions in the manifest via `pipeline.Pipeline` structs parsed from YAML. Deduplicate by schema path; inline contracts appear as entries named `pipeline:step`.

**Rationale**: Contracts are defined inline in pipeline step YAML (`handover.contract`), not in a top-level directory. The data source is the pipeline YAML files in the pipelines directory, already parsed by `DiscoverPipelines()` and `LoadPipelineByName()`. Reusing the same YAML parsing approach keeps the implementation consistent.

**Alternatives Rejected**:
- **Scanning `.wave/contracts/` directory**: Not all contracts have schema files; inline contracts would be missed.
- **Adding a top-level `contracts:` section to manifest**: Architectural change beyond this feature's scope.

## R6: Health Check Architecture — Fixed Checks with Async Execution

**Decision**: Define a fixed set of 6 health checks, each as a named function returning `(HealthStatus, string, map[string]string)`. Checks run asynchronously via `tea.Cmd` closures on first view access. Results arrive as individual `HealthCheckResultMsg` messages.

**Rationale**: Follows the established async data pattern from the header's metadata provider (`GitStateMsg`, `ManifestInfoMsg`, `GitHubInfoMsg`). Each check is independent and can complete at its own pace. The "checking..." placeholder state provides immediate visual feedback.

**Alternatives Rejected**:
- **Blocking checks**: Some checks (adapter binary lookup, database query) are fast, but `exec.Command` calls could block. Async is universally safe.
- **Single aggregated check**: Loses granularity — user can't see which specific check is slow or failing.

## R7: Quit Guard Encapsulation — IsFiltering() and CurrentFocus()

**Decision**: `ContentModel` exposes `IsFiltering() bool` and `CurrentFocus() FocusPane` methods. `AppModel`'s quit guard changes from `!m.content.list.filtering` to `!m.content.IsFiltering()`. `IsFiltering()` delegates to the active view's list model.

**Rationale**: Currently `AppModel` reaches into `m.content.list.filtering` and `m.content.focus` directly. With multiple views, the "list" field refers only to the pipeline list — its filtering state is irrelevant when other views are active. Encapsulation via methods lets `ContentModel` delegate to whichever view is currently active.

**Alternatives Rejected**:
- **Exposing all list models to AppModel**: Breaks encapsulation, creates tight coupling.
- **Disabling quit when non-pipeline views are active**: Too restrictive — `q` should quit from any view when appropriate.
