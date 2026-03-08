# Tasks: Detach Pipeline Execution from TUI Process Lifecycle

**Feature Branch**: `284-tui-detach-execution`
**Generated**: 2026-03-08
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

---

## Phase 1 — Data Layer: Schema & State Store (Foundational)

These tasks modify the state store and data types. All subsequent phases depend on these.

- [X] T001 [P1] [Story1] Add `PID` field to `RunRecord` struct in `internal/state/types.go`. Add `PID int` field after `BranchName` with comment `// OS process ID of detached subprocess (0 = in-process or unknown)`.

- [X] T002 [P1] [Story1] Add SQLite migration to add `pid` column to `pipeline_run` table. Create a new migration in `internal/state/migrations.go` (or the existing migration registration pattern) with `ALTER TABLE pipeline_run ADD COLUMN pid INTEGER DEFAULT 0`. Follow the existing migration system pattern used by the codebase.

- [X] T003 [P1] [Story1] Update `queryRunsWithArgs` in `internal/state/store.go` to scan the new `pid` column into `RunRecord.PID`. Add `&record.PID` (or a `sql.NullInt64` intermediary) to the `rows.Scan()` call, and update all SELECT queries that feed `queryRunsWithArgs` to include `pid` in their column list.

- [X] T004 [P1] [Story1] Add `UpdateRunPID(runID string, pid int) error` method to the `StateStore` interface in `internal/state/store.go` and implement it on `stateStore`. SQL: `UPDATE pipeline_run SET pid = ? WHERE run_id = ?`.

- [X] T005 [P1] [Story1] Add `GetRunPID(runID string) (int, error)` method to the `StateStore` interface in `internal/state/store.go` and implement it on `stateStore`. SQL: `SELECT pid FROM pipeline_run WHERE run_id = ?`.

- [X] T006 [P1] [Story1] Write unit tests for T001–T005 in `internal/state/store_test.go`: test `UpdateRunPID` sets PID, `GetRunPID` retrieves it, `queryRuns` includes PID in scanned records, and default PID is 0 for new runs.

---

## Phase 2 — Subprocess Detachment: PipelineLauncher Rewrite (Story 1 — Pipeline Survives TUI Exit)

Core behavioral change: Launch() spawns a detached subprocess instead of an in-process goroutine.

- [X] T007 [P1] [Story1] Rewrite `PipelineLauncher.Launch()` in `internal/tui/pipeline_launcher.go` to spawn a detached subprocess via `exec.Command(os.Args[0], "run", pipelineName, "--run", runID, "--input", input)` with `cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}`. Create the run record via `store.CreateRun()` before spawn, store PID via `store.UpdateRunPID()` after `cmd.Start()`, then call `cmd.Process.Release()`. Return `PipelineLaunchedMsg` (without `CancelFunc`). Build environment with `buildPassthroughEnv()` per FR-012.

- [X] T008 [P1] [Story1] Add `buildPassthroughEnv()` helper function in `internal/tui/pipeline_launcher.go` that constructs the subprocess environment from `runtime.sandbox.env_passthrough` manifest configuration. Only pass through explicitly allowed environment variables (FR-012: no credentials in CLI args).

- [X] T009 [P1] [Story1] Remove `CancelFunc` field from `PipelineLaunchedMsg` in `internal/tui/pipeline_messages.go`. Update all references to `PipelineLaunchedMsg.CancelFunc` in `internal/tui/pipeline_launcher.go` and `internal/tui/pipeline_list.go`.

- [X] T010 [P1] [Story1] Remove `cancelFns map[string]context.CancelFunc` and `buffers map[string]*EventBuffer` fields from `PipelineLauncher` struct. Remove `GetBuffer`, `HasBuffer` methods. Update `NewPipelineLauncher` constructor. These in-process constructs are replaced by SQLite-based IPC.

- [X] T011 [P1] [Story1] Update `Cleanup()` method on `PipelineLauncher` to be a no-op or remove it — detached subprocesses manage their own lifecycle. Remove buffer cleanup logic.

- [X] T012 [P1] [Story1] Update existing tests in `internal/tui/pipeline_launcher_test.go` to reflect the new subprocess-based architecture. Remove tests for `cancelFns` map, `buffers` map, and `CancelFunc` behavior. Add tests verifying that `Launch()` calls `store.CreateRun()` and `store.UpdateRunPID()` (using a mock store).

---

## Phase 3 — CLI Parity: `wave run --run` Support (Story 3 — TUI-CLI Execution Parity)

- [X] T013 [P2] [Story3] Modify `runRun()` in `cmd/wave/commands/run.go` to support `--run` flag for reusing a pre-created run ID. When `opts.RunID` is set and `opts.FromStep` is empty (not a resume), skip `store.CreateRun()` and use the provided run ID directly. Update the run status to "running" at execution start.

- [X] T014 [P2] [Story3] Write integration test verifying that `wave run --run <id>` reuses the specified run ID instead of creating a new one. Test in `cmd/wave/commands/run_test.go` or `internal/pipeline/executor_test.go`.

---

## Phase 4 — CancelAll & Cancel via Store (Story 2 — Cancel Detached Pipeline)

- [X] T015 [P2] [Story2] Rewrite `CancelAll()` on `PipelineLauncher` in `internal/tui/pipeline_launcher.go` to be a no-op for detached subprocesses (FR-004). It must NOT terminate subprocess PIDs — only clean up TUI-side monitoring state (if any remains).

- [X] T016 [P2] [Story2] Rewrite `Cancel(runID)` on `PipelineLauncher` in `internal/tui/pipeline_launcher.go` to call `store.RequestCancellation(runID, false)` instead of context cancellation (FR-005).

- [X] T017 [P2] [Story2] Add cancellation polling to the executor. In `internal/pipeline/executor.go`, add a goroutine in `executeStep()` (or a wrapper around the adapter run) that polls `store.CheckCancellation(runID)` every 5 seconds (FR-006). When cancellation is detected, cancel the step context to trigger graceful shutdown.

- [X] T018 [P2] [Story2] Add force-kill escalation for cancellation timeout (SC-003). In `internal/tui/pipeline_launcher.go` (or a new `internal/tui/cancel_manager.go`), after `RequestCancellation` is called, start a 30-second timer. If the subprocess PID is still alive after 30 seconds, send `syscall.Kill(-pid, syscall.SIGKILL)` to kill the process group and update the run to "failed" with "cancellation timeout — force killed".

- [X] T019 [P2] [Story2] Update the cancel keybinding handler in `internal/tui/content.go` (the `"c"` key handler around line 351) to use the new `Cancel(runID)` which writes to the store. Remove any direct context cancellation references.

- [X] T020 [P2] [Story2] Write tests for cancellation flow: test that `Cancel()` calls `store.RequestCancellation`, test that the executor polling goroutine detects cancellation and cancels the step context, test force-kill escalation after 30 seconds.

---

## Phase 5 — Stale Run Detection (Story 1 — Pipeline Survives TUI Exit, FR-009)

- [X] T021 [P1] [Story1] Create `internal/tui/stale_detector.go` with `StaleRunDetector` struct. Implement `DetectStaleRuns() ([]string, error)` which queries `store.GetRunningRuns()`, checks each run's PID via `IsProcessAlive(pid)`, and transitions dead runs to "failed" with "stale: subprocess exited unexpectedly". Also implement `IsProcessAlive(pid int) bool` using `os.FindProcess(pid)` + `process.Signal(syscall.Signal(0))`.

- [X] T022 [P1] [Story1] Integrate `StaleRunDetector` into the TUI refresh cycle. In the pipeline data provider or the refresh tick handler, call `DetectStaleRuns()` before returning running pipeline data. This ensures stale runs are detected within 2 refresh cycles (SC-006).

- [X] T023 [P1] [Story1] Write unit tests for `StaleRunDetector` in `internal/tui/stale_detector_test.go`: test `IsProcessAlive` returns true for current process PID, false for a known-dead PID, and `DetectStaleRuns` transitions stale runs to "failed".

---

## Phase 6 — TUI Pipeline Provider & List Updates (Story 1 + Story 4)

- [X] T024 [P1] [Story1] Add `PID int` and `Detached bool` fields to `RunningPipeline` struct in `internal/tui/pipeline_provider.go`. Update `FetchRunningPipelines()` to populate `PID` from `RunRecord.PID` and set `Detached = true` when `PID > 0`.

- [X] T025 [P3] [Story4] Update the running pipeline Enter handler in `internal/tui/content.go` (around line 309) to support reconnection for detached pipelines that have no in-process buffer. Instead of checking `launcher.HasBuffer(r.RunID)`, load historical events from `store.GetEvents(runID, ...)`, populate an `EventBuffer`, and create the `LiveOutputModel` from it.

- [X] T026 [P3] [Story4] Add event polling for live updates from detached pipelines. After populating the `LiveOutputModel` with historical events, start a periodic poll (using `tea.Tick`) that calls `store.GetEvents(runID, EventQueryOptions{Offset: lastCount})` to fetch new events and append them to the buffer.

- [X] T027 [P3] [Story4] Update `LiveOutputModel` in `internal/tui/live_output.go` to support initial population from a pre-filled `EventBuffer` (historical events). Ensure the viewport content is set correctly when historical events are loaded and auto-scroll jumps to the latest.

---

## Phase 7 — TUI Shutdown Behavior (Story 1)

- [X] T028 [P1] [Story1] Update `AppModel.Update()` in `internal/tui/app.go` so that `CancelAll()` on quit (both `q` and `ctrl+c`) does NOT kill detached pipelines. Verify that TUI shutdown proceeds even if pipelines are still running.

- [X] T029 [P1] [Story1] Handle the race condition where TUI exits during subprocess spawn (edge case from spec). Ensure that `cmd.Start()` completes before TUI can exit, or mark the run as "failed" if the spawn did not complete. Add a "starting" guard in `Launch()` that blocks TUI exit until the subprocess PID is stored.

---

## Phase 8 — Polish & Cross-Cutting

- [X] T030 [P] Update `internal/tui/pipeline_list.go` to display a visual indicator (e.g., a detach icon or label) for detached running pipelines to distinguish them from in-process runs.

- [X] T031 [P] Add `--run` flag documentation to the `wave run` help text in `cmd/wave/commands/run.go`. Update the `Long` description and `Example` section to mention the TUI-subprocess use case.

- [X] T032 Ensure `go test -race ./...` passes with all new and existing tests (SC-005). Run the full test suite to verify no regressions from the data layer, launcher, executor, and TUI changes.

- [X] T033 Verify end-to-end parity (SC-007): run the same pipeline via CLI (`wave run`) and TUI, confirm that `wave status` and event logs show identical behavior for both launch paths.
