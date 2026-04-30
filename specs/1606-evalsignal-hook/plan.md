# Implementation Plan: EvalSignal types + post-run hook

## 1. Objective

Add structured `EvalSignal` contract types and wire a post-run hook in `DefaultPipelineExecutor` that aggregates per-step + per-run signals and persists them via `state.EvolutionStore.RecordEval`. This closes Phase 3.1 of the evolution-loop epic (#1565), producing the `pipeline_eval` data feed used downstream by the judge / evolution proposer.

## 2. Approach

- Define a small, contract-package-local domain: `SignalKind` enum + `Signal` struct + `SignalSet` aggregator. Contract package is the right home — the same kinds (contract_failure, judge_score) are produced by validators there.
- Add `internal/pipeline/executor_eval.go` housing:
  - `evalCollector` that lives on the executor (or per-execution) and accumulates step-level `Signal`s.
  - `recordStepEval(step, result, err)` — emits step-level signals (status, duration, contract pass/fail) into the collector. Called from the existing step-completion path in `executor_steps.go`.
  - `recordPipelineEval(execution)` — terminal aggregator that reads the collector, computes pipeline-level fields (failure_class, retry_count, duration_ms, cost_dollars, judge_score average if any) and calls `evolutionStore.RecordEval`. Called from `finalizePipelineExecution` after state save, before `runTerminalHooks`.
- Hook is non-fatal: `RecordEval` errors are logged via the audit logger (or `e.emit` info event) but never bubble up.
- Wired through executor only when `e.store` satisfies `state.EvolutionStore` (state-store concrete type already does — same handle).

## 3. File Mapping

| Action | Path | Purpose |
|---|---|---|
| **Create** | `internal/contract/eval_signal.go` | `SignalKind` enum constants, `Signal` struct, `SignalSet` aggregator with `Add`, `FailureClass`, `Aggregate` helpers |
| **Create** | `internal/contract/eval_signal_test.go` | Unit tests for `SignalKind` round-trip, `SignalSet.Aggregate` (counts retries, picks dominant failure class, averages judge score) |
| **Create** | `internal/pipeline/executor_eval.go` | `recordStepEval`, `recordPipelineEval`, `mapStateToSignal` helpers; `WithEvolutionStore` option (optional override; default uses `e.store`) |
| **Create** | `internal/pipeline/executor_eval_test.go` | Synthetic pipeline run with mock adapter + temp-file SQLite state store; assert `pipeline_eval` rows |
| **Modify** | `internal/pipeline/executor.go` | Add `evalCollector` field on `DefaultPipelineExecutor`; init in `NewDefaultPipelineExecutor` |
| **Modify** | `internal/pipeline/executor_steps.go` | Call `e.recordStepEval(...)` at terminal step states (success, completed_empty, failed) — single call site after state mutation |
| **Modify** | `internal/pipeline/executor_lifecycle.go` | Call `e.recordPipelineEval(execution)` from `finalizePipelineExecution` after `SavePipelineState`, before `runTerminalHooks` |

No new package boundaries. No DB migration (table exists per PRE-5).

## 4. Architecture Decisions

- **Package home for `Signal`/`SignalKind` = `internal/contract`** — the only contract-side producers of judge/contract failures already live there; placing types alongside avoids an import cycle with `internal/pipeline` and gives evolution consumers a stable contract-shaped API.
- **Aggregator lives on executor, not on each step** — pipeline-level `RecordEval` needs cross-step aggregation (retry counts, failure-class voting, total duration). A per-execution map keyed by `runID` is simplest; cleaned up in `cleanupCompletedPipeline`.
- **EvolutionStore is interface-typed** — `recordPipelineEval` takes `state.EvolutionStore`, allowing test injection without a real DB and making future extraction trivial.
- **FailureClass derivation** — priority: `contract_failure` > `failure` > empty (success). Encoded in `SignalSet.FailureClass()`.
- **Cost / duration** — `DurationMs` from `time.Since(execution.Status.StartedAt)`. `CostDollars` from `e.costLedger.RunTotal(runID)` if non-nil; else nil pointer.
- **JudgeScore** — averaged across step-level `judge_score` signals; nil when no judge ran.
- **Skip/cancel semantics** — `stateSkipped` steps produce no signal (matches issue's open question; cleanest semantics: only ran steps emit).

## 5. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Hook double-fires on resume → duplicate `pipeline_eval` rows | `(pipeline_name, run_id)` is PK; INSERT will fail second time. Wrap call to log+swallow `UNIQUE constraint` errors specifically |
| Cost ledger may be nil in unit tests | Guard `e.costLedger != nil` before reading |
| `state.StateStore` may not implement `EvolutionStore` (tests with stubs) | Type-assert; skip recording silently if assertion fails (already standard pattern: `e.store != nil` checks) |
| Adding a field to `DefaultPipelineExecutor` ripples to all `New...` call sites | None — struct fields can be zero-valued; no constructor signature change |
| Contract package gets a runtime-state shape it didn't have before | Keep the file slim and contract-shaped (kinds, struct, aggregator). No DB imports |

## 6. Testing Strategy

- **Unit (`eval_signal_test.go`)** — round-trip `SignalKind` strings; `SignalSet.Aggregate` priority order; judge score averaging; empty set returns zero record.
- **Unit (`executor_eval_test.go`)** —
  - Synthetic 2-step pipeline (one success, one failure) with `MockAdapter` and temp-file SQLite via `state.NewSQLiteStore(t.TempDir()+"/state.db")`.
  - Run executor; assert `state.GetEvalsForPipeline(name, 0)` returns one row with `FailureClass="failure"`, `RetryCount` matching, `DurationMs > 0`.
  - Second test: all-success pipeline → `ContractPass=true`, `FailureClass=""`.
  - Third test: stub `EvolutionStore` returning error → executor still completes (failure non-fatal).
- **Race** — `go test -race ./internal/pipeline/...` (executor mutates collector under existing `execution.mu`).
- **Contract test** — none; this PR adds infrastructure read by future evolution code.
