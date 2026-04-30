# Work Items

## Phase 1: Setup

- [X] 1.1: Verify `state.PipelineEvalRecord` field set + `RecordEval` signature still match plan (read `internal/state/evolution.go`)
- [X] 1.2: Confirm `DefaultPipelineExecutor.store` concrete type satisfies `state.EvolutionStore` (type-assert in one place)

## Phase 2: Core Implementation

- [X] 2.1: Create `internal/contract/eval_signal.go` — `SignalKind` const block (success, failure, contract_failure, judge_score, duration, cost), `Signal` struct, `SignalSet` aggregator with `Add`, `FailureClass`, `Aggregate(runID, pipelineName, startedAt) state.PipelineEvalRecord` [P]
- [X] 2.2: Create `internal/pipeline/executor_eval.go` — per-execution collector map keyed by runID, `recordStepEval(execution, step, result, err)`, `recordPipelineEval(execution)`, `clearEvalCollector(runID)` [P]
- [X] 2.3: Add `evalCollectors map[string]*contract.SignalSet` + `mu` field on `DefaultPipelineExecutor`; initialize in `NewDefaultPipelineExecutor`
- [X] 2.4: Wire `e.recordStepEval(...)` into terminal step state transitions in `executor_steps.go` (after state set to completed/completed_empty/failed)
- [X] 2.5: Wire `e.recordPipelineEval(execution)` into `finalizePipelineExecution` (after `SavePipelineState`, before `runTerminalHooks`); cleanup folded into `cleanupCompletedPipeline`
- [X] 2.6: Map `failure_class`: contract_failure > failure > "" (in `SignalSet.FailureClass`)
- [X] 2.7: Read cost from `e.costLedger.TotalCost()` when non-nil; pass into aggregator
- [X] 2.8: UNIQUE-constraint swallow on `RecordEval` error (resume safety)

## Phase 3: Testing

- [X] 3.1: `internal/contract/eval_signal_test.go` — kind round-trip, aggregator priority, judge score averaging, empty set [P]
- [X] 3.2: `internal/pipeline/executor_eval_test.go` — 2-step pipeline (success + failure) → assert one `pipeline_eval` row with expected `FailureClass`, `DurationMs > 0`, `RetryCount` [P]
- [X] 3.3: All-success variant test [P]
- [X] 3.4: EvolutionStore-error stub → executor still finishes; no panic
- [X] 3.5: `go test -race ./internal/pipeline/... ./internal/contract/...` clean

## Phase 4: Polish

- [X] 4.1: Run `golangci-lint run ./internal/pipeline/... ./internal/contract/...`
- [X] 4.2: Add 1-line GoDoc on `SignalKind` and `Signal`; package-level comment in `executor_eval.go` referencing #1606
- [X] 4.3: Final `go test ./... -race -count=1`
- [ ] 4.4: Open PR titled `feat(evolution): EvalSignal types + post-run hook (#1606)` referencing Epic #1565 Phase 3
