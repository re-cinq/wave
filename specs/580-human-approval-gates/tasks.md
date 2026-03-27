# Tasks

## Phase 1: Core Types & Data Model

- [ ] Task 1.1: Add `GateChoice` struct and extend `GateConfig` with `Choices`, `Freeform`, `Default` fields in `internal/pipeline/types.go`
- [ ] Task 1.2: Add `GateDecision` struct in `internal/pipeline/gate_handler.go` capturing choice, label, text, timestamp, target
- [ ] Task 1.3: Define `GateHandler` interface with `Prompt(ctx, *GateConfig) (*GateDecision, error)` in `internal/pipeline/gate_handler.go`
- [ ] Task 1.4: Add `GateDecisions map[string]*GateDecision` to `PipelineContext` and implement `gate.<step>.*` template variable resolution in `internal/pipeline/context.go`
- [ ] Task 1.5: Add `GateConfig.Validate()` method to validate choice keys are unique, default references a valid choice, targets reference valid step IDs or `_fail`

## Phase 2: Gate Handlers

- [ ] Task 2.1: Implement `AutoApproveHandler` that returns the default choice immediately [P]
- [ ] Task 2.2: Implement `CLIGateHandler` using terminal stdin for interactive choice selection and optional freeform input [P]
- [ ] Task 2.3: Write unit tests for `AutoApproveHandler` and `CLIGateHandler` (mocked stdin) in `internal/pipeline/gate_handler_test.go` [P]

## Phase 3: Gate Executor Refactor

- [ ] Task 3.1: Refactor `GateExecutor.executeApproval` to delegate to `GateHandler` instead of blocking on context/timeout
- [ ] Task 3.2: Add `GateHandler` field to `GateExecutor` and wire it via `NewGateExecutor` constructor
- [ ] Task 3.3: Write gate decision to `.wave/artifacts/gate-<step>-text` when freeform text is provided; register as step artifact
- [ ] Task 3.4: Store `GateDecision` in `PipelineContext.GateDecisions` after handler returns
- [ ] Task 3.5: Update existing gate tests to work with the new handler-based flow; ensure timer/pr_merge/ci_pass still pass

## Phase 4: Choice Routing & Executor Integration

- [ ] Task 4.1: Add `WithAutoApprove(bool)` executor option to `DefaultPipelineExecutor`
- [ ] Task 4.2: Propagate `autoApprove` from executor to `CompositionExecutor.executeGate`
- [ ] Task 4.3: Implement choice routing in executor main loop: `_fail` -> pipeline failure, `<step-id>` -> reset step to pending and re-enter loop
- [ ] Task 4.4: Wire gate decisions into `PipelineContext` so downstream steps resolve `{{ gate.<step>.choice }}` etc.
- [ ] Task 4.5: Add `--auto-approve` flag to `RunOptions` in `cmd/wave/commands/run.go` and pass to executor construction

## Phase 5: TUI & WebUI Handlers

- [ ] Task 5.1: Create `TUIGateHandler` with Bubble Tea modal component in `internal/tui/gate_modal.go` [P]
- [ ] Task 5.2: Add `POST /api/runs/{id}/gates/{step}/approve` REST endpoint in `internal/webui/handlers_control.go` [P]
- [ ] Task 5.3: Create `WebUIGateHandler` that writes decision to state store and waits for resolution [P]

## Phase 6: Testing & Validation

- [ ] Task 6.1: Unit tests for gate decision template variable resolution
- [ ] Task 6.2: Unit tests for choice routing (approve, revise/re-queue, abort/_fail)
- [ ] Task 6.3: Integration test: plan -> approve-gate -> implement pipeline with auto-approve
- [ ] Task 6.4: Integration test: revision loop (approve-gate selects "Revise" -> re-queues plan step)
- [ ] Task 6.5: Verify all existing gate tests pass (`go test ./internal/pipeline/... -run Gate`)
- [ ] Task 6.6: Run full test suite with race detector (`go test -race ./...`)

## Phase 7: Polish

- [ ] Task 7.1: Validate `--auto-approve` required when `--detach` is used with pipelines containing approval gates
- [ ] Task 7.2: Add example pipeline YAML demonstrating plan-approve-implement pattern
- [ ] Task 7.3: Ensure backward compatibility: pipelines using `gate: { type: approval, auto: true }` without choices still work
