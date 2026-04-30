// Package evolution implements the Phase 3.3 trigger surface that decides
// when a pipeline accumulates enough degraded eval signal to merit a
// pipeline-evolve run.
//
// The package owns Service.ShouldEvolve(pipelineName), which scores rows from
// state.EvolutionStore.GetEvalsForPipeline against three configurable
// heuristics (any-of):
//
//   - every-N: ≥every_n_window scored evals since last proposal AND median
//     judge_score has dropped ≥every_n_judge_drop versus the prior window.
//   - drift: contract_pass rate dropped >drift_pass_drop over the most
//     recent drift_window evals.
//   - retry-rate: avg retry_count >retry_avg_threshold over the most
//     recent retry_window evals.
//
// Defaults are compiled-in and overrideable via the top-level evolution:
// block on wave.yaml. Callers (the executor) feed the trigger advisory
// emission in recordPipelineEval; firing emits an event with state
// "evolution_proposed" but never blocks the run.
//
// Phase 3.3 of the evolution-loop epic (#1565); issue #1612.
package evolution
