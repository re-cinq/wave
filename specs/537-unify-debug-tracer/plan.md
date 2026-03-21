# Implementation Plan: Unify Debug Output through DebugTracer

## 1. Objective

Replace the dual debug system (unstructured `[DEBUG]` stderr prints + structured `DebugTracer` NDJSON) with a single unified debug output path through `DebugTracer`. Enhance `--debug` mode to emit richer internal state transitions.

## 2. Approach

### Phase A: Unify executor debug output
Replace all `fmt.Fprintf(os.Stderr, "[DEBUG]...")` calls in `executor.go` with `e.trace()` calls. Each existing debug print becomes a structured `TraceEvent` with an appropriate `event_type` and metadata. The `e.debug` bool remains as a gate for verbose-only trace events (ones that are too noisy for normal `--debug` use).

### Phase B: Enhance DebugTracer with stderr mirror
Add an option to `DebugTracer` to mirror trace events to stderr in a human-readable format when `--debug` is active. This preserves the developer experience of seeing debug output in the terminal while also writing structured NDJSON to the trace file.

### Phase C: Add verbose debug events
Add new trace event types for internal state transitions: `prompt_assembly`, `artifact_write`, `step_state_change`. These fire only when `--debug` is set, giving developers visibility into prompt construction and artifact flow.

### Phase D: Evaluate log/slog adoption
Assess whether `log/slog` (stdlib Go 1.21+) should back the `DebugTracer`. Given Wave targets Go 1.25+, slog is available. The recommendation is to **adopt slog as the backend** for `DebugTracer`, replacing the raw `json.Marshal` + file write with a `slog.Handler` that writes NDJSON. This gives structured logging for free, with level-based filtering.

## 3. File Mapping

| File | Action | Description |
|------|--------|-------------|
| `internal/audit/trace.go` | modify | Add slog backend, stderr mirror option, new event types |
| `internal/audit/trace_test.go` | modify | Tests for slog backend, stderr mirror, new event types |
| `internal/pipeline/executor.go` | modify | Replace 8 `fmt.Fprintf` debug sites with `e.trace()` calls |
| `internal/event/emitter.go` | modify | Add debug verbosity mode to `NDJSONEmitter` |
| `internal/event/emitter_test.go` | modify | Tests for debug verbosity mode |
| `cmd/wave/commands/run.go` | modify | Wire debug verbosity into emitter |

## 4. Architecture Decisions

### AD-1: slog as DebugTracer backend
**Decision**: Adopt `log/slog` as the logging backend for `DebugTracer`.
**Rationale**: Go 1.21+ stdlib, structured by design, supports custom handlers. Wave already targets Go 1.25+. A custom `slog.Handler` can write NDJSON to the trace file AND optionally mirror to stderr.
**Trade-off**: Slightly more abstraction, but eliminates manual `json.Marshal` and gains level filtering.

### AD-2: Preserve TraceEvent as the external format
**Decision**: Keep `TraceEvent` struct as the NDJSON output schema; slog handler maps `slog.Record` → `TraceEvent` internally.
**Rationale**: Existing trace consumers (ReadTraceFile, TUI, webui) depend on the `TraceEvent` format. Changing it would break downstream tools.

### AD-3: Stderr mirror is opt-in
**Decision**: `DebugTracer` gains a `WithStderrMirror(bool)` option. When enabled, each `Emit()` also writes a human-readable line to stderr.
**Rationale**: Replaces the unstructured `[DEBUG]` prints while keeping terminal visibility. Only active in `--debug` mode.

### AD-4: Scope boundary — executor only for Phase 1
**Decision**: Only convert executor.go debug prints in this issue. The broader 140 stderr calls across `cmd/` and `internal/` are user-facing status messages, not debug output, and are out of scope.
**Rationale**: The issue explicitly targets `[DEBUG]` prints and `e.trace()` unification. Over-scoping would touch 20+ files and risk regressions.

## 5. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking trace file format | Low | High | Keep `TraceEvent` struct unchanged; slog handler maps to it |
| Missing debug output after migration | Medium | Medium | Each converted site gets equivalent metadata in TraceEvent |
| slog handler complexity | Low | Low | Use `slog.NewJSONHandler` with custom output; well-documented stdlib |
| Performance regression from slog | Very Low | Low | slog is designed for production use; benchmark if concerned |

## 6. Testing Strategy

- **Unit tests**: `internal/audit/trace_test.go` — test slog backend writes valid NDJSON, stderr mirror outputs human-readable lines, credential scrubbing still works
- **Unit tests**: `internal/event/emitter_test.go` — test debug verbosity mode emits additional events
- **Integration**: `go test -race ./internal/pipeline/...` — ensure executor debug output still works end-to-end with mock adapter
- **Manual**: `wave run --debug <pipeline>` — verify trace file contains new event types, stderr shows debug lines
