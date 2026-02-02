# Implementation Tasks: Wave Ops Commands

**Feature**: 016-wave-ops-commands
**Created**: 2026-02-02
**Status**: Ready for Implementation

## Overview

This document breaks down the Wave Ops Commands feature into implementable tasks. The feature adds operational commands for pipeline monitoring, log inspection, cleanup, listing, cancellation, and artifact management.

---

## Task Index

| ID  | Title                                | Priority | Dependencies |
|-----|--------------------------------------|----------|--------------|
| T1  | Extend state schema for ops data     | P1       | -            |
| T2  | Add StateStore query methods         | P1       | T1           |
| T3  | Implement `wave status` command      | P1       | T2           |
| T4  | Write tests for `wave status`        | P1       | T3           |
| T5  | Implement `wave logs` command        | P1       | T2           |
| T6  | Add log streaming for `--follow`     | P1       | T5           |
| T7  | Write tests for `wave logs`          | P1       | T6           |
| T8  | Enhance `wave clean` command         | P1       | -            |
| T9  | Write tests for enhanced `clean`     | P1       | T8           |
| T10 | Enhance `wave list` for runs         | P2       | T2           |
| T11 | Write tests for enhanced `list`      | P2       | T10          |
| T12 | Implement `wave cancel` command      | P2       | T2           |
| T13 | Add graceful cancellation mechanism  | P2       | T12          |
| T14 | Write tests for `wave cancel`        | P2       | T13          |
| T15 | Implement `wave artifacts` command   | P2       | T2           |
| T16 | Add artifact export functionality    | P2       | T15          |
| T17 | Write tests for `wave artifacts`     | P2       | T16          |
| T18 | Update CLI help and documentation    | P2       | T4,T7,T9,T14,T17 |
| T19 | Integration testing                  | P1       | T18          |

---

## Detailed Tasks

### T1: Extend state schema for ops data

**Description**:
Extend the SQLite schema to capture additional operational data needed for status, logs, and artifacts commands. This includes tracking token usage, log output, current step for running pipelines, and artifact paths.

**Dependencies**: None

**Acceptance Criteria**:
- [ ] Schema includes `token_count` column in `step_state` table
- [ ] Schema includes `event_log` table for step events and logs
- [ ] Schema includes `artifacts` table linking steps to output files
- [ ] Schema includes `current_step` column in `pipeline_state` table
- [ ] Migration is backward-compatible (uses IF NOT EXISTS / ALTER IF NOT EXISTS pattern)
- [ ] All new columns have appropriate indexes for query performance

**Files to Modify/Create**:
- `internal/state/schema.sql` - Add new tables and columns
- `internal/state/migrations.go` - Create for schema migration logic (new file)

---

### T2: Add StateStore query methods

**Description**:
Add new methods to the StateStore interface and implementation to support the ops commands. These methods will query pipeline status, logs, artifacts, and running pipeline information.

**Dependencies**: T1

**Acceptance Criteria**:
- [ ] `GetRunningPipelines() ([]PipelineStateRecord, error)` method added
- [ ] `GetPipelineLogs(pipelineID string, stepFilter string) ([]LogRecord, error)` method added
- [ ] `SaveStepLog(pipelineID, stepID, content string) error` method added
- [ ] `GetPipelineArtifacts(pipelineID string) ([]ArtifactRecord, error)` method added
- [ ] `SaveArtifact(pipelineID, stepID, name, path, artifactType string) error` method added
- [ ] `UpdatePipelineCurrentStep(pipelineID, stepID string) error` method added
- [ ] `SetPipelineCancelled(pipelineID string) error` method added
- [ ] All methods have unit tests
- [ ] Methods handle concurrent access safely

**Files to Modify/Create**:
- `internal/state/store.go` - Add interface methods and implementation
- `internal/state/types.go` - Add LogRecord, ArtifactRecord types (new file)
- `internal/state/store_test.go` - Add tests for new methods

---

### T3: Implement `wave status` command

**Description**:
Create the `wave status` command that displays the current state of running and recent pipelines. Support showing a single pipeline's status by run ID, or listing all recent pipelines.

**Dependencies**: T2

**Acceptance Criteria**:
- [ ] `wave status` shows currently running pipeline(s) with columns: RUN_ID, PIPELINE, STATUS, STEP, ELAPSED, TOKENS
- [ ] `wave status --all` shows table of recent pipelines with status
- [ ] `wave status <run-id>` shows detailed status for specific run
- [ ] `--format json` outputs valid JSON for scripting
- [ ] Response time < 100ms for local state queries
- [ ] Handles case when no pipelines exist gracefully
- [ ] Color-coded status output (running=yellow, completed=green, failed=red)

**Files to Modify/Create**:
- `cmd/wave/commands/status.go` - New command implementation
- `cmd/wave/main.go` - Register new command

---

### T4: Write tests for `wave status`

**Description**:
Create comprehensive unit tests and integration tests for the status command.

**Dependencies**: T3

**Acceptance Criteria**:
- [ ] Test showing running pipeline status
- [ ] Test showing completed pipeline status
- [ ] Test showing failed pipeline status
- [ ] Test `--all` flag with multiple pipelines
- [ ] Test `--format json` output validity
- [ ] Test with specific run-id argument
- [ ] Test with no pipelines (empty state)
- [ ] Test concurrent status queries don't block execution

**Files to Modify/Create**:
- `cmd/wave/commands/status_test.go` - Unit tests

---

### T5: Implement `wave logs` command

**Description**:
Create the `wave logs` command that displays execution logs from pipeline runs. Support filtering by step, persona, and showing only errors.

**Dependencies**: T2

**Acceptance Criteria**:
- [ ] `wave logs` shows chronological output from all steps of most recent run
- [ ] `wave logs <run-id>` shows logs for specific run
- [ ] `--step <id>` filters to specific step
- [ ] `--errors` shows only error messages and failed validations
- [ ] `--tail <n>` shows last N lines
- [ ] `--since <duration>` filters by time (e.g., "10m", "1h")
- [ ] Logs include timestamp, step ID, persona, and content
- [ ] Handles missing logs gracefully with informative message

**Files to Modify/Create**:
- `cmd/wave/commands/logs.go` - New command implementation
- `cmd/wave/main.go` - Register new command

---

### T6: Add log streaming for `--follow`

**Description**:
Implement real-time log streaming for the `wave logs --follow` flag. This allows developers to watch logs as a pipeline executes.

**Dependencies**: T5

**Acceptance Criteria**:
- [ ] `wave logs --follow` streams output in real-time
- [ ] Streaming latency < 500ms
- [ ] Ctrl+C cleanly exits follow mode
- [ ] Follow mode shows new steps as they start
- [ ] Follow mode exits automatically when pipeline completes
- [ ] Works correctly when started mid-pipeline

**Files to Modify/Create**:
- `cmd/wave/commands/logs.go` - Add streaming logic
- `internal/state/store.go` - Add method to poll for new logs

---

### T7: Write tests for `wave logs`

**Description**:
Create comprehensive tests for the logs command including streaming.

**Dependencies**: T6

**Acceptance Criteria**:
- [ ] Test basic log retrieval
- [ ] Test `--step` filtering
- [ ] Test `--errors` filtering
- [ ] Test `--tail` truncation
- [ ] Test `--since` time filtering
- [ ] Test `--follow` streaming (with timeout)
- [ ] Test with empty logs
- [ ] Test with multi-step pipeline logs

**Files to Modify/Create**:
- `cmd/wave/commands/logs_test.go` - Unit tests

---

### T8: Enhance `wave clean` command

**Description**:
Enhance the existing `wave clean` command with additional capabilities for age-based cleanup and status-based filtering.

**Dependencies**: None

**Acceptance Criteria**:
- [ ] `--older-than <duration>` removes workspaces older than specified duration
- [ ] `--status <status>` only cleans workspaces for pipelines with given status (completed, failed)
- [ ] Interactive confirmation shows workspace count and total size
- [ ] `--force` skips confirmation
- [ ] Cleanup is atomic - no partial deletions on error
- [ ] Handles 1000+ workspaces efficiently (batch processing)
- [ ] Progress indicator for large cleanup operations

**Files to Modify/Create**:
- `cmd/wave/commands/clean.go` - Enhance existing command
- `internal/workspace/workspace.go` - Add workspace metadata queries

---

### T9: Write tests for enhanced `clean`

**Description**:
Add tests for the new clean command functionality.

**Dependencies**: T8

**Acceptance Criteria**:
- [ ] Test `--older-than` filtering
- [ ] Test `--status` filtering
- [ ] Test combined filters
- [ ] Test atomic cleanup (simulate error mid-deletion)
- [ ] Test with large number of workspaces
- [ ] Test confirmation prompt behavior
- [ ] Test `--dry-run` with new flags

**Files to Modify/Create**:
- `cmd/wave/commands/clean_test.go` - Add new tests

---

### T10: Enhance `wave list` for runs

**Description**:
Add a `wave list runs` subcommand to show recent pipeline executions with their status and metadata.

**Dependencies**: T2

**Acceptance Criteria**:
- [ ] `wave list runs` shows recent pipeline executions
- [ ] Each run shows: run-id, pipeline name, status, start time, duration
- [ ] `--limit <n>` controls number of runs shown (default 10)
- [ ] `--pipeline <name>` filters to specific pipeline
- [ ] `--status <status>` filters by status
- [ ] `--format json` outputs valid JSON
- [ ] Sorted by most recent first

**Files to Modify/Create**:
- `cmd/wave/commands/list.go` - Add runs subcommand

---

### T11: Write tests for enhanced `list`

**Description**:
Add tests for the new list runs functionality.

**Dependencies**: T10

**Acceptance Criteria**:
- [ ] Test `wave list runs` basic output
- [ ] Test `--limit` flag
- [ ] Test `--pipeline` filter
- [ ] Test `--status` filter
- [ ] Test `--format json` output
- [ ] Test with no runs

**Files to Modify/Create**:
- `cmd/wave/commands/list_test.go` - Add new tests

---

### T12: Implement `wave cancel` command

**Description**:
Create the `wave cancel` command that stops a running pipeline. By default, cancellation is graceful - the current step completes but no further steps start.

**Dependencies**: T2

**Acceptance Criteria**:
- [ ] `wave cancel` cancels the currently running pipeline
- [ ] `wave cancel <run-id>` cancels specific run
- [ ] Shows confirmation message with pipeline name
- [ ] Updates pipeline status to "cancelled" in state
- [ ] Returns error if no running pipeline found
- [ ] Returns error if specified run-id not found or not running

**Files to Modify/Create**:
- `cmd/wave/commands/cancel.go` - New command implementation
- `cmd/wave/main.go` - Register new command

---

### T13: Add graceful cancellation mechanism

**Description**:
Implement the cancellation signaling mechanism that allows the executor to check for cancellation between steps and optionally force-stop the current step.

**Dependencies**: T12

**Acceptance Criteria**:
- [ ] Executor checks for cancellation before starting each step
- [ ] `--force` flag interrupts current step immediately (sends SIGTERM to adapter)
- [ ] Graceful cancellation waits for current step to complete
- [ ] Cancellation status is persisted to state
- [ ] Pipeline can be resumed after graceful cancellation with `wave resume`
- [ ] Force cancellation marks pipeline as `cancelled` (same as graceful, but step may be incomplete)

**Files to Modify/Create**:
- `internal/pipeline/executor.go` - Add cancellation check
- `internal/state/store.go` - Add cancellation flag query
- `internal/adapter/runner.go` - Add force termination support

---

### T14: Write tests for `wave cancel`

**Description**:
Create tests for the cancel command and cancellation mechanism.

**Dependencies**: T13

**Acceptance Criteria**:
- [ ] Test graceful cancellation (step completes)
- [ ] Test force cancellation (step interrupted)
- [ ] Test cancellation with no running pipeline
- [ ] Test cancellation with invalid run-id
- [ ] Test cancellation state persistence
- [ ] Test resume after graceful cancellation
- [ ] Test concurrent cancellation requests

**Files to Modify/Create**:
- `cmd/wave/commands/cancel_test.go` - Unit tests
- `internal/pipeline/executor_test.go` - Add cancellation tests

---

### T15: Implement `wave artifacts` command

**Description**:
Create the `wave artifacts` command that lists artifacts produced by pipeline steps. Artifacts include any files written to the output paths defined in the pipeline YAML.

**Dependencies**: T2

**Acceptance Criteria**:
- [ ] `wave artifacts` lists artifacts from most recent run
- [ ] `wave artifacts <run-id>` lists artifacts from specific run
- [ ] Each artifact shows: step, name, path, size, type
- [ ] `--step <id>` filters to specific step
- [ ] `--format json` outputs valid JSON
- [ ] Handles missing artifacts gracefully
- [ ] Verifies artifact files still exist on disk

**Files to Modify/Create**:
- `cmd/wave/commands/artifacts.go` - New command implementation
- `cmd/wave/main.go` - Register new command

---

### T16: Add artifact export functionality

**Description**:
Add the `--export` flag to the artifacts command that copies artifacts to a specified directory.

**Dependencies**: T15

**Acceptance Criteria**:
- [ ] `--export <dir>` copies all artifacts to specified directory
- [ ] Creates export directory if it doesn't exist
- [ ] Preserves artifact names in export
- [ ] Handles name collisions (prefix with step ID)
- [ ] Shows summary of exported files
- [ ] `--step` filter works with export
- [ ] Fails gracefully if source artifact missing

**Files to Modify/Create**:
- `cmd/wave/commands/artifacts.go` - Add export logic

---

### T17: Write tests for `wave artifacts`

**Description**:
Create tests for the artifacts command including export functionality.

**Dependencies**: T16

**Acceptance Criteria**:
- [ ] Test listing artifacts
- [ ] Test `--step` filter
- [ ] Test `--format json` output
- [ ] Test `--export` to new directory
- [ ] Test `--export` with existing directory
- [ ] Test handling missing artifacts
- [ ] Test handling name collisions in export
- [ ] Test with no artifacts

**Files to Modify/Create**:
- `cmd/wave/commands/artifacts_test.go` - Unit tests

---

### T18: Update CLI help and documentation

**Description**:
Update all command help text to be comprehensive and include examples. Ensure consistent formatting across all commands.

**Dependencies**: T4, T7, T9, T14, T17

**Acceptance Criteria**:
- [ ] All new commands have `--help` with examples
- [ ] Examples show common use cases
- [ ] Error messages include actionable suggestions
- [ ] Help text follows existing command patterns
- [ ] Update CLAUDE.md CLI Commands section with new commands

**Files to Modify/Create**:
- `cmd/wave/commands/status.go` - Enhance help text
- `cmd/wave/commands/logs.go` - Enhance help text
- `cmd/wave/commands/cancel.go` - Enhance help text
- `cmd/wave/commands/artifacts.go` - Enhance help text
- `cmd/wave/commands/clean.go` - Update help for new flags
- `cmd/wave/commands/list.go` - Update help for runs subcommand
- `CLAUDE.md` - Update CLI Commands section

---

### T19: Integration testing

**Description**:
Create end-to-end integration tests that verify all ops commands work correctly with real pipeline executions.

**Dependencies**: T18

**Acceptance Criteria**:
- [ ] Integration test: run pipeline, check status during execution, verify logs after
- [ ] Integration test: run pipeline, cancel mid-execution, verify state
- [ ] Integration test: run pipeline, list artifacts, export to directory
- [ ] Integration test: run multiple pipelines, clean with --keep-last
- [ ] Integration test: run pipeline, list runs, filter by status
- [ ] All integration tests pass with race detector
- [ ] Tests run in CI/CD pipeline

**Files to Modify/Create**:
- `internal/pipeline/integration_test.go` - Add ops command integration tests (new file)

---

## Implementation Order

The recommended implementation order follows dependencies and priority:

### Phase 1: Foundation (P1)
1. T1 - State schema extensions
2. T2 - StateStore query methods

### Phase 2: Core Commands (P1)
3. T3 - Status command
4. T4 - Status tests
5. T5 - Logs command
6. T6 - Log streaming
7. T7 - Logs tests
8. T8 - Clean enhancements
9. T9 - Clean tests

### Phase 3: Additional Commands (P2)
10. T10 - List runs
11. T11 - List tests
12. T12 - Cancel command
13. T13 - Cancellation mechanism
14. T14 - Cancel tests
15. T15 - Artifacts command
16. T16 - Artifact export
17. T17 - Artifacts tests

### Phase 4: Polish (P2)
18. T18 - Documentation
19. T19 - Integration tests

---

## Estimated Effort

| Phase | Tasks | Estimated Hours |
|-------|-------|-----------------|
| Foundation | T1-T2 | 4-6 |
| Core Commands | T3-T9 | 12-16 |
| Additional Commands | T10-T17 | 16-20 |
| Polish | T18-T19 | 6-8 |
| **Total** | **19 tasks** | **38-50 hours** |

---

## Resolved Decisions

> See `clarifications.md` for full context on each decision.

1. **Log levels**: `--level all|info|error` with `--errors` shorthand (Resolved: C1)
2. **Automatic cleanup**: No daemon - manual `wave clean` only with `--force` for CI (Resolved: C3)
3. **Cancellation signal**: Database flag for graceful, SIGTERM/SIGKILL for force (Resolved: C2)
4. **Log storage**: SQLite `event_log` table for consistency with state (Resolved: C4)
