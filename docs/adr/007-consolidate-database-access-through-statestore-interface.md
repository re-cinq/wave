# ADR-007: Consolidate Database Access Through StateStore Interface

## Status
Proposed

## Date
2026-04-13

## Context

Wave uses SQLite (via the `modernc.org/sqlite` pure-Go driver) for pipeline state persistence. The `internal/state/` package defines a `StateStore` interface with over 50 methods spanning 13 concerns: run tracking, event logging, artifact tracking, cancellation, performance metrics, progress tracking, tags, step attempts, chat sessions, ontology, checkpoints, retrospectives, decisions, webhooks, outcomes, and orchestration decisions. The concrete implementation wraps a `*sql.DB` handle and is the intended single point of database access.

Most of the codebase correctly uses the `StateStore` interface. The `internal/pipeline/` package receives it via constructor injection (`WithStateStore` option). The `internal/tui/` package accepts it as a dependency. The `internal/webui/` package uses separate read-only (`s.store`) and read-write (`s.rwStore`) `StateStore` instances. Over a dozen CLI commands in `cmd/wave/commands/` open `state.NewStateStore()` and call interface methods. However, four CLI commands and one webui function bypass the `StateStore` interface entirely, opening their own `sql.Open("sqlite", dbPath)` connections and executing hand-crafted SQL queries directly against the database tables:

- **`cmd/wave/commands/logs.go`** contains 7 functions that take `*sql.DB`, with custom SQL for event filtering, tail mode, follow-mode polling (500ms interval with incremental ID tracking), and performance aggregation (`COUNT`/`SUM`/`AVG` queries).
- **`cmd/wave/commands/list.go`** has a `collectRunsFromDB()` function that duplicates the query logic already available through `StateStore.ListRuns(ListRunsOptions)`.
- **`cmd/wave/commands/decisions.go`** has a `queryDecisions()` function with category-based filtering not exposed by the existing `GetDecisions()` or `GetDecisionsByStep()` methods.
- **`cmd/wave/commands/clean.go`** has a `getWorkspacesWithStatus()` function that queries pipeline names filtered by status.
- **`internal/webui/server.go`** contains a `backfillRunTokens()` migration function that executes raw SQL to backfill token data.

These violations break the layered architecture defined in [ADR-003](003-layered-architecture.md), where the Presentation layer (`cmd/`) should not directly depend on `database/sql`. They duplicate schema knowledge -- table names, column names, and query patterns -- in five locations outside the state package. The existing `depguard` rules in `.golangci.yml` enforce layer boundaries within `internal/` but do not cover `cmd/wave/commands/` imports of `database/sql`, so no CI check catches these violations today.

This consolidation is also time-sensitive. [ADR-006](006-cost-infrastructure.md) plans to add columns (`tokens_input`, `tokens_output`, `estimated_cost_dollars`) to the `pipeline_run`, `step_attempt`, and `performance_metric` tables. If the four violating commands continue to use hand-crafted SQL, each will require manual updates for every schema change. Consolidation before ADR-006 prevents that double-work. Additionally, [ADR-002](002-extract-step-executor.md) plans to extract `StepExecutor` from the monolithic `executor.go`, with `StateStore` as a constructor-injected dependency -- establishing the pattern that all components receive `StateStore` via dependency injection rather than opening their own connections. The project is in prototype phase with no backward compatibility constraints, making interface changes low-cost.

## Decision

Consolidate all direct database access into the `state` package by adding approximately 8-10 new methods to the existing `StateStore` interface, and enforce the boundary with a `depguard` rule that prevents `database/sql` imports outside `internal/state/`.

This approach was chosen because the existing `StateStore` infrastructure already handles the patterns these commands need. The `EventQueryOptions` struct already has an `AfterID` field (used for SSE backfill in webui), which directly supports the incremental polling pattern that `logs.go` requires -- extending it with `SinceTime` and `TailLimit` fields is a natural evolution. The `ListRuns(ListRunsOptions)` method already accepts `PipelineName`, `Status`, and `Limit` fields that match the hand-crafted SQL in `list.go` almost exactly, making that migration near-trivial. For the remaining gaps (`decisions.go` category filtering, `clean.go` status-based pipeline queries, `logs.go` performance aggregation), new methods with option structs follow the established pattern in the codebase.

The alternative of decomposing the `StateStore` interface into role-based sub-interfaces (interface segregation) is the right long-term direction but should be deferred until [ADR-002](002-extract-step-executor.md) lands, when actual consumer patterns are concrete rather than speculative. Designing sub-interfaces around a planned but unimplemented decomposition risks choosing wrong boundaries that need re-drawing. A CLI query service layer was also considered but rejected because it cannot avoid adding `StateStore` methods for aggregation queries, meaning it adds an architectural layer without eliminating the core work.

## Options Considered

### Option 1: Status Quo -- Keep Direct SQL in CLI Commands

Leave the four CLI commands and `backfillRunTokens()` with their existing `sql.Open()` and hand-crafted SQL. No changes to the `StateStore` interface or `depguard` rules.

**Pros:**
- Zero implementation effort and zero risk of introducing regressions
- CLI commands retain full control over query optimization (e.g., `logs.go` follow mode is already tuned for SQLite performance with 500ms polling and incremental ID tracking)
- Avoids growing the already large StateStore interface beyond its current 50+ methods

**Cons:**
- ADR-006 schema changes to `pipeline_run`, `step_attempt`, and `performance_metric` tables will require manual SQL updates in each violating command, duplicating migration work across five files
- Violates ADR-003 layered architecture: Presentation layer directly depends on `database/sql`, and schema knowledge (table names, column names, query patterns) is duplicated in five locations outside the state package
- `list.go`'s `collectRunsFromDB()` duplicates logic already available through `StateStore.ListRuns(ListRunsOptions)`, querying the same `pipeline_run` table with nearly identical filtering
- Test files (`logs_test.go`, `list_test.go`, `decisions_test.go`) maintain parallel `sql.Open` paths for setup, creating a maintenance burden on every schema change
- No static enforcement prevents future commands from copying the direct-SQL pattern, since `depguard` rules cover `internal/` but not `cmd/` `database/sql` imports

### Option 2: Consolidate into Existing StateStore Interface (Recommended)

Move all direct SQL from the four CLI commands and `backfillRunTokens()` into the `state` package by adding approximately 8-10 new methods. Extend `EventQueryOptions` with `SinceTime` and `TailLimit` fields. Add `GetMostRecentRunID()`, `GetRunStatus()`, and `ListPipelineNamesByStatus()` helpers. Extend `GetDecisions()` to accept a `DecisionQueryOptions` struct with `StepID` and `Category` filters. Move `backfillRunTokens()` to `migration_runner.go`. Update the four CLI commands to use `state.NewStateStore()` or `state.NewReadOnlyStateStore()`. Add a `depguard` rule preventing `database/sql` imports outside `internal/state/`.

**Pros:**
- Fully enforces ADR-003 layered architecture: all schema knowledge is consolidated in the state package, and the Presentation layer depends only on the `StateStore` interface
- `EventQueryOptions` already has an `AfterID` field used for SSE backfill in webui, directly supporting the `logs.go` follow-mode pattern with minimal new abstraction
- `list.go` migration is near-trivial since `ListRuns(ListRunsOptions)` already exists with matching filter fields
- Insulates CLI commands from ADR-006 schema changes: when cost columns are added, only the state package implementation needs updating
- A `depguard` rule prevents future violations by failing CI on any new `cmd/` file importing `database/sql`
- Prototype phase means no backward compatibility constraint: interface changes carry no cost, and there are no external consumers to break

**Cons:**
- StateStore interface grows from approximately 50 to 58 methods, increasing the surface that test mocks must implement
- `logs.go` is the most complex migration: 7 functions with conditional query building (tail mode, follow mode, performance aggregation) require careful extraction to preserve 500ms polling behavior and incremental ID tracking
- Test infrastructure in `cmd/wave/commands/` uses direct `sql.Open` for inserting test fixtures with specific IDs, requiring new StateStore test helpers or a shared test factory (approximately 100 lines of test support code)
- Adding methods to a monolithic interface makes it harder to reason about which consumers need which capabilities

### Option 3: Consolidate with Interface Segregation

Same consolidation as Option 2, plus decompose the `StateStore` interface into role-based sub-interfaces: `RunReader`, `EventReader`, `DecisionReader`, `RunWriter`, and a composed `StateStore` that embeds all sub-interfaces. CLI read-only commands accept narrow reader interfaces. The full `StateStore` is used only by the pipeline executor and write paths.

**Pros:**
- All benefits of Option 2 (ADR-003 compliance, ADR-006 protection, `depguard` enforcement)
- Aligns with ADR-002 StepExecutor extraction: the planned `StepExecutor` can accept narrow interfaces (`RunWriter` + `EventWriter`) instead of the full `StateStore`
- CLI commands declare their actual dependencies (`logs.go` takes `EventReader`, `list.go` takes `RunReader`), and test mocks need only 2-5 methods instead of 58
- Read-only vs read-write distinction becomes type-safe at compile time, replacing the runtime `PRAGMA query_only=ON` defense
- Follows Go convention of small, focused interfaces

**Cons:**
- Significantly more design work upfront: choosing correct interface boundaries requires analyzing all 20+ consumers of `StateStore` to determine which methods each actually calls
- Every existing consumer that currently takes `StateStore` must be updated to accept the appropriate sub-interface, touching `pipeline/`, `tui/`, `webui/`, and all `cmd/` commands (potentially 30+ call sites)
- Risk of wrong decomposition: if boundaries do not match real usage patterns, consumers end up needing 3-4 sub-interfaces composed together, which is worse than one interface
- Increases the number of types in the state package (5-8 new interface types plus the composed type), adding cognitive overhead
- ADR-002 StepExecutor extraction has not landed yet, so designing sub-interfaces around a planned but unimplemented decomposition is speculative

### Option 4: CLI Query Service Layer

Introduce a `cmd/wave/internal/query` package that wraps `StateStore` with CLI-specific query logic. Functions like `QueryLogs(store state.StateStore, opts LogQueryOptions)` compose existing `StateStore` methods and add CLI-specific concerns. CLI commands replace `sql.Open` with `state.NewReadOnlyStateStore()` and call query functions. `StateStore` interface stays unchanged.

**Pros:**
- StateStore interface stays at 50 methods with no interface bloat; CLI-specific query composition lives in a dedicated layer
- CLI query logic (tail mode, follow polling, aggregation) stays close to the commands that use it, rather than being pushed into the infrastructure layer
- Query functions are pure logic over the StateStore interface, making them straightforward to test with mocks
- Naturally aligns with the read-only access pattern

**Cons:**
- Requires existing `StateStore` methods to expose enough data for CLI queries; if `GetEvents()` lacks `SinceTime` filtering, the query layer must fetch all events and filter in Go, which is inefficient for large `event_log` tables
- Creates a new package that sits between Presentation and Infrastructure, introducing a layer not defined in ADR-003's architecture model
- Performance aggregation (`COUNT`/`SUM`/`AVG`) has no `StateStore` equivalent and would require either N+1 queries or a new `StateStore` method, defeating the purpose of the separate layer
- Partial solution: some SQL (aggregation queries, filtered decisions by category) cannot be composed from existing methods without adding methods to `StateStore` anyway
- `backfillRunTokens` still needs to move to the state package regardless, as the query layer does not address write-side violations

## Consequences

### Positive
- Schema knowledge is consolidated in a single package (`internal/state/`), eliminating five duplicate locations where table names, column names, and query patterns are hard-coded. When ADR-006 adds `tokens_input`, `tokens_output`, and `estimated_cost_dollars` columns, only the state package SQL needs updating -- zero changes in CLI commands.
- The `depguard` rule provides permanent CI enforcement: any future `cmd/` file importing `database/sql` fails the build, preventing regression to the direct-SQL pattern.
- `list.go` drops its entire `collectRunsFromDB()` function in favor of the existing `ListRuns(ListRunsOptions)` call, removing approximately 40 lines of duplicated query logic.
- Read-only CLI commands (`logs`, `list`, `decisions`) switch from uncontrolled `sql.Open` connections to `state.NewReadOnlyStateStore()`, which sets `PRAGMA query_only=ON` and uses a higher `MaxOpenConns` tuned for concurrent read access.
- Test infrastructure becomes more maintainable: CLI test files use `StateStore` test helpers instead of maintaining parallel `sql.Open` paths that break on schema changes.

### Negative
- The `StateStore` interface grows from approximately 50 to 58 methods. Until interface segregation is implemented (likely alongside ADR-002), test mocks must stub the full interface even when a consumer uses only 2-3 methods. This is a tolerable cost during prototype phase but should not be deferred indefinitely.
- Migrating `logs.go` is the highest-risk change in this consolidation. Its 7 functions with conditional query building (tail mode with `LIMIT` + `ORDER BY id DESC`, follow mode with `WHERE id > ?` polling, and `COUNT`/`SUM`/`AVG` aggregation) must be carefully extracted into `StateStore` methods that preserve the current 500ms polling behavior and incremental ID tracking. A regression here would degrade the `wave logs --follow` user experience.
- Approximately 100 lines of test support code (StateStore test factories or fixture helpers) must be written to replace direct `sql.Open` test setup in `logs_test.go`, `list_test.go`, `decisions_test.go`, and `status_test.go`.

### Neutral
- `backfillRunTokens()` moves from `internal/webui/server.go` to `internal/state/migration_runner.go`. This is a file relocation, not a behavioral change -- the migration logic and SQL remain identical.
- The `depguard` configuration in `.golangci.yml` gains one new rule (`no-cmd-database-sql`). This extends the existing enforcement mechanism established by ADR-003 rather than introducing new tooling.
- Existing consumers that already use `StateStore` correctly (`internal/pipeline/`, `internal/tui/`, most `cmd/wave/commands/`) are unaffected.

## Implementation Notes

1. **Extend `EventQueryOptions` in `internal/state/store.go`**: Add `SinceTime time.Time`, `TailLimit int`, and `Level string` fields to the existing struct. Implement the corresponding `WHERE` clause generation in the SQLite query builder. This covers `logs.go` filtering, tail mode, and level-based filtering.

2. **Add aggregation methods to `StateStore`**: Add `GetEventAggregation(runID string) (EventAggregation, error)` and `GetPerformanceSummary(runID string) (PerformanceSummary, error)` to support `logs.go` performance reporting. Return typed structs rather than raw `*sql.Rows`.

3. **Add query option structs for decisions and clean**: Add `DecisionQueryOptions` with `StepID`, `Category`, and `Limit` fields. Add `ListPipelineNamesByStatus(status string) ([]string, error)` for `clean.go`. Add `GetMostRecentRunID(pipelineName string) (string, error)` and `GetRunStatus(runID string) (string, error)` for shared use by `logs.go` and `decisions.go`.

4. **Migrate CLI commands in order of increasing complexity**:
   - `cmd/wave/commands/list.go` -- Replace `collectRunsFromDB()` with `StateStore.ListRuns()`. Near-trivial, validates the approach.
   - `cmd/wave/commands/clean.go` -- Replace `getWorkspacesWithStatus()` with `StateStore.ListPipelineNamesByStatus()`. Small and contained.
   - `cmd/wave/commands/decisions.go` -- Replace `queryDecisions()` with `StateStore.GetDecisions(DecisionQueryOptions{})`. Moderate: requires the new option struct.
   - `cmd/wave/commands/logs.go` -- Replace all 7 `*sql.DB` functions with `StateStore` method calls. Most complex: requires all new methods from steps 1-3. Test the `--follow` mode polling behavior explicitly.

5. **Move `backfillRunTokens()`**: Relocate from `internal/webui/server.go` to `internal/state/migration_runner.go`. Update `internal/webui/server.go` to call the state package function instead.

6. **Update test infrastructure**: Create a `state/testutil_test.go` (or equivalent) with helper functions for inserting test fixtures via `StateStore` methods, replacing direct `sql.Open` in `cmd/wave/commands/logs_test.go`, `list_test.go`, `decisions_test.go`, and `status_test.go`.

7. **Add `depguard` rule to `.golangci.yml`**:
   ```yaml
   no-cmd-database-sql:
     deny:
       - pkg: "database/sql"
         desc: "CLI commands must use state.StateStore, not direct database/sql"
     files:
       - "cmd/wave/commands/**"
   ```

8. **Verify CI passes**: Run `golangci-lint` with the new rule to confirm no remaining `database/sql` imports in `cmd/wave/commands/`. Run the full test suite to confirm no regressions in `logs`, `list`, `decisions`, and `clean` command behavior.
