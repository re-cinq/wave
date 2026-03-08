# Feature Specification: Detach Pipeline Execution from TUI Process Lifecycle

**Feature Branch**: `284-tui-detach-execution`
**Created**: 2026-03-08
**Status**: Draft
**Input**: GitHub Issue #284 — feat(tui): detach pipeline execution from TUI process lifecycle

## User Scenarios & Testing _(mandatory)_

### User Story 1 - Pipeline Survives TUI Exit (Priority: P1)

A user launches a long-running pipeline from the TUI, realizes they need to close the terminal, and quits the TUI with `q` or `ctrl+c`. The pipeline continues running in the background. When the user reopens the TUI later, they see the pipeline is still running with all events persisted.

**Why this priority**: This is the core problem. Currently, quitting the TUI kills all in-flight pipelines because the executor runs as an in-process goroutine. Users cannot safely close the TUI during multi-hour pipeline runs.

**Independent Test**: Can be fully tested by launching a pipeline from TUI, quitting TUI, then verifying the pipeline process is still alive via `ps` and that `wave logs <run-id>` streams events from it.

**Acceptance Scenarios**:

1. **Given** a pipeline is running via the TUI, **When** the user presses `q` or `ctrl+c`, **Then** the TUI exits but the pipeline subprocess continues running to completion
2. **Given** a pipeline was launched from the TUI and the TUI was closed, **When** the user reopens the TUI, **Then** the running pipeline appears in the Running section with its current status
3. **Given** a detached pipeline is running, **When** the user runs `wave logs <run-id>`, **Then** events stream from the still-running pipeline via SQLite event persistence

---

### User Story 2 - Cancel Detached Pipeline from TUI (Priority: P2)

A user reopens the TUI and sees a previously-launched pipeline still running. They decide to cancel it. Pressing the cancel keybinding sends a cancellation signal through the persistent store, and the detached subprocess receives and honors it.

**Why this priority**: Without cancellation support, detached pipelines become uncontrollable fire-and-forget processes. Users need the ability to stop pipelines they started.

**Independent Test**: Can be tested by launching a pipeline, quitting TUI, reopening TUI, pressing cancel on the running pipeline, and verifying the subprocess terminates gracefully.

**Acceptance Scenarios**:

1. **Given** a detached pipeline is running and the user reopens the TUI, **When** the user presses the cancel key on the pipeline, **Then** a cancellation is persisted via `store.RequestCancellation()` and the detached subprocess terminates within a reasonable grace period
2. **Given** a cancellation has been requested, **When** the detached process checks for cancellation, **Then** it performs graceful shutdown including workspace cleanup and final state persistence

---

### User Story 3 - TUI-CLI Execution Parity (Priority: P2)

Both `wave run <pipeline>` (CLI) and TUI-launched pipelines use the same subprocess-based execution path. This eliminates behavioral divergence between the two entry points and ensures that pipelines behave identically regardless of how they were started.

**Why this priority**: Parity reduces maintenance burden and user confusion. The same execution path means the same failure modes, logging behavior, and cancellation semantics.

**Independent Test**: Can be tested by running the same pipeline via CLI and TUI, comparing event logs, exit codes, and artifact outputs for equivalence.

**Acceptance Scenarios**:

1. **Given** the same pipeline and input, **When** launched from the CLI, **Then** the execution path is identical to launching from the TUI (same subprocess, same state transitions, same event persistence)
2. **Given** a pipeline running from the TUI, **When** inspected via `wave status` or the WebUI, **Then** it appears identical to a CLI-launched pipeline

---

### User Story 4 - Reconnect to Running Pipeline Events (Priority: P3)

When the TUI is reopened, it reconnects to in-progress pipelines by tailing persisted events from SQLite. The user sees the pipeline's current step, recent log lines, and can watch new events appear in real-time.

**Why this priority**: This is the UX polish that makes detached execution feel seamless. Without it, users see a "running" label but no detail until the pipeline completes.

**Independent Test**: Can be tested by launching a pipeline, quitting TUI, waiting for the pipeline to advance several steps, reopening TUI, and verifying that historical events are loaded and new events appear live.

**Acceptance Scenarios**:

1. **Given** a detached pipeline has progressed through several steps, **When** the user reopens the TUI and selects the pipeline, **Then** historical events from SQLite are loaded into the detail view
2. **Given** the TUI is connected to a running detached pipeline, **When** new events are emitted by the subprocess, **Then** they appear in the TUI within the existing refresh interval

---

### Edge Cases

- What happens when the TUI is closed during subprocess spawn (race between fork and TUI exit)? The subprocess must be fully detached before TUI shutdown proceeds, or the run must be marked as failed if the spawn did not complete.
- How does the system handle a detached pipeline whose subprocess crashes (e.g., OOM kill)? Stale run detection must identify the dead process and transition the run to "failed" status.
- What happens if two TUI instances attempt to cancel the same pipeline simultaneously? The cancellation table uses `ON CONFLICT` semantics, so concurrent cancellation requests are idempotent.
- How does the system detect that a detached subprocess has died without a clean exit (stale "running" state in SQLite)? PID liveness checks via `os.FindProcess` + signal 0 are used; if the PID is not alive, the run is transitioned to "failed".
- What happens when the user launches a pipeline and immediately quits before the subprocess is fully started? The subprocess spawn must complete before the TUI is allowed to exit, or a "starting" state must be tracked.
- How does disk space exhaustion affect the detached subprocess writing to the SQLite event log? The subprocess must handle SQLite write failures gracefully and continue execution where possible.

## Requirements _(mandatory)_

### Functional Requirements

- **FR-001**: `PipelineLauncher.Launch()` MUST spawn pipeline execution as a detached OS subprocess (surviving parent process exit) instead of an in-process goroutine. The subprocess MUST be launched by re-executing the `wave` binary via `exec.Command("wave", "run", ...)` with `SysProcAttr{Setsid: true}` to create a new session, ensuring it survives parent exit and terminal SIGHUP
- **FR-002**: The detached subprocess MUST use the same `wave run <pipeline> --input <input>` command path as the CLI, ensuring execution parity
- **FR-003**: The detached subprocess MUST persist all events to SQLite via the existing event logging pattern so they survive process boundaries
- **FR-004**: `CancelAll()` on TUI shutdown MUST NOT terminate detached pipeline subprocesses; it MUST only stop the TUI's monitoring/subscription of those pipelines
- **FR-005**: Pipeline cancellation from the TUI MUST use `store.RequestCancellation()` to persist a cancellation flag that the detached subprocess polls and honors
- **FR-006**: The detached subprocess MUST check `store.CheckCancellation()` every 5 seconds during step execution and perform graceful shutdown when a cancellation is detected
- **FR-007**: On TUI startup, the system MUST detect all pipelines in "running" state from the state store and display them in the Running section of the fleet view
- **FR-008**: The TUI MUST reconnect to running pipelines by tailing `store.GetEvents()` for active runs and displaying historical and live events
- **FR-009**: The system MUST detect stale "running" entries where the subprocess has died by using OS-level PID liveness checks (`os.FindProcess` + signal 0) and transition them to "failed" status with an appropriate error message. No heartbeat mechanism is required
- **FR-010**: The detached subprocess MUST run in its own session (via `Setsid: true`) so it is not killed by terminal SIGHUP or parent exit
- **FR-011**: The TUI MUST store the PID of the detached subprocess in the `pipeline_run` table (via a new `pid` INTEGER column) so that liveness can be checked across TUI restarts
- **FR-012**: The detached subprocess MUST NOT receive credentials or secrets via command-line arguments; environment passthrough MUST follow the existing `runtime.sandbox.env_passthrough` configuration

### Key Entities

- **Detached Run**: A pipeline execution running as an independent OS process, tracked by run ID in SQLite with PID metadata for liveness detection
- **Cancellation Record**: A persistent flag in the `cancellation` table that signals a detached subprocess to stop, including force/graceful semantics
- **Event Log**: SQLite-persisted stream of pipeline progress events, serving as the communication channel between detached subprocess and TUI/WebUI

## Success Criteria _(mandatory)_

### Measurable Outcomes

- **SC-001**: A pipeline launched from the TUI continues running after TUI exit — verified by process liveness and continued event emission in SQLite
- **SC-002**: Reopening the TUI displays all currently-running detached pipelines within the existing refresh cycle
- **SC-003**: Cancellation of a detached pipeline via the TUI results in graceful subprocess termination within 30 seconds; if the subprocess has not exited after 30 seconds, SIGKILL is sent to the process group
- **SC-004**: Historical events from a detached pipeline load into the TUI detail view when the pipeline is selected
- **SC-005**: `go test -race ./...` passes with all new and existing tests
- **SC-006**: Stale "running" pipelines whose subprocess has died are detected and marked as "failed" within 2 refresh cycles
- **SC-007**: No behavioral difference between CLI-launched and TUI-launched pipelines when inspected via `wave status`, WebUI, or `wave logs`

## Clarifications

### C1 — Subprocess Detachment Mechanism (FR-001)

**Question**: How should the TUI detach the pipeline subprocess — in-process goroutine with signal handling, `Setsid` for new session, double-fork daemon pattern, or re-exec of the `wave` binary?

**Resolution**: Re-exec via `exec.Command("wave", "run", ...)` with `SysProcAttr{Setsid: true}`. This creates a new session leader that survives parent exit and terminal SIGHUP. The adapter already uses `Setpgid: true` for process group isolation (`internal/adapter/claude.go:88-91`), so `Setsid` is the natural escalation for full detachment. Re-exec also provides CLI-TUI parity (FR-002) since both paths invoke the same `wave run` command. Double-fork is unnecessary — `Setsid` is the standard Go approach and avoids zombie process complexity.

### C2 — PID Storage Location (FR-011)

**Question**: Should the subprocess PID be stored in the existing `pipeline_run` table or a new dedicated `process` table?

**Resolution**: Add a `pid` INTEGER column to the existing `pipeline_run` table. The `pipeline_run` table already tracks run lifecycle metadata (status, started_at, completed_at, error_message) and PID is a natural extension of that. A separate table would require a JOIN for every liveness check with no benefit — PID has a 1:1 relationship with a run, there's no cardinality mismatch. The existing `RunRecord` Go struct gains a `PID int` field.

### C3 — Cancellation Polling Interval (FR-006)

**Question**: How frequently should the detached subprocess poll `store.CheckCancellation()`?

**Resolution**: Every 5 seconds. This balances responsiveness (users expect cancellation within seconds) against SQLite read overhead. The existing cancellation infrastructure (`cancellation` table with `ON CONFLICT` semantics) is designed for polling. 5 seconds ensures cancellation completes well within the 30-second SC-003 window while keeping database load trivial.

### C4 — Stale Run Detection Mechanism (FR-009)

**Question**: Should stale detection use PID liveness checks or a heartbeat-based timeout?

**Resolution**: PID liveness checks via `os.FindProcess(pid)` + `process.Signal(syscall.Signal(0))`. This is zero-overhead (no writes required from the subprocess), works immediately when a process dies, and is the standard Unix approach for process liveness. A heartbeat mechanism would require the subprocess to write periodically to SQLite, adding complexity and write contention. PID reuse is a theoretical concern but negligible in practice for the short lifetimes involved.

### C5 — Post-Cancellation Grace Period Escalation (SC-003)

**Question**: What happens if the detached subprocess does not terminate within the 30-second cancellation grace period?

**Resolution**: After 30 seconds, send SIGKILL to the process group. The subprocess will have already received a graceful cancellation signal via `store.CheckCancellation()`. If it hasn't responded within 30 seconds, it's likely stuck (e.g., blocked on I/O or a hung adapter subprocess). SIGKILL to the process group ensures all child processes (including Claude Code subprocesses) are terminated. The run state is updated to "failed" with a "cancellation timeout — force killed" error message.
