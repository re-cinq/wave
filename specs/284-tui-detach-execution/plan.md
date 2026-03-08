# Implementation Plan: Detach Pipeline Execution from TUI Process Lifecycle

**Branch**: `284-tui-detach-execution` | **Date**: 2026-03-08 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/284-tui-detach-execution/spec.md`

## Summary

Decouple pipeline execution from the TUI process lifecycle by spawning pipelines as detached OS subprocesses via `exec.Command("wave", "run", ...)` with `SysProcAttr{Setsid: true}`. This allows pipelines to survive TUI exit, enables cross-process cancellation via SQLite's existing `cancellation` table, and provides event reconnection when the TUI reopens. The approach reuses the existing `wave run` CLI code path for TUI-CLI execution parity and leverages SQLite WAL mode as the sole IPC channel between the detached subprocess and the TUI.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**: `github.com/charmbracelet/bubbletea` (TUI), `github.com/spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config)
**Storage**: SQLite via `modernc.org/sqlite` (existing `.wave/state.db`)
**Testing**: `go test -race ./...` with `github.com/stretchr/testify`
**Target Platform**: Linux, macOS (Setsid is Unix-specific; Windows not in scope)
**Project Type**: Single binary CLI/TUI application
**Performance Goals**: Subprocess spawn < 100ms; cancellation response < 30s (SC-003); stale detection within 2 refresh cycles (SC-006)
**Constraints**: No new dependencies; single binary constraint; no credentials in CLI args (FR-012)
**Scale/Scope**: 4-6 concurrently detached pipelines typical; state.db shared via WAL

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| P1: Single Binary | PASS | Re-exec uses the same `wave` binary — no new dependencies |
| P2: Manifest as Source of Truth | PASS | Subprocess loads its own manifest via `wave run` |
| P3: Persona-Scoped Execution | PASS | Subprocess follows standard `wave run` persona binding |
| P4: Fresh Memory at Step Boundary | PASS | Subprocess starts with fresh context (separate process) |
| P5: Navigator-First Architecture | PASS | Pipeline structure unchanged — subprocess executes same DAG |
| P6: Contracts at Every Handover | PASS | Subprocess performs same contract validation |
| P7: Relay via Dedicated Summarizer | PASS | Relay mechanism unchanged within subprocess |
| P8: Ephemeral Workspaces | PASS | Subprocess creates its own workspaces |
| P9: Credentials Never Touch Disk | PASS | FR-012 enforces env-only credential passing |
| P10: Observable Progress | PASS | Events persisted to SQLite, visible via TUI, WebUI, `wave logs` |
| P11: Bounded Recursion | PASS | Same resource limits apply within subprocess |
| P12: Minimal Step State Machine | PASS | Same 5-state machine |
| P13: Test Ownership | PASS | All existing tests must pass + new tests for detached behavior |

**Post-Phase 1 Re-check**: No violations introduced by design.

## Project Structure

### Documentation (this feature)

```
specs/284-tui-detach-execution/
├── plan.md              # This file
├── research.md          # Phase 0: Technology decisions and rationale
├── data-model.md        # Phase 1: Entity definitions and data flow
└── tasks.md             # Phase 2 output (not created by plan)
```

### Source Code (repository root)

```
internal/
├── state/
│   ├── store.go           # MODIFY: Add UpdateRunPID, GetRunPID; scan PID in queryRuns
│   ├── schema.sql         # MODIFY: Add pid column to pipeline_run
│   └── types.go           # MODIFY: Add PID field to RunRecord
├── tui/
│   ├── pipeline_launcher.go    # MODIFY: Rewrite Launch() for subprocess detach
│   ├── pipeline_launcher_test.go # MODIFY: Update tests for new Launch behavior
│   ├── pipeline_provider.go    # MODIFY: Add PID to RunningPipeline
│   ├── pipeline_messages.go    # MODIFY: Update PipelineLaunchedMsg (remove CancelFunc)
│   ├── pipeline_list.go        # MODIFY: Update stale run detection display
│   ├── content.go              # MODIFY: Cancel via store instead of context; CancelAll behavior
│   ├── app.go                  # MODIFY: CancelAll behavior on quit (no subprocess kill)
│   ├── live_output.go          # MODIFY: Support populating buffer from historical events
│   └── stale_detector.go       # NEW: StaleRunDetector for PID liveness checks
├── pipeline/
│   └── executor.go        # NO CHANGE: Executor runs inside subprocess, unchanged
└── adapter/
    └── claude.go          # NO CHANGE: Adapter's Setpgid unchanged, subprocess Setsid is higher level

cmd/wave/commands/
└── run.go                 # MODIFY: Support --run flag to reuse pre-created run ID
```

**Structure Decision**: All changes are within the existing Go package structure. One new file (`internal/tui/stale_detector.go`) for the PID liveness checking logic, keeping it separated from the launcher. No new packages needed.

## Design Details

### D1 — PipelineLauncher Subprocess Spawn

The core change: `PipelineLauncher.Launch()` spawns a detached subprocess instead of running in-process.

```go
func (l *PipelineLauncher) Launch(config LaunchConfig) tea.Cmd {
    // 1. Create run record in state store (returns run_id)
    runID, err := l.deps.Store.CreateRun(pipelineName, input)

    // 2. Build subprocess command
    args := []string{"run", pipelineName, "--run", runID, "--input", input}
    cmd := exec.Command(os.Args[0], args...)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
    cmd.Env = buildPassthroughEnv()  // FR-012: env only, no credentials in args

    // 3. Start subprocess (non-blocking)
    cmd.Start()

    // 4. Store PID for liveness detection
    l.deps.Store.UpdateRunPID(runID, cmd.Process.Pid)

    // 5. Release cmd.Process — don't wait for it
    cmd.Process.Release()

    return func() tea.Msg {
        return PipelineLaunchedMsg{RunID: runID, PipelineName: pipelineName}
    }
}
```

### D2 — `wave run --run` Flag

The CLI `run.go` already has a `--run` flag for resume. Reuse it: when `--run` is specified, use that run ID instead of generating a new one. The TUI creates the run record, so the subprocess reuses it and sets status to "running".

This ensures the TUI-created `run_id` and the subprocess's `run_id` are the same — both reference the same row in `pipeline_run`.

### D3 — CancelAll Behavior Change

```go
// CancelAll on TUI shutdown does NOT kill subprocesses (FR-004).
// It only cleans up TUI-side monitoring state.
func (l *PipelineLauncher) CancelAll() {
    l.mu.Lock()
    defer l.mu.Unlock()
    // Clear internal maps — subprocesses continue independently
    l.cancelFns = make(map[string]context.CancelFunc) // for any in-process monitoring
}
```

### D4 — Cancel via Store

```go
// Cancel sends cancellation via persistent store (FR-005)
func (l *PipelineLauncher) Cancel(runID string) {
    if l.deps.Store != nil {
        l.deps.Store.RequestCancellation(runID, false)
    }
}
```

The subprocess polls `store.CheckCancellation()` every 5 seconds (FR-006). This polling must be integrated into the executor's step loop — either as a wrapper around `adapter.Run()` or as a parallel goroutine during step execution.

### D5 — Stale Run Detection

On TUI startup and each refresh tick:
1. `store.GetRunningRuns()` returns all "running" records with PIDs
2. For each record with `PID > 0`, check `IsProcessAlive(pid)`
3. If dead, transition to "failed" with "stale: subprocess exited unexpectedly"

### D6 — Event Reconnection

When the user selects a running detached pipeline:
1. Load all events via `store.GetEvents(runID, EventQueryOptions{})`
2. Format each into display lines using existing `formatEventLine()`
3. Populate an `EventBuffer` and render in `LiveOutputModel`
4. Start polling with `After: lastTimestamp` for new events

### D7 — Cancellation Polling in Executor

The executor needs a cancellation check mechanism. Two options:
- **Option A**: Add cancellation polling directly in `executeStep()` as a goroutine alongside adapter execution
- **Option B**: Wrap the adapter run context with a cancellation goroutine

Option A is simpler — add a goroutine in `executeStep` that polls `store.CheckCancellation` every 5 seconds and cancels the step context if found.

## Complexity Tracking

_No constitution violations identified._

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| (none)    | —         | —                                    |
