# Tasks

## Phase 1: Fix Gate Registry Initialization (Bug 2)

- [X] Task 1.1: Consolidate `gates` and `gateRegistry` fields in `internal/webui/server.go` — remove `gates *GateRegistry` field, keep `gateRegistry *GateRegistry`, initialize `gateRegistry: NewGateRegistry()` in `NewServer`
- [X] Task 1.2: Update any references to `s.gates` elsewhere in webui package to use `s.gateRegistry`

## Phase 2: Wire Gate Handler into Executor (Bug 3)

- [X] Task 2.1: In `launchPipelineExecution` (`internal/webui/handlers_control.go`), add `pipeline.WithGateHandler(NewWebUIGateHandler(runID, s.gateRegistry))` to the executor options

## Phase 3: Fix Gate Tests (Bug 1)

- [X] Task 3.1: Rewrite `internal/webui/handlers_gate_test.go` against current API — use 3-arg `Register(runID, stepID, *pipeline.GateConfig)`, decision-based `Resolve(runID, *pipeline.GateDecision)` returning error, `Remove(runID)` for cleanup, `GetPending(runID)`, `GetPendingStepID(runID)` [P]
- [X] Task 3.2: Rewrite HTTP handler tests to use `handleGateApprove` (not `handleResolveGate`) with proper `GateApproveRequest` JSON body [P]

## Phase 4: Validation

- [X] Task 4.1: Run `go test -race ./internal/webui/...` — verify package compiles and all tests pass
- [X] Task 4.2: Run `go test -race ./...` — verify no regressions across project
- [X] Task 4.3: Run `golangci-lint run ./...` — verify no lint issues
