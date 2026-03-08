# Research: Detach Pipeline Execution from TUI Process Lifecycle

**Feature Branch**: `284-tui-detach-execution`
**Date**: 2026-03-08

## R1 — Process Detachment in Go

### Decision: `exec.Command` re-exec with `Setsid: true`

**Rationale**: The spec prescribes re-executing the `wave` binary via `exec.Command("wave", "run", ...)` with `SysProcAttr{Setsid: true}`. This creates a new session leader that:
- Survives parent process exit (not in parent's process group)
- Survives terminal SIGHUP (new session = no controlling terminal)
- Shares the same `wave run` code path as CLI (execution parity FR-002)

The adapter layer already uses `Setpgid: true` for process group isolation (`internal/adapter/claude.go:88-91`). `Setsid` is the natural escalation for full TUI detachment — it creates a new session containing a new process group.

**Alternatives Rejected**:
- **In-process goroutine** (current approach): Goroutine dies when TUI exits. Cannot survive parent process lifecycle.
- **Double-fork daemon pattern**: Overly complex for Go. `Setsid` achieves the same goal without zombie process management.
- **`nohup`/background shell exec**: Platform-dependent, fragile, doesn't integrate with Go's `exec.Cmd` API.

### Implementation Approach

1. `PipelineLauncher.Launch()` uses `exec.Command(os.Args[0], "run", pipelineName, "--input", input)` with `SysProcAttr{Setsid: true}`
2. Environment variables are passed via `cmd.Env` using the existing `env_passthrough` mechanism — no credentials in CLI arguments (FR-012)
3. After `cmd.Start()`, the PID is stored in the `pipeline_run` table
4. The goroutine returns `PipelineLaunchedMsg` immediately — no blocking wait
5. The subprocess handles its own state transitions and event logging independently

### Binary Discovery

Use `os.Args[0]` to find the currently-running `wave` binary. This is the same binary the user is running, so it's guaranteed to be available and the correct version. If the binary was invoked via `$PATH`, `os.Args[0]` may be just `"wave"`, which is fine — `exec.LookPath` resolves it.

## R2 — Inter-Process Communication via SQLite

### Decision: SQLite WAL mode as the sole IPC channel

**Rationale**: The codebase already uses SQLite with WAL mode (`PRAGMA journal_mode=WAL`) and busy timeout (`PRAGMA busy_timeout=5000`). WAL mode supports concurrent readers with a single writer, making it ideal for the detached subprocess (writer) and TUI (reader) pattern.

Existing infrastructure:
- `event_log` table: Already stores events per run (`store.LogEvent`)
- `pipeline_run` table: Already tracks status, current step, tokens
- `cancellation` table: Already supports `RequestCancellation` / `CheckCancellation` with `ON CONFLICT` semantics
- `dbLoggingEmitter` in `cmd/wave/commands/run.go:601-617`: Already wraps emitters with DB persistence

No new IPC mechanism needed — the existing SQLite tables provide everything required.

**Alternatives Rejected**:
- **Unix domain sockets / TCP**: Requires connection management, crashes leave sockets dangling
- **Shared memory / mmap**: Complex, no existing Go abstraction in the codebase
- **Named pipes**: Unidirectional, no persistence, platform-dependent

## R3 — PID Liveness Detection

### Decision: `os.FindProcess` + `Signal(0)` for stale run detection

**Rationale**: Per spec FR-009, PID liveness checks via `os.FindProcess(pid)` + `process.Signal(syscall.Signal(0))` provide zero-overhead detection:
- No writes required from the subprocess (unlike heartbeats)
- Immediate detection when process dies
- Standard Unix approach for process liveness
- `Signal(0)` doesn't actually send a signal — it just checks if the process exists and the caller has permission to signal it

**Edge Case — PID Reuse**: Theoretically, after a process dies, its PID could be reassigned to a new process. For Wave's use case (pipeline runs lasting minutes to hours), this is negligible. The probability of PID reuse matching the exact same PID within a TUI refresh cycle (seconds) is extremely low on modern systems with large PID spaces.

**Implementation**: On TUI startup and on each refresh tick, iterate over "running" runs with PIDs. For each, call `os.FindProcess(pid)` then `p.Signal(syscall.Signal(0))`. If the signal returns an error (process not found or permission denied on a different process), transition the run to "failed" with "process not found — stale run" error.

## R4 — Cancellation via Persistent Store

### Decision: Poll-based cancellation via `store.CheckCancellation()`

**Rationale**: The `cancellation` table already exists with the right schema:
```sql
CREATE TABLE IF NOT EXISTS cancellation (
    run_id TEXT PRIMARY KEY,
    requested_at INTEGER NOT NULL,
    force BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (run_id) REFERENCES pipeline_run(run_id) ON DELETE CASCADE
);
```

The existing `RequestCancellation` method uses `ON CONFLICT` for idempotent concurrent requests. The subprocess polls `CheckCancellation` every 5 seconds (FR-006) and initiates graceful shutdown when found.

**Grace Period Escalation (SC-003)**: After persisting cancellation, the TUI starts a 30-second timer. If the process is still alive after 30 seconds, `syscall.Kill(-pid, syscall.SIGKILL)` is sent to the process group (negative PID = kill entire group). The run status is updated to "failed" with "cancellation timeout — force killed".

## R5 — TUI Shutdown Behavior Change

### Decision: `CancelAll()` stops monitoring only, does not kill subprocesses

**Rationale**: Currently `CancelAll()` cancels all in-process goroutine contexts, killing pipelines. With detached execution, `CancelAll()` must only clean up TUI-side resources (cancel context of monitoring goroutines, clear internal maps) WITHOUT sending termination signals to subprocess PIDs.

The subprocess continues independently — it manages its own lifecycle, state transitions, and cleanup.

## R6 — Event Reconnection on TUI Reopen

### Decision: Load historical events from `store.GetEvents()` on selection

**Rationale**: When the TUI reopens and detects running pipelines, it shows them in the fleet view. When the user selects a running pipeline, the TUI:
1. Calls `store.GetEvents(runID, ...)` to load historical events
2. Populates the `EventBuffer` with formatted event lines
3. Starts tailing new events by polling `GetEvents` with timestamp cursor
4. Displays in the `LiveOutputModel` viewport

This reuses existing event infrastructure without adding a new streaming mechanism.

## R7 — Schema Migration for PID Column

### Decision: Add `pid` INTEGER column to `pipeline_run` table

**Rationale**: Per spec FR-011 and C2, PID storage belongs in `pipeline_run` (1:1 relationship with runs). The codebase currently uses schema.sql for initialization. A new `ALTER TABLE` migration adds the `pid` column:

```sql
ALTER TABLE pipeline_run ADD COLUMN pid INTEGER;
```

The `RunRecord` Go struct gains a `PID int` field. The `CreateRun` signature changes to accept a PID, or a separate `UpdateRunPID(runID, pid)` method is added.
