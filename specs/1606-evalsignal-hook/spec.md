# Phase 3.1: EvalSignal types + post-run hook

**Issue:** [re-cinq/wave#1606](https://github.com/re-cinq/wave/issues/1606)
**Labels:** enhancement, ready-for-impl
**State:** OPEN
**Author:** nextlevelshit

## Body

Part of Epic #1565 Phase 3 (evolution loop).

### Goal

Define `EvalSignal` types and add a post-run hook in the executor that emits signals to the `pipeline_eval` table (PRE-5).

### Acceptance criteria

- [ ] `internal/contract/eval_signal.go` — types: SignalKind enum (success/failure/contract_failure/judge_score/duration/cost), Signal struct
- [ ] `internal/pipeline/executor_eval.go` — post-run hook called from executor on each step + final pipeline result
- [ ] Persists to `state.EvolutionStore.RecordEval(...)`
- [ ] Test: synthetic pipeline run → verify rows in pipeline_eval

### Dependencies

- PRE-5 EvolutionStore + pipeline_eval table (MERGED)

## Context (from assessment)

- Quality score: 88
- Complexity: medium
- Skipped speckit steps: specify, clarify
- Branch: `1606-evalsignal-hook`

### Open questions (resolved by inference)

1. **Signal struct shape** — derived from `state.PipelineEvalRecord`:
   `PipelineName, RunID, JudgeScore (*float64), ContractPass (*bool), RetryCount (*int), FailureClass (string), HumanOverride (*bool), DurationMs (*int64), CostDollars (*float64), RecordedAt (time.Time)`.
2. **Hook on skipped/cancelled steps** — emit step-level signals only for terminal step states (`stateCompleted`, `stateCompletedEmpty`, `stateFailed`); skip steps in `stateSkipped` produce no signal. Pipeline-level signal always fires from `finalizePipelineExecution`.
