# Tasks

## Phase 1: DebugTracer slog Backend

- [X] Task 1.1: Add `log/slog` based handler to `DebugTracer` that writes `TraceEvent` NDJSON to the trace file
- [X] Task 1.2: Add `WithStderrMirror(bool)` option to `DebugTracer` that writes human-readable debug lines to stderr
- [X] Task 1.3: Add new trace event types: `prompt_load`, `prompt_load_error`, `artifact_write`, `artifact_skip_empty`, `artifact_preserved`
- [X] Task 1.4: Update `NewDebugTracer` to accept functional options pattern (`DebugTracerOption`)

## Phase 2: Executor Debug Unification

- [X] Task 2.1: Replace `executor.go:1934-1949` prompt loading `[DEBUG]` prints with `e.trace()` calls using `prompt_load` / `prompt_load_error` event types [P]
- [X] Task 2.2: Replace `executor.go:1540` empty ResultContent `[DEBUG]` print with `e.trace()` using `artifact_skip_empty` event type [P]
- [X] Task 2.3: Replace `executor.go:2284-2305` artifact write `[DEBUG]` prints with `e.trace()` calls using `artifact_write` / `artifact_preserved` event types [P]
- [X] Task 2.4: Remove unused `fmt` / `os` imports if all debug prints are converted

## Phase 3: NDJSONEmitter Debug Verbosity

- [X] Task 3.1: Add `SetDebugVerbosity(bool)` to `NDJSONEmitter` that enables emission of internal state transition events
- [X] Task 3.2: Wire `--debug` flag to emitter debug verbosity in `cmd/wave/commands/run.go`

## Phase 4: Testing

- [X] Task 4.1: Add unit tests for slog-backed `DebugTracer` — NDJSON output format, credential scrubbing, stderr mirror [P]
- [X] Task 4.2: Add unit tests for new trace event types emitted by executor [P]
- [X] Task 4.3: Add unit tests for `NDJSONEmitter` debug verbosity mode [P]
- [X] Task 4.4: Run `go test -race ./...` full validation

## Phase 5: Wire and Validate

- [X] Task 5.1: Update `run.go` to pass `WithStderrMirror(true)` when `--debug` is set
- [X] Task 5.2: Verify trace file backward compatibility — `ReadTraceFile` still parses new event types
- [X] Task 5.3: Final `go test -race ./...` and `golangci-lint run ./...`
