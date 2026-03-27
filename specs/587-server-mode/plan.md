# Implementation Plan: Server Mode Bug Fixes

## Objective

Fix three critical bugs in Wave's existing server mode implementation that prevent the webui package from compiling and the gate approval flow from working end-to-end.

## Approach

The server mode feature is **already implemented** — all REST API endpoints, SSE streaming, scheduling, auth modes, TLS, and manifest config exist. The work is fixing integration bugs:

1. **Fix stale gate tests** — rewrite `handlers_gate_test.go` against the current `GateRegistry` API in `gate_handler.go`
2. **Fix gateRegistry initialization** — consolidate the duplicate `gates`/`gateRegistry` fields in `Server` struct
3. **Wire gate handler into executor** — pass `WithGateHandler` using `WebUIGateHandler` in `launchPipelineExecution`

## File Mapping

### Modified Files

| Path | Changes |
|------|---------|
| `internal/webui/handlers_gate_test.go` | Rewrite tests against current `GateRegistry` API (3-arg Register, decision-based Resolve, Remove not Cleanup, handleGateApprove not handleResolveGate) |
| `internal/webui/server.go` | Remove duplicate `gates` field, keep `gateRegistry` initialized to `NewGateRegistry()` |
| `internal/webui/handlers_control.go` | Add `pipeline.WithGateHandler(NewWebUIGateHandler(runID, s.gateRegistry))` to executor options in `launchPipelineExecution` |

### No New Files

All infrastructure already exists.

## Architecture Decisions

### 1. Consolidate to `gateRegistry` field (not `gates`)

The handler code already references `s.gateRegistry`. Remove `gates` and initialize `gateRegistry` in `NewServer`. This is the minimal change.

### 2. WebUIGateHandler per-run (existing pattern)

`NewWebUIGateHandler(runID, registry)` creates a handler scoped to a run ID. This is already the intended pattern — just needs wiring.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Gate tests may reveal additional API mismatches | Medium | Full coverage of Register/Resolve/Remove/GetPending/GetPendingStepID |
| WithGateHandler may conflict with auto-approve | Low | Check executor code — auto-approve is a separate option |

## Testing Strategy

- Fix `handlers_gate_test.go` to compile and pass against current API
- Run `go test -race ./internal/webui/...` to verify package compiles
- Run `go test -race ./...` to verify no regressions
