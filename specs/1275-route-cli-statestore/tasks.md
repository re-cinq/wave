# Work Items — Issue #1275

## Phase 1: StateStore API Surface

- [X] Item 1.1: Extend `internal/state/types.go` with `DecisionQueryOptions`, `EventAggregateStats`, and new fields on `EventQueryOptions` (`SinceUnix int64`, `TailLimit int`, `OrderDesc bool`).
- [X] Item 1.2: Add `GetMostRecentRunID`, `RunExists`, `GetRunStatus` to `StateStore` interface and `stateStore` impl.
- [X] Item 1.3: Extend `(s *stateStore) GetEvents` to honour `SinceUnix`, `TailLimit`, `OrderDesc` (preserve existing `AfterID` behaviour).
- [X] Item 1.4: Add `GetEventAggregateStats(runID)` method (the COUNT/SUM/AVG query currently in `renderPerformanceSummary`).
- [X] Item 1.5: Add `GetDecisionsFiltered(runID, DecisionQueryOptions)` method.
- [X] Item 1.6: Add `ListPipelineNamesByStatus(status)` method (covers both `pipeline_run` and `pipeline_state` fallback).
- [X] Item 1.7: Add `BackfillRunTokens() (int64, error)` method.

## Phase 2: Unit Tests for New StateStore Methods

- [X] Item 2.1: `internal/state/store_test.go` — happy path + empty DB tests for items 1.2–1.7. [P]
- [X] Item 2.2: Coverage for `EventQueryOptions{TailLimit, SinceUnix, OrderDesc}` reordering and limit semantics. [P]
- [X] Item 2.3: `BackfillRunTokens` test covers idempotency (re-running yields 0 affected rows) and only-finalized-runs guard. [P]

## Phase 3: Migrate CLI Commands

- [X] Item 3.1: Rewrite `cmd/wave/commands/logs.go` to use `state.NewReadOnlyStateStore` + new methods; delete local `*sql.DB` helpers (`getMostRecentRunID`, `runExists`, `getRunStatus`, `getLogID`, `queryLogs`, `queryNewLogs`, `renderPerformanceSummary`). [P]
- [X] Item 3.2: Rewrite `cmd/wave/commands/decisions.go` to use `GetDecisionsFiltered`; remove `queryDecisions` and `database/sql` import. [P]
- [X] Item 3.3: Rewrite `cmd/wave/commands/clean.go` `getWorkspacesWithStatus` to call `ListPipelineNamesByStatus`; drop `database/sql` import. [P]
- [X] Item 3.4: Rewrite `cmd/wave/commands/list.go` `collectRunsFromDB` to use `store.ListRuns(state.ListRunsOptions{...})`; drop `database/sql` import. [P]
- [X] Item 3.5: Replace `internal/webui/server.go` `backfillRunTokens` with call to `rwStore.BackfillRunTokens()`; drop `database/sql` and the local helper.

## Phase 4: Migrate Test Fixtures

- [X] Item 4.1: Rewrite `cmd/wave/commands/logs_test.go` `logsTestHelper` to seed via `state.NewStateStore` + `LogEvent` / `CreateRun`. [P]
- [X] Item 4.2: Rewrite `cmd/wave/commands/decisions_test.go` setup to seed via `RecordDecision`. [P]
- [X] Item 4.3: Rewrite `cmd/wave/commands/list_test.go` setup to seed via `CreateRun` + `UpdateRunStatus`. [P]
- [X] Item 4.4: Rewrite `cmd/wave/commands/status_test.go` setup to seed via `StateStore` writes. [P]

## Phase 5: Boundary Enforcement

- [X] Item 5.1: Add `depguard` rule `no-direct-sql` to `.golangci.yml` denying `database/sql` outside `internal/state/**`.
- [X] Item 5.2: Run `golangci-lint run ./...` and fix any residual hits (expected: zero after Phase 3–4).
- [X] Item 5.3: Verify with `grep -R 'database/sql' --include='*.go' .` that only `internal/state/**` matches.

## Phase 6: Polish & Docs

- [X] Item 6.1: Update ADR-007 status block: flip from "Proposed (not started)" to "Accepted (implemented)", clear the bypass list, note the depguard rule.
- [X] Item 6.2: Run full validation: `go vet ./...`, `go build ./...`, `go test -race ./...`, `golangci-lint run ./...`.
- [X] Item 6.3: Smoke test against a real `.agents/state.db`: `wave logs`, `wave logs --follow` (briefly), `wave decisions`, `wave list runs`, `wave clean --status failed --dry-run`. Confirm output parity with pre-refactor binary.
