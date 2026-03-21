# refactor(debug): unify debug output through DebugTracer

**Issue**: [#537](https://github.com/re-cinq/wave/issues/537)
**Author**: nextlevelshit
**State**: OPEN
**Labels**: none
**Source**: DX audit §4 (Debugging)

## Problem Statement

Two parallel debug systems exist but are not unified:

1. `fmt.Fprintf(os.Stderr, "[DEBUG] ...")` — unstructured, inline in executor
2. `audit.DebugTracer` — structured NDJSON to `.wave/traces/`

Most debug output goes through the unstructured path. The executor is 3000+ lines but only a handful of call sites use `e.trace()`.

## Tasks

- Route all `[DEBUG]` stderr prints through `DebugTracer` as structured trace events
- Add `--debug` mode that increases `NDJSONEmitter` verbosity (emit internal state transitions, prompt assembly details)
- Consider adopting `log/slog` (stdlib since Go 1.21) as the unified logging backend

## Current State (Codebase Analysis)

### Unstructured debug output (`e.debug` + `fmt.Fprintf`)
- `executor.go:1935-1949` — 4 `[DEBUG]` prints for prompt loading from `source_path`
- `executor.go:1540` — 1 `[DEBUG]` print for empty ResultContent warning
- `executor.go:2284-2305` — 3 `[DEBUG]` prints for stdout artifact writing
- Total: **8 sites** in executor guarded by `e.debug` bool

### Structured trace output (`e.trace()` via DebugTracer)
- `executor.go:1226` — `artifact_injection` timing
- `executor.go:1405,1413,1477` — `adapter_start`/`adapter_end` timing
- `executor.go:1600,1616,1642,1658` — `contract_validation_start`/`contract_validation_end`
- Total: **8 sites** in executor using structured tracer

### Broader stderr usage
- `internal/` — 40 `fmt.Fprintf(os.Stderr` calls across 9 files
- `cmd/` — 100 calls across 12 files
- Most are user-facing status messages, NOT debug output

## Acceptance Criteria

1. All `[DEBUG]`-prefixed stderr prints in executor.go are replaced with `e.trace()` calls emitting structured `TraceEvent`s
2. The `--debug` flag continues to work: enables both trace file output AND increased verbosity
3. `NDJSONEmitter` gains a debug verbosity mode that emits internal state transitions (step state changes, prompt assembly, artifact injection details)
4. `log/slog` adoption is evaluated; if adopted, `DebugTracer` uses slog as its backend
5. No regression in existing debug trace file output format (NDJSON)
6. All changes pass `go test -race ./...`
