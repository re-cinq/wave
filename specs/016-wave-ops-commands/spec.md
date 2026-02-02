# Feature Specification: Wave CLI Operational Commands

**Feature Branch**: `016-wave-ops-commands`
**Created**: 2026-02-02
**Status**: Draft
**Input**: User description: "Wave CLI Operational Commands - Add ps/status, logs, health, and detached execution mode. Users need to monitor running pipelines, view execution logs, check system health, and run pipelines in background with ability to attach later."

## User Scenarios & Testing _(mandatory)_

### User Story 1 - View Pipeline Status (Priority: P1)

As a developer running Wave pipelines, I want to see the status of all recent and running pipelines so I can monitor progress and identify issues quickly.

**Why this priority**: Core visibility into pipeline execution is essential for any operational workflow. Without status visibility, users are blind to what's happening in their system.

**Independent Test**: Run `wave ps` and verify it displays a table of pipelines with their IDs, names, status, and timing information.

**Acceptance Scenarios**:

1. **Given** pipelines have been executed, **When** I run `wave ps`, **Then** I see a table showing pipeline ID, name, status (running/completed/failed), start time, and duration
2. **Given** a pipeline is currently running, **When** I run `wave ps`, **Then** the running pipeline appears with status "running" and elapsed time
3. **Given** no pipelines have been executed, **When** I run `wave ps`, **Then** I see a message "No pipelines found"
4. **Given** multiple pipelines exist, **When** I run `wave ps --limit 5`, **Then** I see only the 5 most recent pipelines

---

### User Story 2 - View Pipeline Logs (Priority: P1)

As a developer debugging a pipeline, I want to view the execution logs for a specific pipeline or step so I can understand what happened during execution.

**Why this priority**: Logs are critical for debugging failed pipelines and understanding execution flow. This is a core operational need.

**Independent Test**: Run `wave logs <pipeline-id>` and verify it displays the execution trace with timestamps, step names, and tool calls.

**Acceptance Scenarios**:

1. **Given** a pipeline has been executed, **When** I run `wave logs <pipeline-id>`, **Then** I see the full execution log with timestamps and step progression
2. **Given** a pipeline is running, **When** I run `wave logs --follow <pipeline-id>`, **Then** I see live log output as the pipeline executes
3. **Given** a pipeline with multiple steps, **When** I run `wave logs <pipeline-id> --step <step-id>`, **Then** I see only logs for that specific step
4. **Given** logs contain sensitive data, **When** I view logs, **Then** credentials and secrets are scrubbed from the output
5. **Given** an invalid pipeline ID, **When** I run `wave logs <invalid-id>`, **Then** I see an error "Pipeline not found: <invalid-id>"

---

### User Story 3 - Detached Pipeline Execution (Priority: P2)

As a developer running long pipelines, I want to run pipelines in the background so I can continue working while the pipeline executes.

**Why this priority**: Long-running pipelines block the terminal. Detached mode enables productivity during execution, but requires status/logs features first.

**Independent Test**: Run `wave run --detach --pipeline hotfix`, verify it returns immediately with a pipeline ID, then use `wave ps` and `wave logs` to monitor.

**Acceptance Scenarios**:

1. **Given** a valid pipeline, **When** I run `wave run --detach --pipeline <name>`, **Then** the command returns immediately with a pipeline ID and message "Pipeline <id> started in background"
2. **Given** a detached pipeline is running, **When** I run `wave attach <pipeline-id>`, **Then** I see live output from the pipeline as if running in foreground
3. **Given** a detached pipeline completes, **When** I run `wave attach <pipeline-id>`, **Then** I see the final output and exit status
4. **Given** I am attached to a pipeline, **When** I press Ctrl+C, **Then** I detach without stopping the pipeline (with message "Detached. Pipeline continues running.")
5. **Given** a detached pipeline, **When** I run `wave stop <pipeline-id>`, **Then** the pipeline is gracefully terminated

---

### User Story 4 - System Health Check (Priority: P3)

As a developer setting up Wave or troubleshooting issues, I want to check the health of my Wave installation so I can identify configuration problems quickly.

**Why this priority**: Health checks are useful for setup/troubleshooting but not needed for day-to-day operations.

**Independent Test**: Run `wave health` and verify it checks adapters, database, and workspace configuration.

**Acceptance Scenarios**:

1. **Given** a properly configured Wave project, **When** I run `wave health`, **Then** I see a checklist of all health checks with pass/fail status
2. **Given** an adapter binary is missing, **When** I run `wave health`, **Then** the adapter check fails with message "Adapter '<name>' not found: binary '<path>' not in PATH"
3. **Given** the state database is corrupted, **When** I run `wave health`, **Then** the database check fails with a specific error
4. **Given** the workspace directory is not writable, **When** I run `wave health`, **Then** the workspace check fails with permission error
5. **Given** all checks pass, **When** I run `wave health`, **Then** I see "All health checks passed" with exit code 0

---

### Edge Cases

- What happens when trying to attach to a completed pipeline? (Show final output and exit)
- What happens when stopping an already-stopped pipeline? (Idempotent, show "Pipeline already stopped")
- What happens when viewing logs for a pipeline with no trace file? (Show error with explanation)
- How does system handle multiple simultaneous detached pipelines? (Each gets unique ID, all tracked independently)
- What happens to detached pipelines when the terminal session ends? (Continue running, recoverable via `wave ps`)
- What happens when disk space runs out during detached execution? (Pipeline fails, state preserved for debugging)

## Requirements _(mandatory)_

### Functional Requirements

**Status Commands**:
- **FR-001**: System MUST provide a `wave ps` command that lists recent pipelines with their execution status
- **FR-002**: System MUST display pipeline ID, name, status, start time, and duration in `wave ps` output
- **FR-003**: System MUST support `--limit N` flag to control number of pipelines shown (default: 10)
- **FR-004**: System MUST support `--all` flag to show all pipelines regardless of status
- **FR-005**: System MUST support `--running` flag to filter to only running pipelines

**Log Commands**:
- **FR-006**: System MUST provide a `wave logs <pipeline-id>` command to view execution logs
- **FR-007**: System MUST support `--follow` / `-f` flag for live log streaming
- **FR-008**: System MUST support `--step <step-id>` flag to filter logs to a specific step
- **FR-009**: System MUST support `--tail N` flag to show only last N lines (default: all)
- **FR-010**: System MUST scrub credentials and secrets from log output using existing audit patterns

**Detached Execution**:
- **FR-011**: System MUST support `--detach` / `-d` flag on `wave run` command
- **FR-012**: System MUST return immediately with pipeline ID when running detached
- **FR-013**: System MUST provide `wave attach <pipeline-id>` command to connect to running pipeline output
- **FR-014**: System MUST allow detaching from attached pipeline with Ctrl+C without stopping execution
- **FR-015**: System MUST provide `wave stop <pipeline-id>` command to gracefully terminate pipelines
- **FR-016**: System MUST persist detached pipeline output to a log file for later viewing

**Health Commands**:
- **FR-017**: System MUST provide a `wave health` command that checks system configuration
- **FR-018**: System MUST check adapter binary availability and report missing adapters
- **FR-019**: System MUST check state database accessibility and integrity
- **FR-020**: System MUST check workspace directory permissions
- **FR-021**: System MUST check manifest file validity if present
- **FR-022**: System MUST return exit code 0 if all checks pass, non-zero otherwise

### Key Entities

- **PipelineExecution**: Represents a single execution of a pipeline with ID, status, start/end times, and log file path
- **LogEntry**: A single log line with timestamp, pipeline ID, step ID, level, and message
- **HealthCheck**: A single health check with name, status (pass/fail), and message

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: Users can view status of all recent pipelines within 1 second using `wave ps`
- **SC-002**: Users can view live logs of a running pipeline with less than 500ms latency
- **SC-003**: Detached pipelines continue running when terminal session ends
- **SC-004**: Users can diagnose configuration issues within 30 seconds using `wave health`
- **SC-005**: All commands provide clear, actionable error messages when operations fail
- **SC-006**: Log viewing does not expose credentials or sensitive data

## Assumptions

- Detached pipelines write output to `.wave/logs/<pipeline-id>.log` for persistence
- The state database already tracks pipeline execution state (confirmed from existing implementation)
- Trace logs already exist in `.wave/traces/` from the audit logger
- Process management will use standard OS signals (SIGTERM for graceful stop)
- Maximum of 100 concurrent detached pipelines (reasonable for CLI usage)
